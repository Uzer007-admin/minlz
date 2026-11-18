// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0

package minlz

import (
	"bytes"
	"testing"
)

// Exercise both sides of the position-width and size-class boundaries without
// requiring the optional external benchmark corpus used by FuzzEncodingBlocks.
func FuzzFastestWireIdentity(f *testing.F) {
	for _, n := range []int{0, 15, 16, 17, 21, 22, 128, 255, 256, 257, 1024, 1025, 2048, 4096, 4097, 4098, 4608, 6144, 8192, 12288, 16383, 16384, 16385} {
		for _, period := range []int{1, 31, 251} {
			src := make([]byte, n)
			for i := range src {
				src[i] = byte((i%period)*97 + (i%period)>>3)
			}
			f.Add(src)
		}
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		if len(src) > 16385 {
			return
		}
		want := encodeUncompressed(nil, src)
		compressed := false
		if len(src) >= minNonLiteralBlockSize {
			want, compressed = pooledSmallReference(src)
		}
		compressed = compressed && len(want) < len(src)
		storage := bytes.Repeat([]byte{0xa5}, MaxEncodedLen(len(src))+64)
		dst := storage[32 : len(storage)-32 : len(storage)-32]
		got, err := Encode(dst, src, LevelFastest)
		if err != nil || !bytes.Equal(want, got) {
			t.Fatalf("wire differs at size %d: %v", len(src), err)
		}
		decoded, err := Decode(nil, got)
		if err != nil || !bytes.Equal(decoded, src) {
			t.Fatalf("decode at size %d: %v", len(src), err)
		}
		got = TryEncode(dst, src, LevelFastest)
		if compressed && !bytes.Equal(want, got) || !compressed && got != nil {
			t.Fatalf("TryEncode differs at size %d", len(src))
		}
		if !bytes.Equal(storage[:32], bytes.Repeat([]byte{0xa5}, 32)) ||
			!bytes.Equal(storage[len(storage)-32:], bytes.Repeat([]byte{0xa5}, 32)) {
			t.Fatalf("destination guard at size %d", len(src))
		}
	})
}
