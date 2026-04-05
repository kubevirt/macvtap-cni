package util_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/macvtap-cni/pkg/util"
)

var _ = Describe("TemporaryInterfaceName", func() {
	It("returns a deterministic name within IFNAMSIZ", func() {
		deviceID := "a-very-long-device-id-that-would-overflow-ifname"

		ifaceName := util.TemporaryInterfaceName(deviceID)
		ifaceName2 := util.TemporaryInterfaceName(deviceID)

		Expect(ifaceName).To(Equal(ifaceName2))
		Expect(ifaceName).To(HavePrefix("mvt"))
		Expect(len(ifaceName)).To(BeNumerically("<=", 15))
	})
})
