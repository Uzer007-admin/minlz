package minlz_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/Uzer007-admin/minlz"
)

func TestIndexStreamConcatenatedEOF(t *testing.T) {
	const eofChunkType byte = 0x20
	encode := func(data []byte) []byte {
		t.Helper()
		var dst bytes.Buffer
		writer := minlz.NewWriter(&dst, minlz.WriterBlockSize(4096), minlz.WriterAddIndex(false))
		if _, err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		return dst.Bytes()
	}
	eof := func(size int) []byte {
		header := []byte{eofChunkType, 0, 0, 0}
		chunk := binary.AppendUvarint(header, uint64(size))
		chunk[1] = byte(len(chunk) - len(header))
		return chunk
	}
	first := []byte("first independent stream")
	second := bytes.Repeat([]byte("second stream with multiple blocks\n"), 180)
	a, b := encode(first), encode(second)
	footerA, footerB := eof(len(first)), eof(len(second))
	if !bytes.HasSuffix(a, footerA) || !bytes.HasSuffix(b, footerB) {
		t.Fatal("expected an EOF chunk with the source length from the writer")
	}
	aWithoutEOF, bWithoutEOF := a[:len(a)-len(footerA)], b[:len(b)-len(footerB)]
	unsizedEOF := []byte{eofChunkType, 0, 0, 0}
	total := len(first) + len(second)

	cases := []struct {
		name   string
		chunks [][]byte
		total  int
		err    error
	}{
		{name: "two-streams", chunks: [][]byte{a, b}, total: total},
		{name: "unsized-first-eof", chunks: [][]byte{aWithoutEOF, unsizedEOF, b}, total: total},
		{name: "empty-streams-between", chunks: [][]byte{a, eof(0), unsizedEOF, b}, total: total},
		{name: "empty-stream-at-end", chunks: [][]byte{a, b, eof(0)}, total: total},
		{name: "second-eof-is-cumulative", chunks: [][]byte{a, bWithoutEOF, eof(total)}, err: minlz.ErrCorrupt},
		{name: "second-eof-has-wrong-size", chunks: [][]byte{a, bWithoutEOF, eof(len(second) - 1)}, err: minlz.ErrCorrupt},
		{name: "nonempty-eof-after-eof", chunks: [][]byte{a, eof(1)}, err: minlz.ErrCorrupt},
		{name: "truncated-second-eof", chunks: [][]byte{a, b[:len(b)-1]}, err: io.ErrUnexpectedEOF},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := bytes.Join(tc.chunks, nil)
			data, err := minlz.IndexStream(bytes.NewReader(input))
			if !errors.Is(err, tc.err) {
				t.Fatalf("IndexStream error = %v, want %v", err, tc.err)
			}
			if err != nil {
				return
			}
			var index minlz.Index
			tail, err := index.Load(data)
			if err != nil || len(tail) != 0 {
				t.Fatalf("Index.Load tail = %d, error = %v", len(tail), err)
			}
			if index.TotalUncompressed != int64(tc.total) || index.TotalCompressed != int64(len(input)) {
				t.Errorf("indexed sizes = (%d, %d), want (%d, %d)",
					index.TotalUncompressed, index.TotalCompressed, tc.total, len(input))
			}
		})
	}
}
