// Package cp949 is EUC-KR (Windows code page 949, Unified Hangul Code) for
// text that golang.org/x/text/encoding/korean.EUCKR refuses: ₩ U+20A9 and
// Microsoft's other best-fit rows such as ㈱→株 and ©→ⓒ.
package cp949

import (
	"github.com/doxuta/bestfit"
	"golang.org/x/text/encoding/korean"
)

//go:generate go run ../internal/gen -cp 949 -o tables_gen.go

// Encoding is EUC-KR with Windows consulted first. Decoding is unchanged from
// korean.EUCKR.
var Encoding = bestfit.New(korean.EUCKR, Windows)
