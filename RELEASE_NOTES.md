# v1.2.5

Integrate the final review revisions from [PR #84](https://github.com/minio/minlz/pull/84)
and [PR #85](https://github.com/minio/minlz/pull/85) into the fork.

The decoder optimization and compressed-data CRC fixes are already included in
v1.2.4. This release strengthens their regression coverage without changing
runtime code, public APIs, module paths, dependencies, or the encoded format.

Overlap-tail tests now live in `decode_test.go`; `benchmarks_test.go` contains
12 codec-only cases covering copy widths, prefix-growth thresholds, offsets,
and long matches. The application-specific Reindexer/CPROTO harness and corpus
remain outside the repository as an [external attachment](https://github.com/user-attachments/files/33578544/minlz-reindexer-evidence.zip).

CRC tests now live in `reader_test.go`. The boundary-search reproducer forces
the `decodePending` path, verifies the exact match position and restored context,
and asserts that a deferred block is decoded rather than skipped. Separate
coverage checks lazy previous-block decoding, corrupt checksums, malformed
input, CRC-ignore options, and concurrent buffer reuse. The test source also
includes the requested `go fix` update.

```sh
go get github.com/Uzer007-admin/minlz@v1.2.5
go install github.com/Uzer007-admin/minlz/cmd/mz@v1.2.5
```

# v1.2.4

Fix compressed-data CRC handling for MinLZ stream chunks of type `0x03`.

Concurrent decoding now computes the compressed checksum before returning
the input buffer to the reader pool. Deferred search and previous-block context
checks exclude the decoded-size varint, matching the other stream readers.

Incorrect checksums remain errors. `ReaderIgnoreCRC` and the corresponding
search option still skip checksums while rejecting malformed blocks. Chunk
`0x02` continues to validate the uncompressed bytes. Public APIs, stream
formats, dependencies, and the amd64 overlap optimization are unchanged.

Regression tests cover both CRC modes, deferred and lazy decoding, boundary
search matches, previous-block context, concurrent buffer reuse, invalid first
and last checksums, and the CRC-ignore options.

```sh
go get github.com/Uzer007-admin/minlz@v1.2.4
go install github.com/Uzer007-admin/minlz/cmd/mz@v1.2.4
```

# v1.2.3

This release publishes the fork as `github.com/Uzer007-admin/minlz` and keeps
the existing MinLZ block and stream format. Go 1.25 or later is required.

```sh
go get github.com/Uzer007-admin/minlz@v1.2.3
go install github.com/Uzer007-admin/minlz/cmd/mz@v1.2.3
```

The amd64 decoder expands the completed prefix for long overlapping tail
matches and uses bounded 16-byte copies for eligible short matches. This
removes the byte-copy bottleneck on repetitive blocks. Input/output bounds and
corruption checks are preserved. The generator is updated; ARM64 assembly is
unchanged.

Stream index construction now validates complete EOF size varints and checks
the declared size against the indexed data. An omitted optional EOF size is
still accepted.

The release includes upstream changes through `46d6409`, including the encoder
offset bounds, scratch-pool fixes, ARM64 encoder updates, and minimal 64 KiB
literal headers. The dedicated upstream branch retains the canonical MinIO
module path; see [the proposed PR description](UPSTREAM_PR.md).

Validation covered Windows and Linux on Go 1.25.0, 1.26.0, and 1.27.1, normal
and fallback build modes, race, vet, differential fuzzing, and Go/C
interoperability through 14 MiB. ARM64 builds were checked without local ARM64
execution. Both compress v1.20.1 and the consumer-selected v1.20.3 were tested.

For a repeated 16-byte pattern, seven pinned comparisons against stock MinLZ
v1.2.2 measured:

| Platform | Input | Stock | v1.2.3 | Speedup |
|---|---:|---:|---:|---:|
| Windows | 1 MiB | 265.1 us | 25.5 us | 10.38x |
| Windows | 14 MiB | 3.73 ms | 0.659 ms | 5.66x |
| Linux | 1 MiB | 629.3 us | 31.8 us | 19.76x |
| Linux | 14 MiB | 8.90 ms | 0.818 ms | 10.88x |

Decoding reports `0 B/op` and `0 allocs/op`. The measured size grid spans 16
bytes through 14 MiB. Literal paths retain their existing behavior; the results
do not establish a speedup for every input pattern or for an entire application.
