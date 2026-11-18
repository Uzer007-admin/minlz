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

func prefixProbeInput(seed, kind int) []byte {
	src := make([]byte, 4096)
	rand.New(rand.NewSource(int64(seed + 7001))).Read(src)
	if kind == 0 {
		for i := 71; i+8 < len(src); i += 127 {
			copy(src[i:i+8], src[i-63:i-55])
		}
	} else {
		copy(src[512:640], src[:128])
	}
	return src
}

func TestPrefixProbeWeakSamples(t *testing.T) {
	extra := 0
	for seed := 0; seed < 64; seed++ {
		for kind := 0; kind < 2; kind++ {
			src := prefixProbeInput(seed, kind)
			original := bytes.Clone(src)
			strict := make([]byte, 4098)
			old := encodeBlockShort4KSSE2(strict, src)
			storage := bytes.Repeat([]byte{0xa9}, 64+4098+64)
			dst := storage[64 : 64+4098 : 64+4098]
			got := encodeBlockProbe4KSSE2(dst, src)
			if old > 0 && (got != old || !bytes.Equal(dst[:got], strict[:old])) {
				t.Fatal("positive result", seed, kind, old, got)
			}
			if old == 0 && got > 0 {
				extra++
			}
			if got > 0 {
				block := make([]byte, 11+got)
				header := 1 + binary.PutUvarint(block[1:], 4096)
				copy(block[header:], dst[:got])
				decoded, e := Decode(nil, block[:header+got])
				if e != nil || !bytes.Equal(decoded, src) {
					t.Fatal("probe stream", seed, kind, e)
				}
			}
			if !bytes.Equal(src, original) || !bytes.Equal(storage[:64], bytes.Repeat([]byte{0xa9}, 64)) || !bytes.Equal(storage[len(storage)-64:], bytes.Repeat([]byte{0xa9}, 64)) {
				t.Fatal("guard", seed, kind)
			}
		}
	}
	if extra == 0 {
		t.Fatal("weak samples never distinguished from ratio decline")
	}
	t.Logf("SOFT_PROBE_EXTRA_CLASSIFICATIONS=%d", extra)
}
