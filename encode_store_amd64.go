//go:build amd64 && !appengine && !noasm && gc && !purego

package minlz

// storedCopyMode is 0 for an unrecognized hybrid, 1 for a conventional CPU,
// and 2 when CPUID.1A can identify the current Intel hybrid core.
var storedCopyMode = cpuidStoredCopyMode()

//go:noescape
func cpuidStoredCopyMode() uint8

//go:noescape
func cpuidStoreCoreType() uint8

func preferCachedStore() bool {
	if storedCopyMode == 1 {
		return true
	}
	// Query the current core only for a >=1 MiB stored block. Goroutines can
	// migrate: this is a performance hint and never enables an ISA feature.
	return storedCopyMode == 2 && cpuidStoreCoreType() == 0x20
}
