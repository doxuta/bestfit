package bestfit_test

import (
	"reflect"
	"testing"
	"unicode/utf8"

	"github.com/doxuta/bestfit"
	"github.com/doxuta/bestfit/cp932"
)

// Lossy used to range over the string, so an invalid byte arrived as U+FFFD
// and was reported as a refused rune — the same report a genuine U+FFFD gets,
// and the opposite of what the encoder does with the same input
// (encoding.ErrInvalidUTF8). An invalid byte is now a Loss with Invalid set.
func TestRegress20261015_LossyFlagsInvalidUTF8(t *testing.T) {
	e := cp932.Encoding
	got := e.Lossy("a\xffb〜")
	want := []bestfit.Loss{
		{Offset: 1, Rune: utf8.RuneError, Invalid: true},
		{Offset: 3, Rune: '〜', To: "～"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lossy(invalid) = %+v, want %+v", got, want)
	}
	// A real U+FFFD in the input is a rune the base cannot write, not bad input.
	got = e.Lossy("a�b")
	want = []bestfit.Loss{{Offset: 1, Rune: utf8.RuneError}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lossy(U+FFFD) = %+v, want %+v", got, want)
	}
	// A truncated sequence at the end is one invalid byte per byte, as the
	// encoder sees it.
	got = e.Lossy("\xe3\x80")
	want = []bestfit.Loss{
		{Offset: 0, Rune: utf8.RuneError, Invalid: true},
		{Offset: 1, Rune: utf8.RuneError, Invalid: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lossy(truncated) = %+v, want %+v", got, want)
	}
}
