// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build linux && amd64 && !appengine && !noasm && gc && !purego

package minlz

import (
	"bytes"
	"fmt"
	"syscall"
	"testing"
)

func TestEncoderSIMDGuardPages(t *testing.T) {
	page := syscall.Getpagesize()
	guard := func(n int) ([]byte, func()) {
		total := ((n+page-1)/page + 1) * page
		b, e := syscall.Mmap(-1, 0, total, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
		if e != nil {
			t.Fatal(e)
		}
		if e = syscall.Mprotect(b[total-page:], syscall.PROT_NONE); e != nil {
			syscall.Munmap(b)
			t.Fatal(e)
		}
		return b[total-page-n : total-page : total-page], func() {
			if e := syscall.Munmap(b); e != nil {
				t.Error(e)
			}
		}
	}
	for _, n := range []int{1025, 4095, 4096, 4097, 8191, 8192, 16383, 16384, 65535, 65536, 65537, 1048575, 1048576, 1048577, 2097152, 8388608} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			src, release := guard(n)
			defer release()
			dst, release := guard(MaxEncodedLen(n))
			defer release()
			for _, kind := range []int{0, 1, 2} {
				raw := optimizationBoundaryInput(n, kind)
				copy(src, raw)
				got, e := Encode(dst, src, LevelFastest)
				if e != nil {
					t.Fatal(e)
				}
				decoded, e := Decode(nil, got)
				if e != nil || !bytes.Equal(decoded, raw) || !bytes.Equal(src, raw) {
					t.Fatalf("kind=%d: %v", kind, e)
				}
				TryEncode(dst, src, LevelFastest)
			}
		})
	}
}
