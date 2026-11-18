// Copyright 2026 MinIO Inc.
// SPDX-License-Identifier: Apache-2.0

package minlz

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestTryEncodeTinyNoAllocation(t *testing.T) {
	for size := 0; size <= 21; size++ {
		if !hasTinyAsm && size >= minNonLiteralBlockSize {
			continue
		}
		src := bytes.Repeat([]byte{'q'}, size)
		if got := TryEncode(nil, src, LevelFastest); got != nil {
			t.Fatalf("size %d unexpectedly compressed", size)
		}
		if n := testing.AllocsPerRun(100, func() {
			if TryEncode(nil, src, LevelFastest) != nil {
				t.Fatal("unexpected compression")
			}
		}); n != 0 {
			t.Fatalf("size %d: %.1f allocations on a known decline", size, n)
		}
		got, err := Encode(nil, src, LevelFastest)
		if err != nil || !bytes.Equal(got, encodeUncompressed(nil, src)) {
			t.Fatalf("size %d: stored bytes changed: %v", size, err)
		}
	}
}

func TestWriterSyncBufferReuseAndRelease(t *testing.T) {
	src := bytes.Repeat([]byte("repeatable small document;"), 7)
	w := NewWriter(io.Discard, WriterLevel(LevelFastest), WriterConcurrency(1), WriterBlockSize(4096), WriterCreateIndex(false))
	write := func() {
		if n, err := w.Write(src); err != nil || n != len(src) {
			t.Fatalf("Write: %d, %v", n, err)
		}
		if err := w.Flush(); err != nil {
			t.Fatal(err)
		}
	}
	write()
	if len(w.syncBuf) != w.obufLen {
		t.Fatal("single-worker buffer was not retained")
	}
	if n := testing.AllocsPerRun(100, write); n != 0 {
		t.Fatalf("steady-state Write/Flush: %.1f allocations per block", n)
	}
	w.Reset(io.Discard)
	if w.syncBuf != nil {
		t.Fatal("Reset retained the active stream's buffer")
	}
	write()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if w.syncBuf != nil {
		t.Fatal("Close retained the active stream's buffer")
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

type optimizationFailWriter struct {
	calls int
	short bool
	err   error
}

func (w *optimizationFailWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == 2 { // Stream header succeeds, then the first block fails.
		if w.short {
			return len(p) - 1, nil
		}
		return 0, w.err
	}
	return len(p), nil
}

func TestWriterSyncBufferReleaseAfterError(t *testing.T) {
	for _, short := range []bool{false, true} {
		sentinel := errors.New("output failed")
		dst := &optimizationFailWriter{short: short, err: sentinel}
		w := NewWriter(dst, WriterConcurrency(1), WriterBlockSize(4096))
		_, _ = w.Write(bytes.Repeat([]byte{'q'}, 128))
		want := sentinel
		if short {
			want = io.ErrShortWrite
		}
		if err := w.Flush(); !errors.Is(err, want) {
			t.Fatalf("Flush: got %v, want %v", err, want)
		}
		if err := w.Close(); !errors.Is(err, want) {
			t.Fatalf("Close: got %v, want %v", err, want)
		}
		if w.syncBuf != nil {
			t.Fatal("failed stream retained its buffer after Close")
		}
		var recovered bytes.Buffer
		w.Reset(&recovered)
		src := []byte("stream after an output failure")
		_, _ = w.Write(src)
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(NewReader(bytes.NewReader(recovered.Bytes())))
		if err != nil || !bytes.Equal(got, src) {
			t.Fatalf("Reset after failure: %v", err)
		}
	}
}
