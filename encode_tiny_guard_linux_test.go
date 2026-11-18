// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build linux && amd64 && !appengine && !noasm && gc && !purego

package minlz

import (
	"bytes"
	"syscall"
	"testing"
)

// End both logical slices at an inaccessible page. An empty source points at
// the guard itself, so even its first byte is unavailable.
func TestEncodeTinyGuardPages(t *testing.T) {
	page := syscall.Getpagesize()
	guard := func(n int) ([]byte, func()) {
		size := ((n+page-1)/page + 1) * page
		b, e := syscall.Mmap(-1, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
		if e != nil {
			t.Fatal(e)
		}
		if e = syscall.Mprotect(b[size-page:], syscall.PROT_NONE); e != nil {
			syscall.Munmap(b)
			t.Fatal(e)
		}
		return b[size-page-n : size-page : size-page], func() {
			if e := syscall.Munmap(b); e != nil {
				t.Error(e)
			}
		}
	}
	for n := 0; n <= 21; n++ {
		src, releaseSrc := guard(n)
		dst, releaseDst := guard(MaxEncodedLen(n))
		for i := range src {
			src[i] = byte(i*31 + 17)
		}
		want := append([]byte{0, 0}, src...)
		if n == 0 {
			want = []byte{0}
		}
		got, e := Encode(dst, src, LevelFastest)
		if e != nil || !bytes.Equal(got, want) {
			t.Fatalf("n=%d: %v", n, e)
		}
		releaseDst()
		releaseSrc()
	}
}
