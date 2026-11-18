// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0

package minlz

import (
	"bytes"
	"testing"
)

func TestEncodeTinyUnaligned(t *testing.T) {
	for n := 0; n <= 21; n++ {
		for sa := 0; sa < 32; sa++ {
			for da := 0; da < 32; da++ {
				source := bytes.Repeat([]byte{0x59}, 64+sa+n+64)
				start := 64 + sa
				src := source[start : start+n : start+n]
				for i := range src {
					src[i] = byte(i*31 + 17)
				}
				untouched := append([]byte(nil), source...)
				want := append([]byte{0, 0}, src...)
				if n == 0 {
					want = []byte{0}
				}
				output := bytes.Repeat([]byte{0xa7}, 64+da+len(want)+64)
				at := 64 + da
				got, e := Encode(output[at:at:at+len(want)], src, LevelFastest)
				if e != nil || !bytes.Equal(got, want) {
					t.Fatalf("n=%d src-align=%d dst-align=%d: %v", n, sa, da, e)
				}
				if !bytes.Equal(source, untouched) || !bytes.Equal(output[:at], bytes.Repeat([]byte{0xa7}, at)) || !bytes.Equal(output[at+len(want):], bytes.Repeat([]byte{0xa7}, 64)) {
					t.Fatalf("guard n=%d src-align=%d dst-align=%d", n, sa, da)
				}
			}
		}
	}
}
