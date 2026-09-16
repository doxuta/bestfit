// Package bestfit wraps a golang.org/x/text encoding with fallback rows for
// the runes that encoding refuses, so that 〜 can be written to Shift_JIS and
// ₩ to EUC-KR instead of failing with "rune not supported by encoding".
//
// The tables live in the cp932 and cp949 subpackages; this package is the
// mechanism. A Table maps a refused rune to a string the base encoding can
// write. Tables are consulted in order before every rune reaches the base
// encoding, so they also override runes the base could have written.
package bestfit

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

// Table maps a rune to the string that is written in its place.
type Table map[rune]string

// Encoding is an encoding.Encoding: a base encoding plus fallback tables.
type Encoding struct {
	base   encoding.Encoding
	tables []Table
}

// New returns base with tables consulted, in order, for every rune before it
// reaches base. base must be stateless (Shift_JIS, EUC-KR, EUC-JP, GBK …);
// ISO-2022-JP is not.
func New(base encoding.Encoding, tables ...Table) *Encoding {
	return &Encoding{base: base, tables: tables}
}

// Lookup returns the replacement for r from the first table that has one.
func (e *Encoding) Lookup(r rune) (string, bool) {
	for _, t := range e.tables {
		if s, ok := t[r]; ok {
			return s, true
		}
	}
	return "", false
}

// NewDecoder is the base encoding's decoder; tables play no part in decoding.
func (e *Encoding) NewDecoder() *encoding.Decoder { return e.base.NewDecoder() }

// NewEncoder returns an encoder that applies the tables, then the base
// encoding. A rune neither can write fails with an *UnsupportedError.
func (e *Encoding) NewEncoder() *encoding.Encoder {
	return &encoding.Encoder{Transformer: &encoder{e: e, base: e.base.NewEncoder()}}
}

// UnsupportedError reports the first rune that no table and not the base
// encoding could write. Offset is its byte offset in the UTF-8 input.
type UnsupportedError struct {
	Rune   rune
	Offset int
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("bestfit: %q (U+%04X) at byte %d is not in the encoding", e.Rune, e.Rune, e.Offset)
}

// Loss describes one rune of an input that will not be written as itself.
type Loss struct {
	Offset int    // byte offset in the input
	Rune   rune   // the rune in the input
	To     string // what is written instead; "" when the rune is refused
}

// Lossy reports every rune of s that a table replaces or the base refuses,
// in input order, without encoding s.
func (e *Encoding) Lossy(s string) []Loss {
	var out []Loss
	enc := e.base.NewEncoder()
	for i, r := range s {
		if to, ok := e.Lookup(r); ok {
			out = append(out, Loss{Offset: i, Rune: r, To: to})
			continue
		}
		if _, err := enc.String(string(r)); err != nil {
			out = append(out, Loss{Offset: i, Rune: r})
		}
	}
	return out
}

type encoder struct {
	e    *Encoding
	base transform.Transformer
	pos  int // input bytes consumed by earlier Transform calls
}

func (t *encoder) Reset() {
	t.base.Reset()
	t.pos = 0
}

// Transform encodes src in runs: a run of runes with no table entry goes to
// the base encoder in one call; a rune with an entry is encoded from its
// replacement. The base encoder must be stateless for this to be valid.
func (t *encoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	defer func() { t.pos += nSrc }()
	for nSrc < len(src) {
		i := nSrc
		rep, repLen := "", 0
		for i < len(src) {
			r, size := utf8.DecodeRune(src[i:])
			// RuneError of size 1 is a bad or truncated byte, left to the base;
			// every other rune, ASCII included, is looked up.
			bad := r == utf8.RuneError && size == 1
			if s, ok := t.e.Lookup(r); ok && !bad {
				rep, repLen = s, size
				break
			}
			i += size
		}
		if i > nSrc {
			// A run that stops at a replacement ends in complete runes, so the
			// base may judge its trailing bytes as if at EOF.
			nd, ns, err := t.base.Transform(dst[nDst:], src[nSrc:i], atEOF || repLen > 0)
			nDst += nd
			nSrc += ns
			if err != nil {
				return nDst, nSrc, t.wrap(err, src, nSrc)
			}
		}
		if repLen == 0 {
			break
		}
		nd, _, err := t.base.Transform(dst[nDst:], []byte(rep), true)
		if err != nil {
			return nDst, nSrc, err // ErrShortDst: the caller grows dst and retries from nSrc
		}
		nDst += nd
		nSrc += repLen
	}
	return nDst, nSrc, nil
}

// wrap turns the base encoder's repertoire error into an UnsupportedError
// naming the rune at src[at:].
func (t *encoder) wrap(err error, src []byte, at int) error {
	var re interface{ Replacement() byte } // x/text's unexported RepertoireError
	if !errors.As(err, &re) {
		return err
	}
	r, size := utf8.DecodeRune(src[at:])
	if r == utf8.RuneError && size < 2 {
		return encoding.ErrInvalidUTF8
	}
	return &UnsupportedError{Rune: r, Offset: t.pos + at}
}
