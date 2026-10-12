package bestfit_test

import (
	"errors"
	"testing"

	"github.com/doxuta/bestfit"
	"golang.org/x/text/encoding/japanese"
)

// A user table may hold a row whose replacement the base encoding refuses
// (a typo, or a row written for one encoding and used with another). The
// encoder must then fail the way it fails for any other unwritable rune: with
// an *UnsupportedError naming the input rune and its byte offset, and Lossy
// must report that rune as refused rather than claim the replacement is
// written.
func TestRegress20261012_BrokenRowIsUnsupportedError(t *testing.T) {
	e := bestfit.New(japanese.ShiftJIS, bestfit.Table{'x': "😀"})
	in := "東京x"
	got := e.Lossy(in)
	want := []bestfit.Loss{{Offset: 6, Rune: 'x'}}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("Lossy(%q) = %+v, want %+v", in, got, want)
	}
	_, err := e.NewEncoder().Bytes([]byte(in))
	var ue *bestfit.UnsupportedError
	if !errors.As(err, &ue) {
		t.Fatalf("err = %v (%T), want *UnsupportedError", err, err)
	}
	if ue.Rune != 'x' || ue.Offset != 6 {
		t.Errorf("got rune %q at byte %d, want 'x' at byte 6", ue.Rune, ue.Offset)
	}
}
