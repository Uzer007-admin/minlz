//go:build amd64 && !appengine && !noasm && gc && !purego

package minlz

import (
	"os"
	"testing"
)

// Test-only switches select both paths before any tests or parallel workers
// start. Production binaries have no environment-controlled CPU dispatch.
func init() {
	if os.Getenv("MINLZ_TEST_SSE2") == "1" {
		hasAVX2 = false
	}
	switch os.Getenv("MINLZ_TEST_STORED_COPY") {
	case "cached":
		storedCopyMode = 1
	case "runtime":
		storedCopyMode = 0
	}
}

func TestEncoderHardwareDispatch(t *testing.T) {
	core := uint8(0)
	if storedCopyMode == 2 {
		core = cpuidStoreCoreType()
	}
	t.Logf("AVX2=%v stored-copy-mode=%d current-core-type=0x%02x cached=%v", hasAVX2, storedCopyMode, core, preferCachedStore())
}
