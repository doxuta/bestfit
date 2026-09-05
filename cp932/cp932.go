// Package cp932 is Shift_JIS (Windows code page 932) for text that
// golang.org/x/text/encoding/japanese.ShiftJIS refuses: 〜 ‖ − ¢ £ ¬ ¥ ‾ from
// the JIS X 0208 vendor mapping, and Microsoft's best-fit rows such as é→e.
package cp932

import (
	"github.com/doxuta/bestfit"
	"golang.org/x/text/encoding/japanese"
)

//go:generate go run ../internal/gen -cp 932 -o tables_gen.go

// Encoding is Shift_JIS with JIS consulted first, then Windows. Decoding is
// unchanged from japanese.ShiftJIS.
var Encoding = bestfit.New(japanese.ShiftJIS, JIS, Windows)
