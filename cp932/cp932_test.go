package cp932_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/doxuta/bestfit"
	"github.com/doxuta/bestfit/cp932"
	"golang.org/x/text/encoding/japanese"
)

// The nine byte positions where SHIFTJIS.TXT and CP932.TXT disagree, minus
// 0x815F (U+005C, which x/text writes as ASCII 0x5C anyway).
func TestJISRows(t *testing.T) {
	cases := []struct {
		r    rune
		want []byte
	}{
		{'¥', []byte{0x5C}},
		{'‾', []byte{0x7E}},
		{'〜', []byte{0x81, 0x60}},
		{'‖', []byte{0x81, 0x61}},
		{'−', []byte{0x81, 0x7C}},
		{'¢', []byte{0x81, 0x91}},
		{'£', []byte{0x81, 0x92}},
		{'¬', []byte{0x81, 0xCA}},
	}
	if len(cp932.JIS) != len(cases) {
		t.Fatalf("JIS has %d rows, want %d", len(cp932.JIS), len(cases))
	}
	for _, c := range cases {
		if _, err := japanese.ShiftJIS.NewEncoder().Bytes([]byte(string(c.r))); err == nil {
			t.Errorf("%q: x/text now writes this itself; the row is dead", c.r)
		}
		got, err := cp932.Encoding.NewEncoder().Bytes([]byte(string(c.r)))
		if err != nil || !bytes.Equal(got, c.want) {
			t.Errorf("%q: got % X, %v; want % X", c.r, got, err, c.want)
		}
	}
}

func TestWindowsRows(t *testing.T) {
	if len(cp932.Windows) != 84 {
		t.Fatalf("Windows has %d rows, want 84", len(cp932.Windows))
	}
	for r, want := range map[rune]string{'é': "e", '©': "c", '®': "R", 'Æ': "A", 'ß': "s", 'µ': "μ"} {
		if got := cp932.Windows[r]; got != want {
			t.Errorf("%q: got %q want %q", r, got, want)
		}
	}
	for _, tbl := range []bestfit.Table{cp932.JIS, cp932.Windows} {
		for r, to := range tbl {
			if _, err := japanese.ShiftJIS.NewEncoder().Bytes([]byte(to)); err != nil {
				t.Errorf("%q -> %q: replacement is not encodable", r, to)
			}
		}
	}
}

// The example from golang/go#69934.
func TestProposalExample(t *testing.T) {
	got, err := cp932.Encoding.NewEncoder().Bytes([]byte("東京〜大阪 −5℃ —"))
	if err == nil {
		t.Fatalf("— has no row and should be refused, got % X", got)
	}
	var ue *bestfit.UnsupportedError
	if !errors.As(err, &ue) || ue.Rune != '—' {
		t.Fatalf("err = %v", err)
	}
	got, err = bestfit.New(japanese.ShiftJIS, cp932.JIS, cp932.Windows, bestfit.Table{'—': "―"}).
		NewEncoder().Bytes([]byte("東京〜大阪 −5℃ —"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := japanese.ShiftJIS.NewEncoder().Bytes([]byte("東京～大阪 －5℃ ―"))
	if !bytes.Equal(got, want) {
		t.Fatalf("got % X want % X", got, want)
	}
}

func TestPlainTextUnchanged(t *testing.T) {
	in := "吾輩は猫である。名前はまだ無い。ｶﾞ ｷﾞ ㈱ ① Ⅳ 髙 﨑"
	got, err := cp932.Encoding.NewEncoder().Bytes([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	want, err := japanese.ShiftJIS.NewEncoder().Bytes([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("output differs from x/text for text with no fallback")
	}
}
