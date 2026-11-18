// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0

package minlz

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"
)

// The specialized kernels must make the same decisions as the pooled encoder,
// including collisions, stored-block fallback, and both position-width limits.
func TestFastestSmallBlocksMatchPooled(t *testing.T) {
	rng := rand.New(rand.NewSource(87123))
	for n := 17; n <= 4097; n++ {
		for shape := 0; shape < 4; shape++ {
			for _, offset := range []int{0, 1, 7, 31} {
				source := make([]byte, n+offset)
				src := source[offset:]
				_, _ = rng.Read(src)
				if shape < 3 {
					period := 1 + shape*31
					for i := period; i < n; i++ {
						src[i] = src[i%period]
					}
				}
				want := make([]byte, MaxEncodedLen(n))
				d := 1 + binary.PutUvarint(want[1:], uint64(n))
				if m := encodeBlock(want[d:], src); m > 0 {
					want = want[:d+m]
				} else {
					want = encodeUncompressed(want[:0], src)
				}
				storage := bytes.Repeat([]byte{0xa5}, MaxEncodedLen(n)+64)
				dst := storage[32 : len(storage)-32 : len(storage)-32]
				got, err := Encode(dst, src, LevelFastest)
				if err != nil || !bytes.Equal(got, want) {
					t.Fatalf("size=%d shape=%d alignment=%d: output differs: %v", n, shape, offset, err)
				}
				if !bytes.Equal(storage[:32], bytes.Repeat([]byte{0xa5}, 32)) ||
					!bytes.Equal(storage[len(storage)-32:], bytes.Repeat([]byte{0xa5}, 32)) {
					t.Fatalf("size=%d shape=%d alignment=%d: destination overrun", n, shape, offset)
				}
				decoded, err := Decode(nil, got)
				if err != nil || !bytes.Equal(decoded, src) {
					t.Fatalf("size=%d shape=%d alignment=%d: decode differs: %v", n, shape, offset, err)
				}
			}
		}
	}
}
