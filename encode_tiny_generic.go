// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build !amd64 || appengine || noasm || !gc || purego

package minlz

const hasTinyAsm = false

func encodeBlockTiny(dst, src []byte) int { return encodeBlock(dst, src) }

func encodeSmallWhole(dst, src []byte) int { panic("unreachable small ASM") }

func encodeMediumWhole(dst, src []byte) int { panic("unreachable medium ASM") }

func encodeSmallRepair(dst, src []byte) int { panic("unreachable small ASM repair") }

func smallUseWhole(n int) bool                     { return false }
func encodeSmallCanonicalTiny(dst, src []byte) int { panic("unreachable small Tiny repair") }

const useReservedFrontend = false
