package deviceplugin_test

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"strings"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/testutils"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"github.com/vishvananda/netlink"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	. "github.com/kubevirt/macvtap-cni/pkg/deviceplugin"
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

var _ = Describe("Macvtap device plugin", func() {
	var cleanup func()
	var mvdp dpm.PluginInterface
	var masterIfaceName string
	var masterIface netlink.Link
	var sendSpy *ListAndWatchServerSendSpy

	BeforeEach(func() {
		currNs, err := ns.GetCurrentNS()
		Expect(err).NotTo(HaveOccurred())

		testNs, err := testutils.NewNS()
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

		mvdp = NewMacvtapDevicePlugin(masterIfaceName, masterIfaceName, "bridge", 0)

		sendSpy = &ListAndWatchServerSendSpy{}
		go func() {
			err := testNs.Do(func(ns ns.NetNS) error {
				return mvdp.ListAndWatch(nil, sendSpy)
			})
			Expect(err).NotTo(HaveOccurred())
		}()

		runtime.LockOSThread()
		err = testNs.Set()
		Expect(err).NotTo(HaveOccurred())

		cleanup = func() {
			mvdp.(dpm.PluginInterfaceStop).Stop()
			netlink.LinkDel(masterIface)
			currNs.Set()
		}
	})

	AfterEach(func() {
		cleanup()
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

		iface, err := netlink.LinkByName(ifaceName)
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
				err := util.LinkDelete(masterIfaceName)
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
