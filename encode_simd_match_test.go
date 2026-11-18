// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build amd64 && !appengine && !noasm && gc && !purego

package minlz

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"
)

func simdMismatchInput(n, flip int) []byte {
	raw := make([]byte, n)
	for i := range raw {
		j := i % 73
		raw[i] = byte(j*97 + j>>3)
	}
	if flip >= 0 {
		raw[73+flip] ^= 0x59
	}
	return raw
}

// The digest comes from f5bca932, independently of both SIMD implementations.
// Every possible first difference in two vector groups plus their boundaries
// is exercised, with source and destination cycling through all 16 alignments.
// Run the same test with MINLZ_TEST_SSE2=1 to select the no-AVX path.
func TestEncoderSIMDMatchBoundaries(t *testing.T) {
	goldens := map[int]string{
		1025:  "f014b7256dd23162402ea197d62faffb3b26f749e28912a4073eaf13db9f8589",
		4096:  "fcfe4cbe3eaa15b5656de08876d6f6ecb67fc877031050e492150404969887d1",
		4097:  "ab5b0e93519ff766b7c3228ca61c771fb29d1b899604c783284bb88093c2f486",
		16384: "2439076fb9818cb084204c0d634ce4521b52c7c512f4e62add5e41e0d12a5333",
		65537: "2623f30319a08bfb282723ad04afd6be50bbcbfe9da63576133d82c20bac1678",
	}
	for n, want := range goldens {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			h := sha256.New()
			var count [8]byte
			for flip := -1; flip < 273; flip++ {
				raw := simdMismatchInput(n, flip)
				shift := (flip + 1) & 15
				input := make([]byte, n+16)
				src := input[shift : shift+n : shift+n]
				copy(src, raw)
				buf := bytes.Repeat([]byte{0xa5}, MaxEncodedLen(n)+64)
				start := 16 + shift
				capend := start + MaxEncodedLen(n)
				dst := buf[start:capend:capend]
				got, e := Encode(dst, src, LevelFastest)
				if e != nil {
					t.Fatal(e)
				}
				binary.LittleEndian.PutUint64(count[:], uint64(len(got)))
				h.Write(count[:])
				h.Write(got)
				decoded, e := Decode(nil, got)
				if e != nil || !bytes.Equal(decoded, raw) {
					t.Fatalf("flip=%d decode: %v", flip, e)
				}
				saved := append([]byte(nil), got...)
				got = TryEncode(dst, src, LevelFastest)
				if !bytes.Equal(saved, got) {
					t.Fatalf("flip=%d TryEncode differs", flip)
				}
				if !bytes.Equal(src, raw) || !bytes.Equal(buf[:start], bytes.Repeat([]byte{0xa5}, start)) || !bytes.Equal(buf[capend:], bytes.Repeat([]byte{0xa5}, len(buf)-capend)) {
					t.Fatalf("flip=%d source or destination boundary", flip)
				}
			}
			if got := hex.EncodeToString(h.Sum(nil)); got != want {
				t.Fatalf("wire digest: %s; want %s", got, want)
			}
		})
	}
}
