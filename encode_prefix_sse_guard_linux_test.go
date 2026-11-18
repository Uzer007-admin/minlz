// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build linux && amd64 && !appengine && !noasm && gc && !purego

package minlz

import (
	"bytes"
	"os"
	"syscall"
	"testing"
)

func TestPrefixSSEGuardPages(t *testing.T) {
	if !prefixSSEHardware {
		t.Skip("SSE4.2 target required")
	}
	page := syscall.Getpagesize()
	guard := func(n int) ([]byte, func()) {
		boundary := (n + page - 1) / page * page
		b, e := syscall.Mmap(-1, 0, boundary+page, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
		if e != nil {
			t.Fatal(e)
		}
		if e = syscall.Mprotect(b[boundary:], syscall.PROT_NONE); e != nil {
			syscall.Munmap(b)
			t.Fatal(e)
		}
		return b[boundary-n : boundary : boundary], func() {
			if e := syscall.Munmap(b); e != nil {
				t.Error(e)
			}
		}
	}
	sizes := []int{524289, 1048576, 2097152, 2097153, 2162687, 2162688, 3145728, 8388608}
	if os.Getenv("MINLZ_WINDOW_QUICK") == "1" {
		sizes = []int{1048576, 3145728}
	}
	fallbacks, cases := 0, 0
	for _, n := range sizes {
		for kind := 0; kind < 4; kind++ {
			src, freeSrc := guard(n)
			dst, freeDst := guard(MaxEncodedLen(n))
			scratch, freeScratch := guard(262144)
			x := uint64(n + kind + 1)
			for i := range src {
				x ^= x << 13
				x ^= x >> 7
				x ^= x << 17
				src[i] = byte(x)
			}
			switch kind {
			case 1:
				for i := 71; i+8 < n; i += 127 {
					copy(src[i:i+8], src[i-63:i-55])
				}
			case 2:
				for base := 32768; base < n; base += 32768 {
					end := min(n, base+8192)
					copy(src[base:end], src[:end-base])
				}
			case 3:
				for i := 0; i < n/2; i++ {
					src[i] = 'R'
				}
			}
			original := bytes.Clone(src)
			want := make([]byte, MaxEncodedLen(n))
			tmp := new([131072]byte)
			wn := 0
			if n > 2<<20 {
				wn = encodeBlockLarge8M(want, src, tmp)
			} else {
				wn = encodeBlockLarge2M(want, src, tmp)
			}
			gn := 0
			if n > 2<<20 {
				gn = encodeBlockPrefixSSE8M(dst, src, (*[262144]byte)(scratch))
			} else {
				gn = encodeBlockPrefixSSE2M(dst, src, (*[262144]byte)(scratch))
			}
			if gn < 0 {
				fallbacks++
				if n > 2<<20 {
					gn = encodeBlockLarge8M(dst, src, (*[131072]byte)(scratch))
				} else {
					gn = encodeBlockLarge2M(dst, src, (*[131072]byte)(scratch))
				}
			}
			if gn != wn || !bytes.Equal(dst[:gn], want[:wn]) || !bytes.Equal(src, original) {
				t.Fatalf("n=%d kind=%d got=%d want=%d", n, kind, gn, wn)
			}
			freeScratch()
			freeDst()
			freeSrc()
			cases++
		}
	}
	if fallbacks != 0 {
		t.Fatal("complete checked SSE parser unexpectedly restarted")
	}
	t.Logf("SSE_GUARD_CASES=%d FALLBACKS=%d AVX2=%v", cases, fallbacks, hasAVX2)
}
