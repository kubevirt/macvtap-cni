package cni_test

import (
	"fmt"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/testutils"
	"github.com/kubevirt/macvtap-cni/pkg/cni"
	"github.com/kubevirt/macvtap-cni/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
)

const LOWER_DEVICE = "eth0"

var _ = Describe("Macvtap CNI", func() {
	Context("with configuration set", func() {
		var originalNS ns.NetNS
		var targetNs ns.NetNS
		var lowerDevice netlink.Link
		var macvtapInterface netlink.Link
		var stdInArgs string

		deviceID := "dev500"
		macvtapIfaceName := "macvtap0"

		BeforeEach(func() {
			var err error
			originalNS, err = testutils.NewNS()
			Expect(err).NotTo(HaveOccurred())

			targetNs, err = testutils.NewNS()
			Expect(err).NotTo(HaveOccurred())

			err = originalNS.Do(func(ns.NetNS) error {
				defer GinkgoRecover()

				// create lower device
				err = netlink.LinkAdd(&netlink.Dummy{
					LinkAttrs: netlink.LinkAttrs{
						Name: LOWER_DEVICE,
					},
				})
				Expect(err).NotTo(HaveOccurred())

				lowerDevice, err = netlink.LinkByName(LOWER_DEVICE)
				Expect(err).NotTo(HaveOccurred())

				// create macvtap on top of lower device
				_, err = util.CreateMacvtap(deviceID, LOWER_DEVICE, "bridge")
				Expect(err).NotTo(HaveOccurred())

				// cache the macvtap interface
				macvtapInterface, err = netlink.LinkByName(deviceID)
				Expect(err).NotTo(HaveOccurred())

				stdInArgs = fmt.Sprintf(`{
				"cniVersion": "0.3.1",
				"name": "mynet",
				"type": "macvtap",
				"deviceID": "%s"
			}`, deviceID)

				return nil
			})

			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			// cleanup the original namespace
			originalNS.Do(func(ns.NetNS) error {
				return netlink.LinkDel(lowerDevice)
			})
			Expect(originalNS.Close()).To(Succeed())
			Expect(testutils.UnmountNS(originalNS)).To(Succeed())

			// cleanup the target namespace
			targetNs.Do(func(ns.NetNS) error {
				return netlink.LinkDel(macvtapInterface)
			})
			Expect(targetNs.Close()).To(Succeed())
			Expect(testutils.UnmountNS(targetNs)).To(Succeed())
		})

		Context("WHEN importing a macvtap interface into the target netns without further configuration", func() {
			var args *skel.CmdArgs

			BeforeEach(func() {
				args = &skel.CmdArgs{
					ContainerID: "dummy",
					Netns:       targetNs.Path(),
					IfName:      macvtapIfaceName,
					StdinData:   []byte(stdInArgs),
				}

				originalNS.Do(func(ns.NetNS) error {
					defer GinkgoRecover()

					_, _, err := testutils.CmdAdd(args.Netns, args.ContainerID, args.IfName, args.StdinData, func() error { return cni.CmdAdd(args) })
					Expect(err).NotTo(HaveOccurred())

					return nil
				})
			})

			It("SHOULD successfully import the macvtap interface into the target netns, using an auto-generated MAC address", func() {
				// confirm macvtap is available on target namespace
				targetNs.Do(func(ns.NetNS) error {
					defer GinkgoRecover()

					link, err := netlink.LinkByName(macvtapIfaceName)
					Expect(err).NotTo(HaveOccurred())

					Expect(link.Attrs().HardwareAddr.String()).NotTo(BeNil())

					return nil
				})
			})

			It("SHOULD successfully remove the macvtap interface, once requested via CmdDel", func() {
				targetNs.Do(func(ns.NetNS) error {
					defer GinkgoRecover()

					err := testutils.CmdDel(args.Netns, args.ContainerID, args.IfName, func() error { return cni.CmdDel(args) })
					Expect(err).NotTo(HaveOccurred())

					_, err = netlink.LinkByName(macvtapIfaceName)
					Expect(err).To(HaveOccurred())

					return nil
				})
			})
		})

		Context("WHEN importing a macvtap interface into the target netns with MAC address configuration", func() {
			const macAddress = "0a:59:00:dc:6a:e0"

			BeforeEach(func() {
				args := &skel.CmdArgs{
					ContainerID: "dummy",
					Netns:       targetNs.Path(),
					IfName:      macvtapIfaceName,
					StdinData:   []byte(stdInArgs),
					Args:        fmt.Sprintf("MAC=%s", macAddress),
				}

				originalNS.Do(func(ns.NetNS) error {
					defer GinkgoRecover()

					_, _, err := testutils.CmdAdd(args.Netns, args.ContainerID, args.IfName, args.StdinData, func() error { return cni.CmdAdd(args) })
					Expect(err).NotTo(HaveOccurred())
					return nil
				})
			})

			It("SHOULD successfully import the macvtap interface into the target netns, having the configured MAC address", func() {
				// confirm macvtap is available on target namespace, and the correct configurations were applied
				targetNs.Do(func(ns.NetNS) error {
					defer GinkgoRecover()

					link, err := netlink.LinkByName(macvtapIfaceName)
					Expect(err).NotTo(HaveOccurred())
					Expect(link.Attrs().HardwareAddr.String()).To(Equal(macAddress))

					return nil
				})
			})
		})

		Context("WHEN importing a macvtap interface into the target netns with link MTU configuration", func() {
			const mtu = 1000

			BeforeEach(func() {
				updatedMtuArgs := fmt.Sprintf(`{
				"cniVersion": "0.3.1",
				"name": "mynet",
				"type": "macvtap",
				"deviceID": "%s",
				"mtu": %d
			}`, deviceID, mtu)
				args := &skel.CmdArgs{
					ContainerID: "dummy",
					Netns:       targetNs.Path(),
					IfName:      deviceID,
					StdinData:   []byte(updatedMtuArgs),
				}

				originalNS.Do(func(ns.NetNS) error {
					defer GinkgoRecover()

					_, _, err := testutils.CmdAdd(args.Netns, args.ContainerID, args.IfName, args.StdinData, func() error { return cni.CmdAdd(args) })
					Expect(err).NotTo(HaveOccurred())

					return nil
				})
			})

			It("SHOULD successfully import the macvtap interface into the target netns, having the link MTU configured", func() {
				// confirm macvtap is available on target namespace, and the correct configurations were applied
				targetNs.Do(func(ns.NetNS) error {
					defer GinkgoRecover()

					link, err := netlink.LinkByName(deviceID)
					Expect(err).NotTo(HaveOccurred())
					Expect(link.Attrs().MTU).To(Equal(mtu))

					return nil
				})
			})
		})

		When("importing a macvtap interface into the target netns with promiscous mode enabled", func() {
			BeforeEach(func() {
				promiscousModeArgs := fmt.Sprintf(`{
				"cniVersion": "0.3.1",
				"name": "mynet",
				"type": "macvtap",
				"deviceID": "%s",
				"promiscMode": true
			}`, deviceID)
				args := &skel.CmdArgs{
					ContainerID: "dummy",
					Netns:       targetNs.Path(),
					IfName:      deviceID,
					StdinData:   []byte(promiscousModeArgs),
				}

				originalNS.Do(func(ns.NetNS) error {
					defer GinkgoRecover()

					_, _, err := testutils.CmdAdd(args.Netns, args.ContainerID, args.IfName, args.StdinData, func() error { return cni.CmdAdd(args) })
					Expect(err).NotTo(HaveOccurred())

					return nil
				})
			})

			It("SHOULD successfully import the macvtap interface into the target netns, having the link promisc mode enabled", func() {
				// confirm macvtap is available on target namespace, and the correct configurations were applied
				targetNs.Do(func(ns.NetNS) error {
					const enabled = 1
					defer GinkgoRecover()

					link, err := netlink.LinkByName(deviceID)
					Expect(err).NotTo(HaveOccurred())
					Expect(link.Attrs().Promisc).To(Equal(enabled))

					return nil
				})
			})
		})
	})
})
