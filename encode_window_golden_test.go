//go:build amd64

package minlz

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"os"
	"testing"
)

// Fixtures come from the independent pinned upstream, with matching build tags.
func TestWindowGolden(t *testing.T) {
	name := "testdata/window-golden.json"
	if !hasAsm {
		name = "testdata/window-golden-purego.json"
	}
	data, e := os.ReadFile(name)
	if e != nil {
		t.Fatal(e)
	}
	var rows []struct {
		N, Kind, Bytes int
		SHA            string
	}
	if e = json.Unmarshal(data, &rows); e != nil {
		t.Fatal(e)
	}
	for _, c := range rows {
		if os.Getenv("MINLZ_WINDOW_QUICK") == "1" && (c.N != 2162688 || c.Kind != 0) {
			continue
		}
		src := windowGoldenInput(c.N, c.Kind)
		original := bytes.Clone(src)
		for _, alignment := range []int{0, 1, 31} {
			cap := MaxEncodedLen(len(src))
			storage := bytes.Repeat([]byte{0xb3}, 64+alignment+cap+64)
			dst := storage[64+alignment : 64+alignment : len(storage)-64]
			out, e := Encode(dst, src, 1)
			if e != nil {
				t.Fatal(e)
			}
			h := sha256.Sum256(out)
			if len(out) != c.Bytes || hex.EncodeToString(h[:]) != c.SHA {
				t.Fatalf("window n=%d kind=%d alignment=%d differs", c.N, c.Kind, alignment)
			}
			decoded, e := Decode(nil, out)
			if e != nil || !bytes.Equal(decoded, src) {
				t.Fatal("window roundtrip", e)
			}
			want := bytes.Clone(out)
			tried := TryEncode(dst, src, 1)
			if len(want) < len(src) {
				if !bytes.Equal(tried, want) {
					t.Fatal("window TryEncode bytes")
				}
			} else if tried != nil {
				t.Fatal("window TryEncode should decline")
			}
			if !bytes.Equal(src, original) || !bytes.Equal(storage[:64+alignment], bytes.Repeat([]byte{0xb3}, 64+alignment)) || !bytes.Equal(storage[len(storage)-64:], bytes.Repeat([]byte{0xb3}, 64)) {
				t.Fatal("window source/capacity guard")
			}
		}
	}
}

func windowGoldenInput(n, kind int) []byte {
	src := make([]byte, n)
	rand.New(rand.NewSource(int64(n*997 + kind))).Read(src)
	end := n
	if kind != 0 {
		end = 4096
	}
	for i := 71; i+8 < end; i += 127 {
		copy(src[i:i+8], src[i-63:i-55])
	}
	switch kind {
	case 2:
		for i := 4096; i < n/2; i++ {
			src[i] = byte(i % 31)
		}
	case 3:
		for i := n / 2; i < n; i++ {
			src[i] = byte(i % 31)
		}
	case 4:
		for i := 4096; i+64 < n; i += 65537 {
			copy(src[i:i+64], src[1024:1088])
		}
	case 5:
		for _, distance := range []int{2162685, 2162686, 2162687} {
			for at := distance + 257; at+32 < n; at += 32771 {
				copy(src[at:at+32], src[at-distance:at-distance+32])
			}
		}
	}
	return src
}
