// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0
//go:build ignore

// Exact parser-preserving layout pass for generated large amd64 kernels.
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func must(ok bool, detail string) {
	if !ok {
		panic(detail)
	}
}
func replace(s, a, b string) string {
	must(strings.Count(s, a) == 1, "unexpected generator shape: "+a)
	return strings.Replace(s, a, b, 1)
}
func rx(s, pattern, repl string, n int) string {
	r := regexp.MustCompile(pattern)
	must(len(r.FindAllStringIndex(s, -1)) == n, "unexpected pattern: "+pattern)
	return r.ReplaceAllString(s, repl)
}

func cacheCold(s string) string {
	for _, suffix := range []string{"8M", "2M"} {
		name := "encodeBlockCold" + suffix
		label := "search_loop_" + name
		mul := "R10"
		if suffix == "8M" {
			mul = "R11"
		}
		start := strings.Index(s, label+":")
		end := strings.Index(s[start:], "\ncandidate3_match_"+name+":") + start
		must(start >= 0 && end > start, "cold labels")
		b := s[start:end]
		b = replace(b, label+":", label+":\n\tMOVQ 16(SP), BP\n\tMOVQ SI, R14\n\tSUBQ 24(SP), R14\n\tMOVQ $0xcf1bbcdcbf9b0000, R15\nsearch_fast_"+name+":")
		b = rx(b, `SUBQ\s+16\(SP\), AX`, `SUBQ BP, AX`, 1)
		b = rx(b, `\tMOVQ\s+\$0xcf1bbcdcbf9b0000, `+mul+`\n`, "", 1)
		b = rx(b, `IMULQ\s+`+mul+`,`, `IMULQ R15,`, 3)
		b = rx(b, `\tMOVL\s+BX, `+mul+`\n\tSUBQ\s+24\(SP\), `+mul+`\n`, "", 1)
		b = rx(b, `1\(SI\)\(`+mul+`\*1\)`, `1(R14)(BX*1)`, 1)
		b = rx(b, `(\tJMP\s+)`+label+`\n`, `${1}search_fast_`+name+"\n", 1)
		s = s[:start] + b + s[end:]
	}
	return s
}

func splitWindow(s string) string {
	name := "encodeBlockPrefix8M"
	start := strings.Index(s, "search_loop_"+name+":")
	end := strings.Index(s[start:], "\nprefix_hit_"+name+":") + start
	must(start >= 0 && end > start, "window labels")
	body := s[start:end]
	scanStart := strings.Index(body, "\tMOVQ     BX, R10")
	scanEnd := strings.Index(body, "\nwindow_ready_"+name+":")
	must(scanStart >= 0 && scanEnd > scanStart, "window scan")
	scan := body[scanStart:scanEnd]
	scan = replace(scan, "\tCMPL         R10, DI", "\tCMPL         R10, DX")
	scan = replace(scan, "\tVMOVD        X0, R10", "\tVMOVD        X0, R10\n\tLEAL -2(DI), R11\n\tCMPL R10, R11\n\tCMOVLGT R11, R10")
	scan = replace(scan, "\tLEAL         65536(DI), R10", "\tMOVQ         8(SP), R10")
	scan = replace(scan, "JMP          window_ready_"+name, "JMP          search_loop_"+name)
	scan = replace(scan, "\tMOVQ R10, 24(SP)", "\tCMPQ R10, 8(SP)\n\tCMOVQGT 8(SP), R10\n\tMOVQ R10, 24(SP)\n\tJMP search_loop_"+name)
	// Avoid paying for a table proof when few probes or little output are left.
	// This only selects the fully checked parser; it never assumes a match miss.
	scan = replace(scan, "\tMOVQ         $0x00000001, 32(SP)", "window_no_proof_"+name+":\n\tMOVQ         $0x00000001, 32(SP)")
	scan = "\tMOVL DI, R10\n\tSUBQ CX, R10\n\tCMPQ R10, $4096\n\tJAE window_no_proof_" + name + "\n\tMOVQ SI, R10\n\tSUBQ dst_base+0(FP), R10\n\tADDQ R10, R10\n\tCMPQ R10, DI\n\tJB window_no_proof_" + name + "\n" + scan
	kernel := body[strings.Index(body, "\tMOVQ         (R8)(DI*1), R10"):]
	r0 := strings.Index(kernel, "\tCMPB         32(SP)")
	r1 := strings.Index(kernel, "\nwindow_range_ok_"+name+":") + len("\nwindow_range_ok_"+name+":")
	must(r0 >= 0 && r1 > r0, "window range")
	fast := strings.ReplaceAll(kernel[:r0]+kernel[r1:], "search_fast_"+name, "search_valid_"+name)
	checked := replace(kernel, "\tCMPB         32(SP), $0x00\n\tJE           window_range_ok_"+name+"\n", "")
	checked = strings.ReplaceAll(checked, "search_fast_"+name, "search_checked_"+name)
	pre := body[strings.Index(body, "\tMOVL     DI, DX"):strings.Index(body, "\tCMPL     DI, 24(SP)")]
	pre = replace(pre, "DX, 8(SP)", "DX, 24(SP)")
	pre = replace(pre, "emit_remainder_"+name, "window_limit_"+name)
	text := "// Proven windows omit all per-candidate range tests. The checked copy\n// preserves the old clamps when table coverage cannot prove a window.\nsearch_loop_" + name + ":\n MOVQ $0xcf1bbcdcbf9b0000, R9\n CMPB 32(SP), $0\n JNE search_checked_" + name + "\n JMP search_valid_" + name + "\n PCALIGN $32\nsearch_valid_" + name + ":\n" + pre + fast + "\n PCALIGN $32\nsearch_checked_" + name + ":\n" + pre + checked + "\nwindow_limit_" + name + ":\n\tCMPQ DX, 8(SP)\n\tJAE emit_remainder_" + name + "\n" + scan + "\n"
	s = s[:start] + text + s[end:]
	needle := "\tMOVQ DI, 8(SP)"
	at := strings.Index(s, needle) + len(needle)
	must(at >= len(needle), "initial window limit")
	return s[:at] + "\n\tCMPQ DI, 24(SP)\n\tJAE window_initial_ok_" + name + "\n\tMOVQ DI, 24(SP)\nwindow_initial_ok_" + name + ":" + s[at:]
}

func avxLiterals(s string) string {
	re := regexp.MustCompile(`(?P<start>\t// genMemMoveLong\n\tCMPQ\s+(?P<len>\w+), \$0x00000080\n\tJA\s+[^\n]+\n)(?P<moves>(?:\tMOVOU[^\n]*\n){16})(?P<end>\tJMP\s+[^\n]+)`)
	count := 0
	s = re.ReplaceAllStringFunc(s, func(block string) string {
		m := re.FindStringSubmatch(block)
		get := func(n string) string { return m[re.SubexpIndex(n)] }
		moves := strings.Split(strings.TrimSuffix(get("moves"), "\n"), "\n")
		from := regexp.MustCompile(`\tMOVOU\s+\((\w+)\), X0`).FindStringSubmatch(moves[0])
		to := regexp.MustCompile(`\tMOVOU\s+X0, \((\w+)\)`).FindStringSubmatch(moves[8])
		must(len(from) == 2 && len(to) == 2, "literal registers")
		address := func(base string, off int) string {
			if off == 0 {
				return "(" + base + ")"
			}
			if off > 0 {
				return fmt.Sprint(off) + "(" + base + ")"
			}
			return fmt.Sprint(off) + "(" + base + ")(" + get("len") + "*1)"
		}
		var lines []string
		for i, off := range []int{0, 32, -64, -32} {
			lines = append(lines, fmt.Sprintf("\tVMOVDQU %s, Y%d", address(from[1], off), i))
		}
		for i, off := range []int{0, 32, -64, -32} {
			lines = append(lines, fmt.Sprintf("\tVMOVDQU Y%d, %s", i, address(to[1], off)))
		}
		lines = append(lines, "\tVZEROUPPER")
		count++
		return get("start") + strings.Join(lines, "\n") + "\n" + get("end")
	})
	must(count == 9, "expected nine exact-span AVX2 literal sites")
	return s
}
func main() {
	mode := flag.String("mode", "", "cold or prefix")
	path := flag.String("file", "", "generated assembly")
	flag.Parse()
	input := *path
	if *mode == "whole" {
		input = "../encode_short_amd64.s"
	}
	data, e := os.ReadFile(input)
	if e != nil {
		panic(e)
	}
	s := string(data)
	switch *mode {
	case "cold":
		s = cacheCold(s)
		copyCachedCold(s)
	case "prefix":
		s = prefetch2M(earlyTable(avxLiterals(splitWindow(s))))
		s = ssePrefix(s)
		s = sparseCachedStride(s)
		s = sparseScalarMiss(s)
		s = sparseMemoryHash(s)
		s = sparseCompactStride(s)
		s = sparseRefreshStride(s)
	case "sse-only":
		s = ssePrefix(s)
	case "probe-only":
		emitSSEProbe()
	case "whole":
		s = wholeThrough16K(s)
		emitSSEProbe()
	default:
		panic("unknown mode")
	}
	if e = os.WriteFile(*path, []byte(s), 0644); e != nil {
		panic(e)
	}
}

// Emit an SSE4.1 prefix dictionary implementation. The public minimum target
// is GOAMD64=v2 (SSE4.2). Old candidate positions are used only in proved windows;
// otherwise the original clamped candidate checks continue in SSE4.2.
func ssePrefix(s string) string {
	var out strings.Builder
	out.WriteString("// Code generated by go run postopt.go augment.go extend.go -mode prefix. DO NOT EDIT.\n//go:build !appengine && !noasm && gc && !purego\n\n#include \"textflag.h\"\n\n")
	out.WriteString(`DATA prefixVectorMask<>+0(SB)/8, $0x0403020103020100
DATA prefixVectorMask<>+8(SB)/8, $0x8080808005040302
GLOBL prefixVectorMask<>(SB), RODATA|NOPTR, $16
DATA prefixVectorDelta<>+0(SB)/8, $0x0000000100000000
DATA prefixVectorDelta<>+8(SB)/8, $0x0000000300000002
GLOBL prefixVectorDelta<>(SB), RODATA|NOPTR, $16

`)
	for _, size := range []string{"8M", "2M"} {
		name := "encodeBlockPrefix" + size
		a := strings.Index(s, "// func "+name+"(")
		must(a >= 0, "sse prefix function")
		b := len(s)
		if n := strings.Index(s[a+8:], "// func "); n >= 0 {
			b = a + 8 + n
		}
		body := s[a:b]
		if size == "8M" {
			re := regexp.MustCompile(`(?s)	LEAL\s+-2162685\(DI\), R11
.*?	JNE\s+prefix_range_slow_` + name + `
`)
			must(len(re.FindAllStringIndex(body, -1)) == 1, "SSE checked range block")
			body = re.ReplaceAllStringFunc(body, func(string) string {
				return fmt.Sprintf("\tLEAL -2162685(DI), R11\n\tMOVLQSX R11, R11\n\tSHLQ $32, R11\n\tMOVQ R11, X6\n\tPUNPCKLQDQ X6, X6\n\tMOVOU X2, X1\n\tPUNPCKLQDQ X3, X1\n\tPCMPGTQ X6, X1\n\tMOVOU X4, X7\n\tPCMPGTQ X6, X7\n\tMOVMSKPD X1, R13\n\tMOVMSKPD X7, R14\n\tANDL $1, R14\n\tSHLL $2, R14\n\tORL R14, R13\n\tCMPL R13, $7\n\tJNE prefix_range_slow_%s\n", name)
			})
			a = strings.Index(body, "\tMOVQ     BX, R10\n")
			b = strings.Index(body, "\tLEAL -2(DI), R11\n")
			must(a >= 0 && b > a, "sse min scan")
			scan := fmt.Sprintf("\tMOVQ BX, R10\n\tMOVL $16384, R11\n\tPCMPEQL X0, X0\nwindow_scan_%s:\n\tMOVOU (R10), X2\n\tPSHUFD $0xdd, X2, X2\n\tPMINUD X2, X0\n\tADDQ $16, R10\n\tDECL R11\n\tJNZ window_scan_%s\n\tPSHUFD $0x4e, X0, X1\n\tPMINUD X1, X0\n\tPSHUFD $0xb1, X0, X1\n\tPMINUD X1, X0\n\tPEXTRD $0, X0, R10\n", name, name)
			body = body[:a] + scan + body[b:]
		}
		// Restore the original exact-span 16-byte literal copies, including all
		// loads before any stores. These spans are also valid for <=128 bytes.
		lit := regexp.MustCompile(`(?s)(\t// genMemMoveLong\n\tCMPQ\s+(\w+), \$0x00000080\n\tJA\s+[^\n]+\n)((?:\tVMOVDQU[^\n]*\n){8})\tVZEROUPPER\n`)
		count := 0
		body = lit.ReplaceAllStringFunc(body, func(block string) string {
			m := lit.FindStringSubmatch(block)
			lines := strings.Split(strings.TrimSpace(m[3]), "\n")
			from := regexp.MustCompile(`VMOVDQU\s+\((\w+)\), Y0`).FindStringSubmatch(lines[0])
			to := regexp.MustCompile(`VMOVDQU\s+Y0, \((\w+)\)`).FindStringSubmatch(lines[4])
			must(len(from) == 2 && len(to) == 2, "sse literal registers")
			address := func(base string, off int) string {
				if off == 0 {
					return "(" + base + ")"
				}
				if off > 0 {
					return fmt.Sprint(off) + "(" + base + ")"
				}
				return fmt.Sprint(off) + "(" + base + ")(" + m[2] + "*1)"
			}
			text := m[1]
			offsets := []int{0, 16, 32, 48, -64, -48, -32, -16}
			for i, off := range offsets {
				text += fmt.Sprintf("\tMOVOU %s, X%d\n", address(from[1], off), i)
			}
			for i, off := range offsets {
				text += fmt.Sprintf("\tMOVOU X%d, %s\n", i, address(to[1], off))
			}
			count++
			return text
		})
		must(count == 3, "sse literal count")
		feature := regexp.MustCompile(`\tCMPB\s+·hasAVX2\+0\(SB\), \$0x00\n\tJE\s+([^\n]+)\n`)
		body = feature.ReplaceAllString(body, "\tJMP $1\n")
		var lines []string
		binary := map[string]string{"VPSHUFB": "PSHUFB", "VPADDD": "PADDL", "VPUNPCKLDQ": "PUNPCKLLQ", "VPUNPCKHDQ": "PUNPCKHLQ", "VMOVLHPS": "MOVLHPS", "VPCMPEQD": "PCMPEQL", "VPAND": "PAND", "VPCMPEQB": "PCMPEQB", "VPMINUD": "PMINUD"}
		simple := map[string]string{"VMOVQ": "MOVQ", "VMOVD": "MOVL", "VMOVDQU": "MOVOU", "VPSHUFD": "PSHUFD", "VPEXTRQ": "PEXTRQ", "VPEXTRD": "PEXTRD", "VPMOVMSKB": "PMOVMSKB"}
		for _, line := range strings.Split(body, "\n") {
			fields := strings.Fields(line)
			if len(fields) == 0 {
				lines = append(lines, line)
				continue
			}
			op := fields[0]
			if op == "VZEROUPPER" {
				continue
			}
			if op == "TZCNTQ" {
				line = strings.Replace(line, op, "BSFQ", 1)
			}
			// Remaining 256-bit instructions are in unreachable original AVX match
			// loops and zeroing blocks; every feature branch above is unconditional SSE.
			if strings.Contains(line, "Y") {
				lines = append(lines, line)
				continue
			}
			if code, ok := binary[op]; ok {
				args := strings.Split(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), op)), ",")
				must(len(args) == 3, "sse binary operands")
				for i := range args {
					args[i] = strings.TrimSpace(args[i])
				}
				if args[1] != args[2] {
					must(args[0] != args[2], "sse destructive source alias")
					lines = append(lines, "\tMOVOU "+args[1]+", "+args[2])
				}
				line = "\t" + code + " " + args[0] + ", " + args[2]
			} else if code, ok := simple[op]; ok {
				line = strings.Replace(line, op, code, 1)
			} else {
				must(!strings.HasPrefix(op, "V"), "unconverted SSE instruction "+line)
			}
			lines = append(lines, line)
		}
		body = strings.Join(lines, "\n")
		body = strings.ReplaceAll(body, name, "encodeBlockPrefixSSE"+size)
		out.WriteString(stripSSEDead(body))
	}
	if e := os.WriteFile("../encode_prefix_sse_amd64.s", []byte(out.String()), 0644); e != nil {
		panic(e)
	}
	return s
}

// Drop the original AVX blocks after the SSE branch conversion. Reachability
// is computed from the entry through labels, conditional branches and returns.
// The resulting function contains no VEX instructions, even in dead code.
func stripSSEDead(body string) string {
	lines := strings.Split(body, "\n")
	labels := map[string]int{}
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasSuffix(t, ":") {
			labels[strings.TrimSuffix(t, ":")] = i
		}
	}
	live := make([]bool, len(lines))
	pending := []int{0}
	for len(pending) > 0 {
		i := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if i >= len(lines) || live[i] {
			continue
		}
		live[i] = true
		f := strings.Fields(lines[i])
		if len(f) == 0 {
			pending = append(pending, i+1)
			continue
		}
		op := f[0]
		if op == "RET" {
			continue
		}
		if strings.HasPrefix(op, "J") {
			must(len(f) == 2, "SSE branch arity")
			target, ok := labels[f[1]]
			must(ok, "SSE branch label "+f[1])
			pending = append(pending, target)
			if op == "JMP" {
				continue
			}
		}
		pending = append(pending, i+1)
	}
	var out []string
	for i, line := range lines {
		if !live[i] {
			continue
		}
		f := strings.Fields(line)
		if len(f) > 0 {
			must(!strings.HasPrefix(f[0], "V"), "reachable VEX in SSE backend: "+line)
		}
		if strings.Contains(line, "// Requires:") {
			line = "// Requires: SSE2, SSSE3, SSE4.2. Selected only for GOAMD64>=v2."
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n") + "\n"
}

// A classification-only parser accepts weakly compressible samples. Its bytes
// never become part of the public encoded stream. Retain the original 17-byte
// safety margin while removing only the 1/32 ratio requirement.
func emitSSEProbe() {
	data, e := os.ReadFile("../encode_short4k_sse_amd64.s")
	if e != nil {
		panic(e)
	}
	s := string(data)
	s = strings.Replace(s, strings.SplitN(s, "\n", 2)[0], "// Code generated by go run postopt.go augment.go extend.go -mode whole. DO NOT EDIT.", 1)
	s = strings.ReplaceAll(s, "encodeBlockShort4KSSE2", "encodeBlockProbe4KSSE2")
	s = rx(s, `\tSUBL\s+DX, DI\n`, "", 1)
	feature := regexp.MustCompile(`\tCMPB\s+·hasAVX2\+0\(SB\), \$0x00\n\tJE\s+([^\n]+)\n`)
	must(len(feature.FindAllStringIndex(s, -1)) > 0, "probe SSE guards")
	s = feature.ReplaceAllString(s, "\tJMP $1\n")
	s = regexp.MustCompile(`(?m)^\s*VZEROUPPER\s*\n`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "TZCNTQ", "BSFQ")
	s = stripSSEDead(s)
	if e := os.WriteFile("../encode_probe4k_sse_amd64.s", []byte(s), 0644); e != nil {
		panic(e)
	}
}
