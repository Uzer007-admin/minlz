# Accelerate overlapping amd64 decoder tails

Branch: `dev/minio-overlap-tail-prefix-copy`.
Base: `minio/minlz` main at `46d6409939197f4216d88f6d5049779cdac90c8e`.
Module path: `github.com/minio/minlz`.

Long overlapping matches in the checked amd64 tail currently run through a
byte-at-a-time loop. Highly repetitive blocks can put most of their output in
one final match, making this loop dominate decompression.

For matches of at least 128 bytes, copy from the completed prefix and grow that
prefix after each step. Each chunk is limited by both the initialized source
length and the remaining output. Eligible shorter overlaps use exact 16-byte
forward copies; the byte fallback remains available. The existing input,
offset, output, and final-position checks remain in place.

The change is made in the generator and regenerated amd64 assembly. ARM64
assembly is unchanged. The block format, stream format, and public API are
unchanged.

Commits:

- `820bb2d`: `amd64: expand decoded prefixes for overlapping tail copies`.
- `e073e44`: `test: exercise overlap tails across offsets and block sizes`.

Validation:

- Go 1.25.0, 1.26.0, and 1.27.1 on Windows and Linux amd64: normal, purego,
  noasm, nounsafe, nounsafe/noasm, vet, and race with one and four CPUs.
- Full race suites on Go 1.27.1; differential block/stream fuzzing on both OSes.
- Exact-capacity destinations with canaries, malformed lengths, truncated tags,
  copy offsets 1 through 65,537, and maximum-size blocks.
- Executed 386 tests on both OSes; GOAMD64=v3 on both and v4 on Linux;
  Linux/Windows ARM64 builds. ARM64 was not executed locally.
- Assembly generation is reproducible; ARM64 output hashes remain unchanged.
- Go/C block and stream interoperability over 15 sizes, five levels, and forced
  C backends, including 14 MiB as 8+6 MiB blocks. Recovered bytes are checked.
- The same decoding tests and comparisons also pass with compress v1.20.3.

Performance below is the median of seven alternating process runs at 150 ms
per case, using Go 1.27.1, GOAMD64=v2, GOMAXPROCS=1, and one pinned CPU. Both
implementations use compress v1.20.1. The reference is stock MinLZ v1.2.2.

| Platform | Repeated 16-byte pattern | Stock ns/op | Patched ns/op | Speedup |
|---|---:|---:|---:|---:|
| Windows | 64 B | 37.96 | 15.27 | 2.49x |
| Windows | 256 B | 87.02 | 21.62 | 4.03x |
| Windows | 4 KiB | 1,054 | 78.11 | 13.49x |
| Windows | 1 MiB | 265,079 | 25,546 | 10.38x |
| Windows | 14 MiB | 3,726,414 | 658,824 | 5.66x |
| Linux | 64 B | 44.33 | 19.46 | 2.28x |
| Linux | 256 B | 165.0 | 29.47 | 5.60x |
| Linux | 4 KiB | 2,456 | 101.5 | 24.20x |
| Linux | 1 MiB | 629,325 | 31,841 | 19.76x |
| Linux | 14 MiB | 8,899,678 | 817,643 | 10.88x |

All decode cases report zero allocations. The full matrix also covers a
one-byte repetition, a document pattern, random input, and explicit match
offsets/lengths: 105 cases per platform. Literal-only paths remain unchanged;
this patch makes no claim of improving their memory-copy throughput or total
application/query performance.

An apparent Windows regression for a 4 KiB offset and a 1 MiB match disappeared
in a focused 10 x 500 ms confirmation (rank permutation p=0.912). Large Windows
literal measurements vary with memory conditions; warmed built-in-copy
controls show the same variation, and the Decode wrapper and runtime.memmove
instruction sequences are identical between builds.

Reproduce the decoder benchmarks with:

```sh
GOMAXPROCS=1 go test -run '^$' -bench '^BenchmarkDecode(SizeMatrix|OverlapTail)$' -benchmem -benchtime=150ms -count=7
```

Pin the process and alternate reference/patched binaries for comparisons.
