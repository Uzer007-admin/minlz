// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build !appengine && !noasm && gc && !purego

package minlz

func encodeBlockLarge8M(dst, src []byte, tmp *[131072]byte) int { return encodeBlockAsm(dst, src, tmp) }
func encodeBlockLarge2M(dst, src []byte, tmp *[131072]byte) int {
	return encodeBlockAsm2MB(dst, src, tmp)
}
func encodeBlockLarge512K(dst, src []byte, tmp *[65536]byte) int {
	return encodeBlockAsm512K(dst, src, tmp)
}
