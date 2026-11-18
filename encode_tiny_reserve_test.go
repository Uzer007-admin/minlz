// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0

package minlz

import (
	"bytes"
	"testing"
)

// A stored block must reuse any sufficient capacity and respect the actual
// slice capacity, including an empty destination and a one-byte short buffer.
func TestEncodeTinyReservation(t *testing.T) {
	for n := 0; n <= 21; n++ {
		for _, level := range []int{-9, -1, 0, 1, 2, 3, 9} {
			if n >= minNonLiteralBlockSize && (!hasTinyAsm || level != LevelFastest) {
				continue
			}
			raw := make([]byte, n)
			for i := range raw {
				raw[i] = byte(i*31 + 17)
			}
			original := append([]byte(nil), raw...)
			want := append([]byte{0, 0}, raw...)
			if n == 0 {
				want = []byte{0}
			}
			for _, capacity := range []int{0, 1, max(0, len(want)-1), len(want), len(want) + 7} {
				backing := bytes.Repeat([]byte{0xa7}, 64+capacity+64)
				dst := backing[64 : 64 : 64+capacity]
				got, err := Encode(dst, raw, level)
				if err != nil || !bytes.Equal(got, want) {
					t.Fatalf("n=%d level=%d cap=%d got=%x err=%v", n, level, capacity, got, err)
				}
				if capacity >= len(want) && &got[0] != &backing[64] {
					t.Fatalf("sufficient capacity not reused: n=%d cap=%d", n, capacity)
				}
				if !bytes.Equal(raw, original) || !bytes.Equal(backing[:64], bytes.Repeat([]byte{0xa7}, 64)) || !bytes.Equal(backing[64+capacity:], bytes.Repeat([]byte{0xa7}, 64)) {
					t.Fatalf("source/destination guard n=%d cap=%d", n, capacity)
				}
				if capacity >= len(want) && !bytes.Equal(backing[64+len(want):64+capacity], bytes.Repeat([]byte{0xa7}, capacity-len(want))) {
					t.Fatalf("write beyond result n=%d", n)
				}
			}
		}
	}
	for _, n := range []int{16, 17, 21, 22} {
		for _, level := range []int{-9, 9} {
			if _, e := Encode(nil, make([]byte, n), level); e != ErrInvalidLevel {
				t.Fatalf("invalid level n=%d err=%v", n, e)
			}
		}
	}
}
