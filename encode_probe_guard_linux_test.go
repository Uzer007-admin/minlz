// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build linux && amd64 && !appengine && !noasm && gc && !purego

package minlz

import (
	"bytes"
	"encoding/binary"
	"syscall"
	"testing"
)

func TestPrefixProbeGuardPages(t *testing.T) {
	const n = 4096
	guard := func(length int) ([]byte, func()) {
		page := syscall.Getpagesize()
		boundary := (length + page - 1) / page * page
		b, e := syscall.Mmap(-1, 0, boundary+page, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
		if e != nil {
			t.Fatal(e)
		}
		if e = syscall.Mprotect(b[boundary:], syscall.PROT_NONE); e != nil {
			syscall.Munmap(b)
			t.Fatal(e)
		}
		return b[boundary-length : boundary : boundary], func() {
			if e := syscall.Munmap(b); e != nil {
				t.Error(e)
			}
		}
	}
	extra := 0
	for seed := 0; seed < 64; seed++ {
		for kind := 0; kind < 2; kind++ {
			src, freeSrc := guard(n)
			dst, freeDst := guard(n + 2)
			copy(src, prefixProbeInput(seed, kind))
			original := bytes.Clone(src)
			strict := make([]byte, n+2)
			old := encodeBlockShort4KSSE2(strict, src)
			got := encodeBlockProbe4KSSE2(dst, src)
			if old > 0 && (got != old || !bytes.Equal(dst[:got], strict[:old])) {
				t.Fatal("positive parser result changed", seed)
			}
			if old == 0 && got > 0 {
				extra++
			}
			if got > 0 {
				block := make([]byte, 1+binary.MaxVarintLen64+got)
				header := 1 + binary.PutUvarint(block[1:], n)
				copy(block[header:], dst[:got])
				block = block[:header+got]
				decoded, e := Decode(nil, block)
				if e != nil || !bytes.Equal(decoded, src) {
					t.Fatal("probe stream", seed, e)
				}
			}
			if !bytes.Equal(src, original) {
				t.Fatal("source changed")
			}
			freeDst()
			freeSrc()
		}
	}
	if extra == 0 {
		t.Fatal("weak samples never distinguished from ratio decline")
	}
	t.Logf("SOFT_PROBE_EXTRA_CLASSIFICATIONS=%d", extra)
}
