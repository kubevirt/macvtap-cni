package util

import (
	"fmt"
	"time"

	"github.com/vishvananda/netlink"
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
