package deviceplugin_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDevicePlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deviceplugin Suite")
}
