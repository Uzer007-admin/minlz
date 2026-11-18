// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0

package minlz

import (
	"bytes"
	"math/rand"
	"testing"
)

// 4097 used to enter the pooled size class. The replacement must preserve
// every match decision through the new stack/pooled boundary at 16384/16385.
func TestFastestStack16MatchesPooled(t *testing.T) {
	rng := rand.New(rand.NewSource(409716384))
	for _, n := range []int{4095, 4096, 4097, 4098, 4128, 4608, 6144, 8191, 8192, 12288, 16383, 16384, 16385} {
		for shape := 0; shape < 4; shape++ {
			for _, alignment := range []int{0, 1, 7, 31, 63} {
				raw := make([]byte, n+alignment)
				src := raw[alignment:]
				rng.Read(src)
				switch shape {
				case 1, 2:
					period := 1 + (shape-1)*250
					for i := period; i < n; i++ {
						src[i] = src[i%period]
					}
				case 3:
					for i := 71; i+8 < n; i += 127 {
						copy(src[i:i+8], src[i-63:i-55])
					}
				}
				original := bytes.Clone(src)
				want, compressed := pooledSmallReference(src)
				storage := bytes.Repeat([]byte{0xa5}, MaxEncodedLen(n)+128)
				dst := storage[64 : len(storage)-64 : len(storage)-64]
				got, err := Encode(dst, src, LevelFastest)
				if err != nil || !bytes.Equal(got, want) {
					t.Fatalf("Encode n=%d shape=%d alignment=%d: %v", n, shape, alignment, err)
				}
				got = TryEncode(dst, src, LevelFastest)
				if compressed && !bytes.Equal(got, want) || !compressed && got != nil {
					t.Fatalf("TryEncode n=%d shape=%d alignment=%d", n, shape, alignment)
				}
				if !bytes.Equal(src, original) ||
					!bytes.Equal(storage[:64], bytes.Repeat([]byte{0xa5}, 64)) ||
					!bytes.Equal(storage[len(storage)-64:], bytes.Repeat([]byte{0xa5}, 64)) {
					t.Fatalf("buffer guard n=%d shape=%d alignment=%d", n, shape, alignment)
				}
			}
		}
	}
}
