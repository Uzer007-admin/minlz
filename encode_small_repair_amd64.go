// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build !appengine && !noasm && gc && !purego

package minlz

import "github.com/Uzer007-admin/minlz/internal/race"

// encodeSmallRepair preserves the canonical Tiny parser and output.
// Encode has already bounded the source to 22..1024 and reserved len(src)+2.
func smallUseWhole(n int) bool { return (prefixSSEHardware || n < 128 || n > 256) && n < 512 }

func encodeSmallCanonicalTiny(dst, src []byte) int {
	n := len(src)
	// The reservation and the complete 1K kernel have the same wire
	// contract as Tiny's 1K body with the Go header. The compile-time v2
	// choice does not require AVX, and v1 keeps its existing path.
	if prefixSSEHardware && uint(n-512) <= 512 {
		race.ReadSlice(src)
		race.WriteSlice(dst)
		return encodeSmallWhole1024Asm(dst, src)
	}
	dst[0] = 0
	header := 2
	if n < 128 {
		dst[1] = byte(n)
	} else {
		dst[1] = byte(n) | 0x80
		dst[2] = byte(n >> 7)
		header = 3
	}
	encoded := encodeBlockTiny(dst[header:], src)
	if encoded > 0 {
		return header + encoded
	}
	dst[1] = 0
	copy(dst[2:], src)
	return n + 2
}
