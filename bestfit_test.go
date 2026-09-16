package bestfit_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/doxuta/bestfit"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var tbl = bestfit.Table{'〜': "～", '•': "*", 'ヶ': "ケ"} // ヶ is encodable: tables override the base

func enc() *bestfit.Encoding { return bestfit.New(japanese.ShiftJIS, tbl) }

func TestReplacesAndPassesThrough(t *testing.T) {
	got, err := enc().NewEncoder().Bytes([]byte("東京〜大阪•ヶ"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := japanese.ShiftJIS.NewEncoder().Bytes([]byte("東京～大阪*ケ"))
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X", got, want)
	}
}

func TestUnsupportedErrorNamesRuneAndOffset(t *testing.T) {
	in := "東京〜😀後"
	_, err := enc().NewEncoder().Bytes([]byte(in))
	var ue *bestfit.UnsupportedError
	if !errors.As(err, &ue) {
		t.Fatalf("err = %v, want *UnsupportedError", err)
	}
	if ue.Rune != '😀' || ue.Offset != strings.Index(in, "😀") {
		t.Fatalf("got %+v, want rune 😀 at byte %d", ue, strings.Index(in, "😀"))
	}
	if !strings.Contains(ue.Error(), "U+1F600") || !strings.Contains(ue.Error(), "byte 9") {
		t.Fatalf("message %q", ue.Error())
	}
}

func TestOffsetAcrossTransformCalls(t *testing.T) {
	// Feed the input in two pieces through one transformer, as a
	// transform.Writer would; the reported offset must be global.
	tr := enc().NewEncoder()
	tr.Reset()
	a, b := []byte("東京〜"), []byte("大阪😀")
	var dst [64]byte
	if _, n, err := tr.Transform(dst[:], a, false); err != nil || n != len(a) {
		t.Fatalf("first call: n=%d err=%v", n, err)
	}
	_, _, err := tr.Transform(dst[:], b, true)
	var ue *bestfit.UnsupportedError
	if !errors.As(err, &ue) || ue.Offset != len(a)+len("大阪") {
		t.Fatalf("err = %v, want offset %d", err, len(a)+len("大阪"))
	}
}

func TestShortDstMakesProgress(t *testing.T) {
	// Encode with a destination too small for the whole output; the loop
	// transform.Bytes runs must converge to the same bytes.
	in := strings.Repeat("東京〜大阪•", 50)
	want, err := enc().NewEncoder().Bytes([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	tr := enc().NewEncoder()
	tr.Reset()
	var out []byte
	src := []byte(in)
	for len(src) > 0 {
		var dst [7]byte // smaller than one replacement's neighbourhood
		nDst, nSrc, err := tr.Transform(dst[:], src, true)
		out = append(out, dst[:nDst]...)
		src = src[nSrc:]
		if err != nil && err != transform.ErrShortDst {
			t.Fatal(err)
		}
		if nDst == 0 && nSrc == 0 {
			t.Fatal("no progress")
		}
	}
	if !bytes.Equal(out, want) {
		t.Fatalf("chunked output differs: %d vs %d bytes", len(out), len(want))
	}
}

func TestShortSrcOnIncompleteRune(t *testing.T) {
	tr := enc().NewEncoder()
	tr.Reset()
	full := []byte("〜")
	var dst [8]byte
	nDst, nSrc, err := tr.Transform(dst[:], full[:2], false)
	if err != transform.ErrShortSrc || nDst != 0 || nSrc != 0 {
		t.Fatalf("got nDst=%d nSrc=%d err=%v, want ErrShortSrc", nDst, nSrc, err)
	}
	// A reader delivering byte by byte must still produce the right bytes.
	r := transform.NewReader(&iotest{data: []byte("〜a•")}, enc().NewEncoder())
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if want := []byte{0x81, 0x60, 'a', '*'}; !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("got % X want % X", buf.Bytes(), want)
	}
}

// iotest returns one byte per Read.
type iotest struct{ data []byte }

func (r *iotest) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	p[0] = r.data[0]
	r.data = r.data[1:]
	return 1, nil
}

func TestInvalidUTF8IsStillAnError(t *testing.T) {
	// A bad byte anywhere, including a truncated sequence right before a
	// replaced rune, is invalid UTF-8, not an unsupported rune and not a
	// request for more input.
	for _, in := range []string{"a\xffb", "\xe3\x80〜", "〜\xe3\x80"} {
		_, err := enc().NewEncoder().Bytes([]byte(in))
		if err != encoding.ErrInvalidUTF8 {
			t.Errorf("%q: err = %v, want ErrInvalidUTF8", in, err)
		}
	}
}

func TestLossy(t *testing.T) {
	in := "a〜b😀c"
	got := enc().Lossy(in)
	want := []bestfit.Loss{
		{Offset: 1, Rune: '〜', To: "～"},
		{Offset: 5, Rune: '😀'},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %+v want %+v", got, want)
		}
	}
}

func TestASCIIKeyedRowIsApplied(t *testing.T) {
	// Tables are consulted for every rune before it reaches the base encoding,
	// so a row keyed by an ASCII rune overrides the base like any other row,
	// and Lossy reports exactly the substitution the encoder makes.
	e := bestfit.New(japanese.ShiftJIS, bestfit.Table{'~': "～"})
	got, err := e.NewEncoder().Bytes([]byte("a~b"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{'a', 0x81, 0x60, 'b'}; !bytes.Equal(got, want) {
		t.Errorf("encode(%q) = % X, want % X", "a~b", got, want)
	}
	want := []bestfit.Loss{{Offset: 1, Rune: '~', To: "～"}}
	if l := e.Lossy("a~b"); len(l) != 1 || l[0] != want[0] {
		t.Errorf("Lossy(%q) = %+v, want %+v", "a~b", l, want)
	}
}

func TestDecoderIsBase(t *testing.T) {
	s, err := enc().NewDecoder().Bytes([]byte{0x81, 0x60})
	if err != nil || string(s) != "～" {
		t.Fatalf("got %q, %v", s, err)
	}
}

func FuzzEncode(f *testing.F) {
	for _, s := range []string{"", "a", "〜", "東京〜大阪", "😀", "\xff〜", "•\xe3\x80", "\xe3\x80〜", "ヶ〜ヶ"} {
		f.Add(s)
	}
	base := japanese.ShiftJIS.NewEncoder()
	f.Fuzz(func(t *testing.T, s string) {
		e := enc()
		out, err := e.NewEncoder().Bytes([]byte(s))
		var ue *bestfit.UnsupportedError
		switch {
		case err == nil:
			// Everything the table did not touch must match the base encoder.
			if !strings.ContainsAny(s, "〜•ヶ") {
				want, _ := base.Bytes([]byte(s))
				if !bytes.Equal(out, want) {
					t.Fatalf("%q: got % X want % X", s, out, want)
				}
			}
			if _, err := e.NewDecoder().Bytes(out); err != nil {
				t.Fatalf("%q: output does not decode: %v", s, err)
			}
		case errors.As(err, &ue):
			r, size := utf8.DecodeRuneInString(s[ue.Offset:])
			if r != ue.Rune || size == 0 {
				t.Fatalf("%q: offset %d does not hold %q", s, ue.Offset, ue.Rune)
			}
			if _, err := base.Bytes([]byte(string(ue.Rune))); err == nil {
				t.Fatalf("%q: %q reported unsupported but base writes it", s, ue.Rune)
			}
		default:
			if utf8.ValidString(s) {
				t.Fatalf("%q: unexpected error %v", s, err)
			}
		}
	})
}
