// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build !appengine && !noasm && gc && !purego

package minlz

import (
	"github.com/Uzer007-admin/minlz/internal/race"
	"sync"
)

var prefixPools [2]sync.Pool

func prefixHardware(src []byte) bool {
	if hasAVX2 {
		return prefixPartial(src)
	}
	return prefixSSEHardware && prefixSSEPartial(src) && prefixSSEPartial(src[len(src)/2:])
}

//go:noinline
func prefixPartial(src []byte) bool {
	var tmp [4098]byte
	race.ReadSlice(src)
	race.WriteSlice(tmp[:])
	var n int
	if hasAVX2 {
		n = encodeBlockShort4KAVX2(tmp[:], src[:4096])
	} else {
		n = encodeBlockShort4KSSE2(tmp[:], src[:4096])
	}
	return n > 3072 && n < 4096
}

func encodePrefixBlock(dst, src []byte) (d int) {
	if len(src) > 512<<10 {
		const sz = 262144
		p, ok := prefixPools[0].Get().(*[sz]byte)
		if !ok {
			p = &[sz]byte{}
		}
		race.WriteSlice(p[:])
		if prefixSSEHardware && !hasAVX2 {
			if len(src) > 2<<20 {
				d = encodeBlockPrefixSSE8M(dst, src, p)
			} else {
				d = encodeBlockPrefixSSE2M(dst, src, p)
			}
			if d < 0 {
				if len(src) > 2<<20 {
					d = encodeBlockLarge8M(dst, src, (*[131072]byte)(p[:]))
				} else {
					d = encodeBlockLarge2M(dst, src, (*[131072]byte)(p[:]))
				}
			}
		} else if len(src) > 2<<20 {
			d = encodeBlockPrefix8M(dst, src, p)
		} else {
			d = encodeBlockPrefix2M(dst, src, p)
		}
		prefixPools[0].Put(p)
		return d
	}
	const sz = 131072
	p, ok := prefixPools[1].Get().(*[sz]byte)
	if !ok {
		p = &[sz]byte{}
	}
	race.WriteSlice(p[:])
	d = encodeBlockPrefix512K(dst, src, p)
	prefixPools[1].Put(p)
	return d
}

//go:noescape
func encodeBlockPrefixSSE8M(dst, src []byte, tmp *[262144]byte) int

//go:noescape
func encodeBlockPrefixSSE2M(dst, src []byte, tmp *[262144]byte) int

//go:noinline
func prefixSSEPartial(src []byte) bool {
	var tmp [4098]byte
	race.ReadSlice(src)
	race.WriteSlice(tmp[:])
	n := encodeBlockProbe4KSSE2(tmp[:], src[:4096])
	return n > 3072 && n < 4096
}

//go:noescape
func encodeBlockProbe4KSSE2(dst, src []byte) int
