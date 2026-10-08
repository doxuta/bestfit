package bestfit_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/doxuta/bestfit"
	"github.com/doxuta/bestfit/cp932"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
)

// encoding.ReplaceUnsupported and encoding.HTMLEscapeUnsupported act on the
// repertoire error the inner encoder returns (an error with a
// Replacement() byte method). They wrap every x/text encoder; they must wrap
// this one the same way, so that "best fit first, ? for the rest" composes.
func TestReplaceUnsupportedWrapsEncoder(t *testing.T) {
	in := "a〜😀b"

	got, err := encoding.ReplaceUnsupported(cp932.Encoding.NewEncoder()).Bytes([]byte(in))
	want, _ := encoding.ReplaceUnsupported(japanese.ShiftJIS.NewEncoder()).Bytes([]byte("a～😀b"))
	if err != nil || !bytes.Equal(got, want) {
		t.Errorf("ReplaceUnsupported: got % X, %v; want % X", got, err, want)
	}

	got, err = encoding.HTMLEscapeUnsupported(cp932.Encoding.NewEncoder()).Bytes([]byte(in))
	if want := []byte("a\x81\x60&#128512;b"); err != nil || !bytes.Equal(got, want) {
		t.Errorf("HTMLEscapeUnsupported: got % X, %v; want % X", got, err, want)
	}

	// The unwrapped error still names the rune and offset, and carries the
	// base encoding's substitute byte (0x1A for x/text's CJK encoders).
	_, err = cp932.Encoding.NewEncoder().Bytes([]byte(in))
	var ue *bestfit.UnsupportedError
	if !errors.As(err, &ue) || ue.Rune != '😀' || ue.Offset != 4 {
		t.Fatalf("err = %v, want UnsupportedError for 😀 at byte 4", err)
	}
	var re interface{ Replacement() byte }
	if !errors.As(err, &re) || re.Replacement() != encoding.ASCIISub {
		t.Errorf("Replacement() missing or wrong on %T", err)
	}
}
