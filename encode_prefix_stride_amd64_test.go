// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build amd64 && !appengine && !noasm && gc && !purego

package minlz

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestPrefixCachedStrideExact(t *testing.T) {
	if !hasAVX2 {
		t.Skip("AVX2 kernel")
	}
	for _, n := range []int{524289, 1048576, 2097152, 2097153, 4194304, 8388608} {
		src := make([]byte, n)
		rand.New(rand.NewSource(int64(n))).Read(src)
		// At the first probe (s=1), h(s) == h(s+2). The old third
		// entry cannot match, but the just-written first entry must.
		copy(src, "zababababK")
		for i := 71; i+16 < n; i += 127 {
			copy(src[i:i+16], src[i-63:i-47])
		}
		// Include offset-1 and offset-2 matches and a checked 8M tail.
		for i := 2048; i+10 < n; i += 65533 {
			copy(src[i:], "zababababK")
			copy(src[i+11:], "zaaaaaaaaK")
		}
		want, got := make([]byte, MaxEncodedLen(n)), make([]byte, MaxEncodedLen(n))
		var a, b int
		if n > 2<<20 {
			a = encodeBlockLarge8M(want, src, new([131072]byte))
			b = encodeBlockPrefix8M(got, src, new([262144]byte))
		} else {
			a = encodeBlockLarge2M(want, src, new([131072]byte))
			b = encodeBlockPrefix2M(got, src, new([262144]byte))
		}
		if a <= 0 || a != b || !bytes.Equal(want[:a], got[:b]) {
			t.Fatalf("size=%d canonical=%d cached=%d", n, a, b)
		}
	}
}
