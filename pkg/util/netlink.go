package util

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/containernetworking/cni/pkg/types/current"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
)

func ModeFromString(s string) (netlink.MacvlanMode, error) {
	switch s {
	case "", "bridge":
		return netlink.MACVLAN_MODE_BRIDGE, nil
	case "private":
		return netlink.MACVLAN_MODE_PRIVATE, nil
	case "vepa":
		return netlink.MACVLAN_MODE_VEPA, nil
	default:
		return 0, fmt.Errorf("unknown macvtap mode: %q", s)
	}
}

func CreateMacvtap(name string, lowerDevice string, mode string) (int, error) {
	ifindex := 0

	m, err := netlink.LinkByName(lowerDevice)
	if err != nil {
		return ifindex, fmt.Errorf("failed to lookup lowerDevice %q: %v", lowerDevice, err)
	}

	nlmode, err := ModeFromString(mode)
	if err != nil {
		return ifindex, err
	}

	mv := &netlink.Macvtap{
		Macvlan: netlink.Macvlan{
			LinkAttrs: netlink.LinkAttrs{
				Name:        name,
				ParentIndex: m.Attrs().Index,
				// we had crashes if we did not set txqlen to some value
				TxQLen: m.Attrs().TxQLen,
			},
			Mode: nlmode,
		},
	}

	if err := netlink.LinkAdd(mv); err != nil {
		return ifindex, fmt.Errorf("failed to create macvtap: %v", err)
	}

	if err := netlink.LinkSetUp(mv); err != nil {
		return ifindex, fmt.Errorf("failed to set %q UP: %v", name, err)
	}

	ifindex = mv.Attrs().Index
	return ifindex, nil
}

func RecreateMacvtap(name string, lowerDevice string, mode string) (int, error) {
	err := LinkDelete(name)
	if err != nil {
		return 0, err
	}
	return CreateMacvtap(name, lowerDevice, mode)
}

func LinkExists(link string) (bool, error) {
	_, err := netlink.LinkByName(link)
	if _, ok := err.(netlink.LinkNotFoundError); ok {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func LinkDelete(link string) error {
	l, err := netlink.LinkByName(link)
	if _, ok := err.(netlink.LinkNotFoundError); ok {
		return nil
	}
	if err != nil {
		return err
	}
	err = netlink.LinkDel(l)
	return err
}

func isLoopback(link netlink.Link) bool {
	return link.Attrs().Flags&net.FlagLoopback != 0
}

func isSuitableMacvtapParent(link netlink.Link) bool {
	if isLoopback(link) {
		return false
	}

	switch link.(type) {
	case *netlink.Bond, *netlink.Device:
	default:
		return false
	}

	return true
}

// FindSuitableMacvtapParents lists all the links on the system and filters out
// those deemed inappropriate to be used as macvtap parents.
func FindSuitableMacvtapParents() ([]string, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, err
	}

	linkNames := make([]string, 0)
	for _, link := range links {
		if isSuitableMacvtapParent(link) {
			linkNames = append(linkNames, link.Attrs().Name)
		}
	}

	return linkNames, nil
}

// OnLinkEvent listens for events on a specific interface and namespace, and
// callbacks if any. See onLinkEvent for more details.
func OnLinkEvent(name string, nsPath string, do func(), stop <-chan struct{}, errcb func(error)) {
	matcher := func(link netlink.Link) bool {
		return name == link.Attrs().Name
	}

	onLinkEvent(matcher, nsPath, do, stop, errcb)
}

// OnSuitableMacvtapParentEvent listens for events on any suitable macvtap
// parent link on a given namespace and callbacks if any. See onLinkEvent
// for more details.
func OnSuitableMacvtapParentEvent(nsPath string, do func(), stop <-chan struct{}, errcb func(error)) {
	onLinkEvent(isSuitableMacvtapParent, nsPath, do, stop, errcb)
}

// onLinkEvent upkeeps a subscription to netlink events and callbacks for any
// that matches the predicate on the related link.
// The subscription might temporarily fail. On re-subscription, the callback is
// invoked to cover for events that might have been missed during that time.
// That means some spurious callbacks unrelated to the predicate might happen
// and the caller should account for it. For convenience, to avoid losing any
// relevant information between the time of this function call (or a previous
// time when the caller initializes state) and the time the subscription is
// effective, the callback is also invoked upon first subscription. As a
// summary, callback is invoked:
//
// * A first time, after first subscription
// * Once every re-subscription
// * On any event matching the predicate
//
func onLinkEvent(match func(netlink.Link) bool, nsPath string, do func(), stop <-chan struct{}, errcb func(error)) {
	done := make(chan struct{})
	defer close(done)

	options := netlink.LinkSubscribeOptions{
		ListExisting: false,
		ErrorCallback: func(err error) {
			errcb(fmt.Errorf("Error while listening on link events: %v", err))
		},
	}

	subscribed := false
	var netlinkCh chan netlink.LinkUpdate
	subscribe := func() {
		ns, err := netns.GetFromPath(nsPath)
		if err != nil {
			errcb(fmt.Errorf("Could not open namespace: %v", err))
			return
		}
		defer ns.Close()

		options.Namespace = &ns
		netlinkCh = make(chan netlink.LinkUpdate)
		err = netlink.LinkSubscribeWithOptions(netlinkCh, done, options)
		if err != nil {
			errcb(fmt.Errorf("Error while subscribing for link events: %v", err))
			return
		}
		subscribed = true

		// Callback on every subscription
		do()
	}

	subscribe()

	for {
		if !subscribed {
			select {
			case <-time.After(10 * time.Second):
				subscribe()
				continue
			case <-stop:
				return
			}
		}

		var update netlink.LinkUpdate
		select {
		case update, subscribed = <-netlinkCh:
			if subscribed {
				if match(update.Link) {
					do()
				}
			}
		case <-stop:
			return
		}
	}
}

// Move an existing macvtap interface from the current netns to the target netns, and rename it..
// Optionally configure the MAC address of the interface and the link's MTU.
func ConfigureInterface(currentIfaceName string, newIfaceName string, macAddr *net.HardwareAddr, mtu int, netns ns.NetNS) (*current.Interface, error) {
	var err error

	macvtapIface, err := netlink.LinkByName(currentIfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup device %q: %v", currentIfaceName, err)
	}

	// move the macvtap interface to the pod's netns
	if err = netlink.LinkSetNsFd(macvtapIface, int(netns.Fd())); err != nil {
		return nil, fmt.Errorf("failed to move iface %s to the netns %d because: %v", macvtapIface, netns.Fd(), err)
	}

	var macvtap *current.Interface = nil

	// configure the macvtap iface
	err = netns.Do(func(_ ns.NetNS) error {
		defer func() {
			if err != nil {
				LinkDelete(currentIfaceName)
				LinkDelete(newIfaceName)
			}
		}()

		if mtu != 0 {
			if err := netlink.LinkSetMTU(macvtapIface, mtu); err != nil {
				return fmt.Errorf("failed to set the macvtap MTU for %s: %v", currentIfaceName, err)
			}
		}

		if macAddr != nil {
			if err := netlink.LinkSetHardwareAddr(macvtapIface, *macAddr); err != nil {
				return fmt.Errorf("failed to add hardware addr to %q: %v", currentIfaceName, err)
			}
		}

		renamedMacvtapIface, err := renameInterface(macvtapIface, newIfaceName)
		if err != nil {
			return err
		}

		if err := netlink.LinkSetUp(renamedMacvtapIface); err != nil {
			return fmt.Errorf("failed to set macvtap iface up: %v", err)
		}

		// Re-fetch macvtap to get all properties/attributes
		// This enables us to report back the MAC address assigned to the macvtap iface
		// and now that we've handed the macvtap over, update the netns where it runs
		macvtapIface, err = netlink.LinkByName(newIfaceName)
		if err != nil {
			return err
		}

		macvtap = &current.Interface{
			Name:    newIfaceName,
			Mac:     macvtapIface.Attrs().HardwareAddr.String(),
			Sandbox: netns.Path(),
		}

		return nil
	})

	return macvtap, err
}

func renameInterface(currentIface netlink.Link, newIfaceName string) (netlink.Link, error) {
	currentIfaceName := currentIface.Attrs().Name
	if err := ip.RenameLink(currentIfaceName, newIfaceName); err != nil {
		return nil, fmt.Errorf("failed to rename macvlan to %q: %v", newIfaceName, err)
	}

	renamedMacvtapIface := currentIface
	renamedMacvtapIface.Attrs().Name = newIfaceName

	return renamedMacvtapIface, nil
}

// GetMainThreadNetNsPath returns the path of the main thread's namespace
func GetMainThreadNetNsPath() string {
	return fmt.Sprintf("/proc/%d/ns/net", os.Getpid())
}
