package util

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/vishvananda/netlink"

	"github.com/containernetworking/cni/pkg/types/current"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
)

const (
	// IPv4InterfaceArpProxySysctlTemplate allows proxy_arp on a given interface
	IPv4InterfaceArpProxySysctlTemplate = "net.ipv4.conf.%s.proxy_arp"
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

func CreateMacvtap(name string, master string, mode string) (int, error) {
	ifindex := 0

	m, err := netlink.LinkByName(master)
	if err != nil {
		return ifindex, fmt.Errorf("failed to lookup master %q: %v", master, err)
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

func RecreateMacvtap(name string, master string, mode string) (int, error) {
	err := LinkDelete(name)
	if err != nil {
		return 0, err
	}
	return CreateMacvtap(name, master, mode)
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

// Listen for events on a specific interface and callback if any. The interface
// does not have to exist. Use the stop channel to stop listening.
func OnLinkEvent(name string, do func(), stop <-chan struct{}, errcb func(error)) {
	done := make(chan struct{})
	defer close(done)

	options := netlink.LinkSubscribeOptions{
		ListExisting: true,
		ErrorCallback: func(err error) {
			errcb(fmt.Errorf("Error while listening on link events: %v", err))
		},
	}

	subscribed := false
	var netlinkCh chan netlink.LinkUpdate
	subscribe := func() {
		netlinkCh = make(chan netlink.LinkUpdate)
		err := netlink.LinkSubscribeWithOptions(netlinkCh, done, options)
		if err != nil {
			errcb(fmt.Errorf("Error while subscribing for link events: %v", err))
			return
		}
		subscribed = true
	}

	subscribe()

	for {
		if !subscribed {
			select {
			case <-time.After(10 * time.Second):
				subscribe()
				continue
			case <-stop:
				break
			}
		}

		var update netlink.LinkUpdate
		select {
		case update, subscribed = <-netlinkCh:
			if subscribed {
				if name == update.Link.Attrs().Name {
					do()
				}
			}
		case <-stop:
			break
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

		// set proxy_arp on the interface
		if err := configureArp(newIfaceName); err != nil {
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

func configureArp(ifaceName string) error {
	// For sysctl, dots are replaced with forward slashes
	ifaceNameAllowingDots := strings.Replace(ifaceName, ".", "/", -1)

	// TODO: duplicate following lines for ipv6 support, when it will be added in other places
	ipv4SysctlValueName := fmt.Sprintf(IPv4InterfaceArpProxySysctlTemplate, ifaceNameAllowingDots)
	_, err := sysctl.Sysctl(ipv4SysctlValueName, "1")
	if err != nil {
		// the link will be removed in the CmdAdd deferred cleanup action
		return fmt.Errorf("failed to set proxy_arp on newly added interface %q: %v", ifaceName, err)
	}

	return nil
}
