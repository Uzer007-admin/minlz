//go:build !appengine && !noasm && gc && !purego

package minlz

func prefixHardware(src []byte) bool        { return false }
func encodePrefixBlock(dst, src []byte) int { panic("unreachable prefix kernel") }
