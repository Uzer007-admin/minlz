// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build linux && amd64 && !appengine && !noasm && gc && !purego

package minlz

import (
	"bytes"
	"math/rand"
	"syscall"
	"testing"
)

func TestFastestStack16GuardPages(t *testing.T) {
	page := syscall.Getpagesize()
	guarded := func(n int) ([]byte, func()) {
		size := ((n+page-1)/page + 1) * page
		b, err := syscall.Mmap(-1, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
		if err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mprotect(b[size-page:], syscall.PROT_NONE); err != nil {
			syscall.Munmap(b)
			t.Fatal(err)
		}
		return b[size-page-n : size-page : size-page], func() {
			if err := syscall.Munmap(b); err != nil {
				t.Error(err)
			}
		}
	}
	rng := rand.New(rand.NewSource(163844097))
	for _, n := range []int{4096, 4097, 4098, 8192, 16383, 16384, 16385} {
		for shape := 0; shape < 3; shape++ {
			src, releaseSrc := guarded(n)
			dst, releaseDst := guarded(MaxEncodedLen(n))
			rng.Read(src)
			if shape == 1 {
				for i := 251; i < n; i++ {
					src[i] = src[i%251]
				}
			} else if shape == 2 {
				for i := 71; i+8 < n; i += 127 {
					copy(src[i:i+8], src[i-63:i-55])
				}
			}
			want, compressed := pooledSmallReference(src)
			got, err := Encode(dst, src, LevelFastest)
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("Encode n=%d shape=%d: %v", n, shape, err)
			}
			got = TryEncode(dst, src, LevelFastest)
			if compressed && !bytes.Equal(got, want) || !compressed && got != nil {
				t.Fatalf("TryEncode n=%d shape=%d", n, shape)
			}
			releaseDst()
			releaseSrc()
		}
	}
}
