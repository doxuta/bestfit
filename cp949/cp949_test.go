package cp949_test

import (
	"bytes"
	"testing"

	"github.com/doxuta/bestfit/cp949"
	"golang.org/x/text/encoding/korean"
)

func TestWindowsRows(t *testing.T) {
	if len(cp949.Windows) != 394 {
		t.Fatalf("Windows has %d rows, want 394", len(cp949.Windows))
	}
	cases := []struct {
		r    rune
		want []byte
	}{
		{'₩', []byte{0x5C}},       // the won sign is a backslash on Korean Windows
		{'㈱', []byte{0xF1, 0xBB}}, // 株
		{'©', []byte{0xA8, 0xCF}}, // ⓒ
		{'¥', []byte{0xA1, 0xCD}}, // ￥
		{'é', []byte{'e'}},
	}
	for _, c := range cases {
		if _, err := korean.EUCKR.NewEncoder().Bytes([]byte(string(c.r))); err == nil {
			t.Errorf("%q: x/text now writes this itself; the row is dead", c.r)
		}
		got, err := cp949.Encoding.NewEncoder().Bytes([]byte(string(c.r)))
		if err != nil || !bytes.Equal(got, c.want) {
			t.Errorf("%q: got % X, %v; want % X", c.r, got, err, c.want)
		}
	}
	for r, to := range cp949.Windows {
		if _, err := korean.EUCKR.NewEncoder().Bytes([]byte(to)); err != nil {
			t.Errorf("%q -> %q: replacement is not encodable", r, to)
		}
	}
}

func TestPlainTextUnchanged(t *testing.T) {
	in := "금액: ￦12,000 (부가세 별도) 똠방각하 ㈜한글"
	got, err := cp949.Encoding.NewEncoder().Bytes([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	want, err := korean.EUCKR.NewEncoder().Bytes([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("output differs from x/text for text with no fallback")
	}
}
