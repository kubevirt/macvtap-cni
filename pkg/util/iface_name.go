package util

import (
	"fmt"

	"github.com/cespare/xxhash/v2"
)

// TemporaryInterfaceName returns a deterministic interface name that fits
// within the Linux IFNAMSIZ limit (15 chars).
func TemporaryInterfaceName(deviceID string) string {
	h := xxhash.Sum64String(deviceID)
	return fmt.Sprintf("mvt%012x", h&0xffffffffffff)
}
