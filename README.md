# bestfit

Shift_JIS and EUC-KR encoders for Go that write 〜 and ₩ instead of failing.

```
$ echo '東京〜大阪 ¥100' | bestfit cp932 | xxd
00000000: 938c 8b9e 8160 91e5 8de3 205c 3130 300a  .....`.... \100.

$ echo '₩12,000 ㈱' | bestfit cp949 | xxd
00000000: 5c31 322c 3030 3020 f1bb 0a              \12,000 ...

$ printf 'café〜😀' | bestfit -check cp932
byte 3: é U+00E9 -> e
byte 5: 〜 U+301C -> ～
byte 8: 😀 U+1F600 refused
```

`golang.org/x/text/encoding/japanese.ShiftJIS` refuses all three of 〜 (U+301C
WAVE DASH), − (U+2212 MINUS SIGN) and ¥ (U+00A5) with
`encoding: rune not supported by encoding`; `korean.EUCKR` refuses ₩ (U+20A9
WON SIGN). Those are the characters a Mac, a Linux box or a Japanese IME hands
you. This package puts the fallback rows that other implementations use in
front of x/text's encoder, each row traceable to the file it came from, and
names the rune and byte offset when nothing applies.

## Why

[golang/go#69934](https://github.com/golang/go/issues/69934) (open since
October 2024) asks x/text to replace "visually similar" characters before
encoding to Shift_JIS. Rob Pike's answer: "there should already be an official
defining table for how to handle the translation. Go's implementation should
not be the one to codify it." mattn's: Japanese Windows still passes Shift_JIS
file names through some code paths and "we want to use fallback characters".

There are official tables. There are three of them, they disagree, and none
covers everything the proposer asked for:

| Input | x/text | Python `cp932` | iconv `CP932` | iconv `SHIFT_JIS` | Windows (`bestfit932.txt`) | **bestfit** |
|---|---|---|---|---|---|---|
| 〜 U+301C | error | `81 60` | error | `81 60` | `?` | `81 60` |
| − U+2212 | error | `81 7C` | error | `81 7C` | `?` | `81 7C` |
| ¥ U+00A5 | error | error | error | `5C` | `5C` | `5C` |
| é U+00E9 | error | error | error | error | `65` (e) | `65` |
| ₩ U+20A9 → CP949 | error | error | error | — | `5C` | `5C` |

(Measured on macOS 26 with x/text v0.41.0, Python 3 and libiconv; the Windows
column is the table, not a Windows machine.)

## Tables

Every row is generated from a vendor file on unicode.org by `internal/gen`,
verified against x/text at generation time (the replacement must encode to
exactly the bytes the vendor file gives), and committed. `go generate ./...`
regenerates them; it needs the network.

**`cp932.JIS` — 8 rows.** Unicode's
[`SHIFTJIS.TXT`](https://www.unicode.org/Public/MAPPINGS/OBSOLETE/EASTASIA/JIS/SHIFTJIS.TXT)
(the JIS X 0208 mapping used by Apple, glibc and iconv) and Microsoft's
[`CP932.TXT`](https://www.unicode.org/Public/MAPPINGS/VENDORS/MICSFT/WINDOWS/CP932.TXT)
disagree on exactly nine byte positions. x/text follows Microsoft (through the
[WHATWG index](https://encoding.spec.whatwg.org/#index-jis0208)), so the JIS
side's code points become unwritable. This table writes them to the same bytes
the JIS side reads them from:

| U+ | JIS side | bytes | Microsoft side |
|---|---|---|---|
| 00A5 | ¥ | `5C` | \ |
| 203E | ‾ | `7E` | ~ |
| 301C | 〜 | `81 60` | ～ U+FF5E |
| 2016 | ‖ | `81 61` | ∥ U+2225 |
| 2212 | − | `81 7C` | － U+FF0D |
| 00A2 | ¢ | `81 91` | ￠ U+FFE0 |
| 00A3 | £ | `81 92` | ￡ U+FFE1 |
| 00AC | ¬ | `81 CA` | ￢ U+FFE2 |

The ninth position, `81 5F` (\ vs ＼), needs no row because x/text writes
U+005C as ASCII. The
[WHATWG Shift_JIS encoder](https://encoding.spec.whatwg.org/#shift_jis-encoder)
special-cases three of these (¥→`5C`, ‾→`7E`, −→－); x/text implements none of
the three, so this table is also the missing WHATWG behaviour.

**`cp932.Windows` — 84 rows** and **`cp949.Windows` — 394 rows.** What
`WideCharToMultiByte` does when `WC_NO_BEST_FIT_CHARS` is *not* set, from
Microsoft's
[`bestfit932.txt`](https://www.unicode.org/Public/MAPPINGS/VENDORS/MICSFT/WindowsBestFit/bestfit932.txt)
and
[`bestfit949.txt`](https://www.unicode.org/Public/MAPPINGS/VENDORS/MICSFT/WindowsBestFit/bestfit949.txt):
only the WCTABLE rows that do not round-trip. For 932 that is Latin-1 losing
its diacritics (é→e, Æ→A, ©→c, ®→R, ¦→|). For 949 it is ₩→`5C` (Korean fonts
draw the backslash cell as ₩, which is why the Korean keyboard's ₩ key types
U+005C), ㈱→株, ©→ⓒ, 52 halfwidth/fullwidth forms, and Cyrillic and Greek
lookalikes. These rows *lose information*; they are second in line after `JIS`
and you can leave them out.

`KSC5601.TXT` and `CP949.TXT` agree on every byte, so there is no Korean
equivalent of the `JIS` tier.

## Install

```
go install github.com/doxuta/bestfit/cmd/bestfit@latest
```

```
bestfit cp932 < utf8.txt > sjis.txt
bestfit -check cp949 < utf8.txt   # what would change or fail, nothing written
```

Exit status 1 and a message naming the rune and byte offset when a rune has
no row and the base encoding refuses it.

## Library

```go
import (
    "github.com/doxuta/bestfit"
    "github.com/doxuta/bestfit/cp932"
)

out, err := cp932.Encoding.NewEncoder().String("東京〜大阪")   // 93 8C 8B 9E 81 60 91 E5 8D E3
var ue *bestfit.UnsupportedError
if errors.As(err, &ue) {
    fmt.Println(ue.Rune, ue.Offset)                            // the first rune nothing can write
}
for _, l := range cp932.Encoding.Lossy("café〜😀") {           // audit before writing
    fmt.Println(l.Offset, l.Rune, l.To)                        // 3 é "e" / 5 〜 "～" / 8 😀 ""
}
```

`cp932.Encoding` and `cp949.Encoding` are `encoding.Encoding`s, so they work
with `transform.NewWriter`, `transform.NewReader` and everything else that
takes one. Decoding is unchanged from x/text.

Tables are `map[rune]string` and are consulted in order, in front of the base
encoding, so your own rows come first:

```go
mine := bestfit.Table{'—': "―", '•': "・"}
e := bestfit.New(japanese.ShiftJIS, mine, cp932.JIS, cp932.Windows)
```

`bestfit.New` accepts any stateless x/text encoding (EUC-JP, GBK, Big5 …).

## Limitations

- **Lossy by design.** The `JIS` rows change code points, not meaning. The
  `Windows` rows drop diacritics and merge symbols. Run `Lossy` first if the
  bytes go somewhere that matters, or use only the tier you accept.
- **— U+2014 and • U+2022 have no row**, because no vendor table has one
  (Windows writes `?`). Add your own `Table` in front, as above. … U+2026 needs
  no row: x/text writes it as `81 63` in Shift_JIS and `A1 A6` in EUC-KR, the
  same bytes `bestfit932.txt` and `bestfit949.txt` give it.
- **Not a Windows emulation.** Windows writes `?` for 〜 and −; this package
  writes `81 60` and `81 7C`. If you need bytes identical to
  `WideCharToMultiByte`, use `cp932.Windows` alone.
- **Decoding is not touched.** `81 60` still decodes to ～ U+FF5E, as in x/text
  and every browser. Round-tripping 〜 through this package gives you ～.
- **The base encoding must be stateless.** ISO-2022-JP switches modes between
  runs and would produce wrong escapes; it is not supported.
- Tables are as of the unicode.org files on 2026-09-05. Rows x/text can
  already write are dropped at generation time (one in `bestfit932.txt`: U+6E38
  游, an IBM-extension duplicate).

## Tests

`go test -race ./...`: the 8 `JIS` rows against the bytes in `SHIFTJIS.TXT`,
the counts 84 and 394, the ₩/㈱/©/¥ rows of CP949, every replacement in every
table encodable by the base, output identical to x/text for text that needs no
fallback, the proposal's own example, the rune and byte offset of an
`UnsupportedError` across chunked `Transform` calls, progress under
`ErrShortDst`, byte-at-a-time reading, invalid UTF-8 before and after a replaced
rune, and — by fuzzing — that every error is either an `UnsupportedError`
pointing at a rune the base refuses or `ErrInvalidUTF8`, and that every
successful output decodes.

## Prior art and sources

- [`nyaosorg/go-windows-mbcs`](https://github.com/nyaosorg/go-windows-mbcs)
  (hymkor) calls `WideCharToMultiByte` and gets Windows' best fit on Windows;
  this package is the same table without the OS, plus the JIS rows Windows lacks.
- [`akihiroy/pytextcodec`](https://github.com/akihiroy/pytextcodec) ports
  Python's `cp932`, which writes 〜 but refuses ¥.
- `encoding.ReplaceUnsupported` in x/text writes a fixed `?`.
- [golang/go#69934](https://github.com/golang/go/issues/69934) by yuki2006,
  with Rob Pike's and mattn's comments;
  [`nao1215/filesql#933`](https://github.com/nao1215/filesql/issues/933) for the
  error-without-a-position problem.
- Microsoft's WindowsBestFit tables and the JIS/KSC/CP932/CP949 mappings are
  hosted by the Unicode Consortium at
  [unicode.org/Public/MAPPINGS](https://www.unicode.org/Public/MAPPINGS/).

Built with an AI coding agent (Claude) under human review, as with the other
repositories on this account. The tables are generated, not typed; the
generator refuses any row it cannot verify against x/text.

## 日本語要約

`golang.org/x/text` の Shift_JIS エンコーダは 〜（U+301C 波ダッシュ）・−（U+2212）・
¥（U+00A5）を `rune not supported` で拒否します。このパッケージは、Unicode
コンソーシアムの `SHIFTJIS.TXT` と Microsoft の `CP932.TXT` が食い違う 9 バイト位置
（いわゆる波ダッシュ問題）から 8 行、`bestfit932.txt`（`WideCharToMultiByte` の
ベストフィット）から 84 行を生成し、x/text の前段に置きます。行を持たない文字
（— や • など）は文字と位置を添えたエラーになります。`Lossy` で書き換わる文字を
事前に一覧できます。デコードは x/text のままです。

## 한국어 요약

`golang.org/x/text`의 EUC-KR 인코더는 ₩(U+20A9 원화 기호)를 쓰지 못합니다.
Windows는 `bestfit949.txt`에 따라 백슬래시(0x5C)로 기록하고, 한글 글꼴은 그 칸을
₩로 그립니다. 이 패키지는 그 표에서 왕복되지 않는 394행(₩→\, ㈱→株, ©→ⓒ 등)을
생성해 x/text 앞에 두고, 표에 없는 문자는 문자와 바이트 위치를 담은 오류로
돌려줍니다. `KSC5601.TXT`와 `CP949.TXT`는 모든 바이트에서 일치하므로 일본어의
JIS 계층에 해당하는 표는 없습니다.

## License

MIT © Xuan Tai Doan
