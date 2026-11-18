// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0

package minlz

import (
	"bytes"
	"testing"
)

func TestFastestReservationPaths(t *testing.T) {
	for n := 0; n <= 1100; n++ {
		for kind := 0; kind < 2; kind++ {
			src := make([]byte, n)
			x := uint64(n + 1)
			for i := range src {
				x ^= x << 13
				x ^= x >> 7
				x ^= x << 17
				src[i] = byte(x)
				if kind == 1 {
					src[i] = byte(i % 19)
				}
			}
			original := bytes.Clone(src)
			want, e := Encode(nil, src, LevelFastest)
			if e != nil {
				t.Fatal(e)
			}
			for _, capacity := range []int{0, max(0, MaxEncodedLen(n)-1), MaxEncodedLen(n), MaxEncodedLen(n) + 31} {
				backing := bytes.Repeat([]byte{0xad}, 64+capacity+64)
				dst := backing[64 : 64 : 64+capacity]
				got, e := Encode(dst, src, LevelFastest)
				if e != nil || !bytes.Equal(got, want) {
					t.Fatalf("length=%d kind=%d capacity=%d: %v", n, kind, capacity, e)
				}
				if !bytes.Equal(src, original) || !bytes.Equal(backing[:64], bytes.Repeat([]byte{0xad}, 64)) || !bytes.Equal(backing[64+capacity:], bytes.Repeat([]byte{0xad}, 64)) {
					t.Fatalf("guard length=%d kind=%d capacity=%d", n, kind, capacity)
				}
			}
		}
	}
}
