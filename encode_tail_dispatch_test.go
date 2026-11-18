// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0

package minlz

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func pooledSmallReference(src []byte) ([]byte, bool) {
	dst := make([]byte, MaxEncodedLen(len(src)))
	d := 1 + binary.PutUvarint(dst[1:], uint64(len(src)))
	n := encodeBlock(dst[d:], src)
	if n > 0 {
		return dst[:d+n], true
	}
	return encodeUncompressed(dst[:0], src), false
}

func TestFastestSmallTryEncodeMatchesPooled(t *testing.T) {
	rng := rand.New(rand.NewSource(49117))
	for n := 17; n <= 4097; n++ {
		for shape := 0; shape < 4; shape++ {
			src := make([]byte, n)
			rng.Read(src)
			if shape < 3 {
				period := 1 + shape*31
				for i := period; i < n; i++ {
					src[i] = src[i%period]
				}
			}
			want, compressed := pooledSmallReference(src)
			storage := bytes.Repeat([]byte{0xa5}, MaxEncodedLen(n)+64)
			dst := storage[32 : len(storage)-32 : len(storage)-32]
			got := TryEncode(dst, src, LevelFastest)
			if compressed && !bytes.Equal(got, want) || !compressed && got != nil {
				t.Fatalf("TryEncode differs at size=%d shape=%d", n, shape)
			}
			if !bytes.Equal(storage[:32], bytes.Repeat([]byte{0xa5}, 32)) || !bytes.Equal(storage[len(storage)-32:], bytes.Repeat([]byte{0xa5}, 32)) {
				t.Fatalf("destination overrun at %d", n)
			}
		}
	}
}

//go:noinline
func tailDispatchDeep(depth int, dst, src []byte) ([]byte, error) {
	var canary [256]byte
	for i := range canary {
		canary[i] = byte(depth + i)
	}
	var out []byte
	var err error
	if depth == 0 {
		out, err = Encode(dst, src, LevelFastest)
	} else {
		out, err = tailDispatchDeep(depth-1, dst, src)
	}
	for i, b := range canary {
		if b != byte(depth+i) {
			return nil, fmt.Errorf("caller frame overwritten at depth %d", depth)
		}
	}
	return out, err
}

func TestFastestSmallTailCallStacks(t *testing.T) {
	var wg sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(117 + worker)))
			for _, n := range []int{17, 20, 21, 22, 35, 127, 128, 256, 257, 1024, 1025, 4096, 4097} {
				src := make([]byte, n)
				rng.Read(src)
				for _, period := range []int{0, 1, 7, 63} {
					if period != 0 {
						for i := period; i < n; i++ {
							src[i] = src[i%period]
						}
					}
					want, _ := pooledSmallReference(src)
					for _, depth := range []int{0, 1, 8, 32} {
						got, err := tailDispatchDeep(depth, make([]byte, MaxEncodedLen(n)), src)
						if err != nil || !bytes.Equal(got, want) {
							t.Errorf("tail call stack worker=%d size=%d depth=%d: %v", worker, n, depth, err)
							return
						}
					}
				}
			}
		}(worker)
	}
	wg.Wait()
}
