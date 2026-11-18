// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build amd64 && !appengine && !noasm && gc && !purego

package minlz

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"testing"
)

// The third probe must see a just-written entry if it shares a bucket with
// either earlier probe. The second write wins if all three buckets coincide.
func TestPrefixDictionaryCollisions(t *testing.T) {
	if !hasAVX2 && !prefixSSEHardware {
		t.Skip("prefix dictionary requires AVX2 or an SSE4.2 target")
	}
	words := [][]byte{{0x31, 0xc5, 0x31, 0xc5, 0x31, 0xc5, 0x31, 0xc5}, {0xc5, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31}, {0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0xc5}, {0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31, 0x31}}
	for _, n := range []int{1 << 20, 3 << 20} {
		for kind, word := range words {
			src := make([]byte, n)
			rand.New(rand.NewSource(int64(n + kind))).Read(src)
			for i := 71; i+8 < n; i += 127 {
				copy(src[i:i+8], src[i-63:i-55])
			}
			src[0] = 0x7f
			copy(src[1:9], word)
			encode := func(prefix bool) []byte {
				storage := bytes.Repeat([]byte{0xa9}, 64+MaxEncodedLen(n)+64)
				dst := storage[64 : len(storage)-64 : len(storage)-64]
				dst[0] = 0
				d := 1 + binary.PutUvarint(dst[1:], uint64(n))
				var size int
				if prefix {
					size = encodePrefixBlock(dst[d:], src)
				} else {
					tmp := new([131072]byte)
					if n <= 2<<20 {
						size = encodeBlockAsm2MB(dst[d:], src, tmp)
					} else {
						size = encodeBlockAsm(dst[d:], src, tmp)
					}
				}
				if size == 0 {
					dst = encodeUncompressed(dst[:0], src)
				} else {
					dst = dst[:d+size]
				}
				if !bytes.Equal(storage[:64], bytes.Repeat([]byte{0xa9}, 64)) || !bytes.Equal(storage[len(storage)-64:], bytes.Repeat([]byte{0xa9}, 64)) {
					t.Fatalf("n=%d kind=%d: capacity guard", n, kind)
				}
				return dst
			}
			want, got := encode(false), encode(true)
			if !bytes.Equal(got, want) {
				t.Fatalf("n=%d collision=%d: prefix and canonical streams differ", n, kind)
			}
		}
	}
}
