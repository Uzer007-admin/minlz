// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build linux && amd64 && !appengine && !noasm && gc && !purego

package minlz

import (
	"bytes"
	"syscall"
	"testing"
)

// Exercise the complete short block with the last source and reservation
// bytes adjacent to inaccessible pages, including the 16383/16384 transition and arbitrary page counts.
func TestEncodeWholeSmallGuardPages(t *testing.T) {
	page := syscall.Getpagesize()
	guard := func(n int) ([]byte, func()) {
		b, e := syscall.Mmap(-1, 0, ((n+page-1)/page+1)*page, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
		if e != nil {
			t.Fatal(e)
		}
		if e = syscall.Mprotect(b[((n+page-1)/page)*page:], syscall.PROT_NONE); e != nil {
			syscall.Munmap(b)
			t.Fatal(e)
		}
		return b[((n+page-1)/page)*page-n : ((n+page-1)/page)*page : ((n+page-1)/page)*page], func() {
			if e := syscall.Munmap(b); e != nil {
				t.Error(e)
			}
		}
	}
	for n := 22; n <= 16385; n++ {
		for shape := 0; shape < 3; shape++ {
			src, releaseSrc := guard(n)
			dst, releaseDst := guard(MaxEncodedLen(n))
			x := uint64(n*13 + shape + 1)
			for i := range src {
				x ^= x << 13
				x ^= x >> 7
				x ^= x << 17
				src[i] = byte(x)
			}
			if shape < 2 {
				for i := 1 + shape*31; i < n; i++ {
					src[i] = src[i%(1+shape*31)]
				}
			}
			try := TryEncode(nil, src, LevelFastest)
			want := try
			if want == nil {
				want = append([]byte{0, 0}, src...)
			}
			out, e := Encode(dst, src, LevelFastest)
			if e != nil || !bytes.Equal(out, want) {
				t.Fatalf("n=%d shape=%d: %v", n, shape, e)
			}
			decoded, e := Decode(nil, out)
			if e != nil || !bytes.Equal(decoded, src) {
				t.Fatalf("decode n=%d shape=%d: %v", n, shape, e)
			}
			releaseDst()
			releaseSrc()
		}
	}
}
