package deviceplugin

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/testutils"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"github.com/vishvananda/netlink"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/kubevirt/macvtap-cni/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type ListAndWatchServerSendSpy struct {
	calls int
	last  *pluginapi.ListAndWatchResponse
}

// Records that update has been received and fails or not depending on the fake server configuration.
func (s *ListAndWatchServerSendSpy) Send(resp *pluginapi.ListAndWatchResponse) error {
	s.calls++
	s.last = resp
	return nil
}

// Mandatory to implement pluginapi.DevicePlugin_ListAndWatchServer
func (s *ListAndWatchServerSendSpy) Context() context.Context {
	return nil
}

func (s *ListAndWatchServerSendSpy) RecvMsg(m interface{}) error {
	return nil
}

func (s *ListAndWatchServerSendSpy) SendMsg(m interface{}) error {
	return nil
}

func (s *ListAndWatchServerSendSpy) SendHeader(m metadata.MD) error {
	return nil
}

func (s *ListAndWatchServerSendSpy) SetHeader(m metadata.MD) error {
	return nil
}

func (s *ListAndWatchServerSendSpy) SetTrailer(m metadata.MD) {
}

var _ = Describe("Macvtap", func() {
	var masterIfaceName string
	var masterIface netlink.Link
	var testNs ns.NetNS

	BeforeEach(func() {
		var err error
		testNs, err = testutils.NewNS()
		Expect(err).NotTo(HaveOccurred())

		masterIfaceName = fmt.Sprintf("master%d", rand.Intn(100))
		masterIface = &netlink.Dummy{
			LinkAttrs: netlink.LinkAttrs{
				Name:      masterIfaceName,
				Namespace: netlink.NsFd(int(testNs.Fd())),
			},
		}

		err = netlink.LinkAdd(masterIface)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		testNs.Do(func(ns ns.NetNS) error {
			netlink.LinkDel(masterIface)
			return nil
		})
	})

	Describe("plugin", func() {
		var mvdp dpm.PluginInterface
		var sendSpy *ListAndWatchServerSendSpy

		BeforeEach(func() {
			mvdp = NewMacvtapDevicePlugin(masterIfaceName, masterIfaceName, "bridge", 0, testNs.Path())
			sendSpy = &ListAndWatchServerSendSpy{}
			go func() {
				err := mvdp.ListAndWatch(nil, sendSpy)
				Expect(err).NotTo(HaveOccurred())
			}()
		})

		AfterEach(func() {
			mvdp.(dpm.PluginInterfaceStop).Stop()
		})

		It("should allocate a new device upon request", func() {
			ifaceName := masterIfaceName + "Mvp99"
			req := &pluginapi.AllocateRequest{
				ContainerRequests: []*pluginapi.ContainerAllocateRequest{
					{
						DevicesIDs: []string{
							ifaceName,
						},
					},
				},
			}

			res, err := mvdp.Allocate(nil, req)
			Expect(err).NotTo(HaveOccurred())

			var iface netlink.Link
			err = testNs.Do(func(ns ns.NetNS) error {
				var err error
				iface, err = netlink.LinkByName(ifaceName)
				return err
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(iface.Type()).To(Equal("macvtap"))

			dev := res.ContainerResponses[0].Devices[0]
			index := iface.Attrs().Index
			Expect(strings.HasSuffix(dev.ContainerPath, strconv.Itoa(index))).To(BeTrue())
			Expect(dev.HostPath).To(Equal(dev.ContainerPath))
		})

		Context("when master device does not exist", func() {
			It("should not advertise devices", func() {
				By("first advertising healthy devices", func() {
					Eventually(func() int {
						return sendSpy.calls
					}).Should(Equal(1))

					Expect(sendSpy.last.Devices).To(HaveLen(100))
				})

				By("then deleting the master device", func() {
					err := testNs.Do(func(ns ns.NetNS) error {
						return util.LinkDelete(masterIfaceName)
					})
					Expect(err).NotTo(HaveOccurred())
				})

				By("then no longer advertising devices", func() {
					Eventually(func() int {
						return sendSpy.calls
					}).Should(Equal(2))

					Expect(sendSpy.last.Devices).To(HaveLen(0))
				})
			})
		})
	})

	Describe("lister", func() {
		var lister dpm.ListerInterface
		var pluginListCh chan dpm.PluginNameList

		BeforeEach(func() {
			pluginListCh = make(chan dpm.PluginNameList)
			lister = NewMacvtapLister(testNs.Path())
		})

		JustBeforeEach(func() {
			go func() {
				lister.Discover(pluginListCh)
			}()
		})

		AfterEach(func() {
			close(pluginListCh)
		})

		Context("WHEN provided a non empty configuration", func() {
			resourceName := "dataplane"
			mode := "vepa"
			capacity := 30
			config := `[{"name":"%s","master":"%s","mode":"%s","capacity":%d}]`

			BeforeEach(func() {
				config = fmt.Sprintf(config, resourceName, masterIfaceName, mode, capacity)
				os.Setenv(ConfigEnvironmentVariable, config)
			})

			AfterEach(func() {
				os.Unsetenv(ConfigEnvironmentVariable)
			})

			It("SHOULD report the appropriate list of resources", func() {
				Eventually(pluginListCh).Should(Receive(ConsistOf(resourceName)))
				Consistently(pluginListCh).ShouldNot(Receive(Not(ConsistOf(resourceName))))

				plugin := lister.NewPlugin(resourceName)
				Expect(plugin.(*macvtapDevicePlugin).Name).To(Equal(resourceName))
				Expect(plugin.(*macvtapDevicePlugin).Master).To(Equal(masterIfaceName))
				Expect(plugin.(*macvtapDevicePlugin).Mode).To(Equal(mode))
				Expect(plugin.(*macvtapDevicePlugin).Capacity).To(Equal(capacity))
			})
		})

		Context("WHEN provided an empty configuration", func() {
			BeforeEach(func() {
				os.Setenv(ConfigEnvironmentVariable, "[]")
			})

			AfterEach(func() {
				os.Unsetenv(ConfigEnvironmentVariable)
			})

			It("SHOULD update the list of available resources", func() {
				const parentName = "bond0"

				By("initially reporting the appropriate list of resources", func() {
					Eventually(pluginListCh).Should(Receive(BeEmpty()))
					Consistently(pluginListCh).ShouldNot(Receive(Not(BeEmpty())))
				})

				By("adding a new resource when a suitable macvtap parent appears", func() {
					parent := netlink.NewLinkBond(
						netlink.LinkAttrs{
							Name:      parentName,
							Namespace: netlink.NsFd(int(testNs.Fd())),
						},
					)
					err := netlink.LinkAdd(parent)
					Expect(err).NotTo(HaveOccurred())

					Eventually(pluginListCh).Should(Receive(ConsistOf(parentName)))
					Consistently(pluginListCh).ShouldNot(Receive(Not(ConsistOf(parentName))))

					plugin := lister.NewPlugin(parentName)
					Expect(plugin.(*macvtapDevicePlugin).Name).To(Equal(parentName))
					Expect(plugin.(*macvtapDevicePlugin).Master).To(Equal(parentName))
					Expect(plugin.(*macvtapDevicePlugin).Mode).To(Equal(DefaultMode))
					Expect(plugin.(*macvtapDevicePlugin).Capacity).To(Equal(DefaultCapacity))
				})

				By("removing the resource when a suitable macvtap parent disappears", func() {
					err := testNs.Do(func(ns ns.NetNS) error {
						parent, err := netlink.LinkByName(parentName)
						if err == nil {
							err = netlink.LinkDel(parent)
						}
						return err
					})
					Expect(err).NotTo(HaveOccurred())

					Eventually(pluginListCh).Should(Receive(BeEmpty()))
					Consistently(pluginListCh).ShouldNot(Receive(Not(BeEmpty())))
				})
			})
		})
	})
})
