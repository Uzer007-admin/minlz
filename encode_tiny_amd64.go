// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build !appengine && !noasm && gc && !purego

package minlz

import "github.com/Uzer007-admin/minlz/internal/race"

const hasTinyAsm = true

// encodeBlockTiny preserves the regular size-class encoders' search order and output.
// Callers limit this path to at most 16384-byte Fastest blocks.
// Inputs of at most 21 bytes decline before entering a search kernel. Positions fit
// in uint8 for inputs <=256 bytes, and in uint16 for the remaining inputs.
// Keep the Go wrapper inlinable. The ABI0 dispatcher tail-jumps to a
// kernel with the same argument layout, so no extra Go frame is needed.
func encodeBlockTiny(dst, src []byte) int {
	race.ReadSlice(src)
	race.WriteSlice(dst)
	return encodeBlockTinyAsm(dst, src)
}

//go:noescape
func encodeBlockTinyAsm(dst, src []byte) int

//go:noescape
func encodeBlockShort(dst, src []byte) int

//go:noescape
func encodeBlockShort1K(dst, src []byte) int

//go:noescape
func encodeBlockShort4K(dst, src []byte) int

//go:noescape
func encodeBlockShort4KSSE2(dst, src []byte) int

//go:noescape
func encodeBlockShort4KAVX2(dst, src []byte) int

// encodeSmallWhole handles only 22..1024-byte Fastest inputs after reservation.
func encodeSmallWhole(dst, src []byte) int {
	race.ReadSlice(src)
	race.WriteSlice(dst)
	return encodeSmallWholeAsm(dst, src)
}

//go:noescape
func encodeSmallWholeAsm(dst, src []byte) int

//go:noescape
func encodeSmallWhole256Asm(dst, src []byte) int

//go:noescape
func encodeSmallWhole1024Asm(dst, src []byte) int

// encodeMediumWhole preserves the canonical 4K/16K parsers and writes the
// two-byte length header for 1025..16383. Zero declines to Go's stored copy.
func encodeMediumWhole(dst, src []byte) int {
	race.ReadSlice(src)
	race.WriteSlice(dst)
	return encodeMediumWholeAsm(dst, src)
}

//go:noescape
func encodeMediumWholeAsm(dst, src []byte) int

//go:noescape
func encodeSmallWhole4KAVX2Asm(dst, src []byte) int

//go:noescape
func encodeSmallWhole4KSSE2Asm(dst, src []byte) int

//go:noescape
func encodeSmallWhole16KAsm(dst, src []byte) int
