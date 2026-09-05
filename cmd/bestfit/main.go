// Command bestfit converts UTF-8 on stdin to Shift_JIS or EUC-KR on stdout,
// writing fallback characters for the runes golang.org/x/text refuses.
//
//	bestfit cp932 < in.txt > out.txt
//	bestfit -check cp932 < in.txt   # list what would change or fail
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/doxuta/bestfit"
	"github.com/doxuta/bestfit/cp932"
	"github.com/doxuta/bestfit/cp949"
)

func main() {
	check := flag.Bool("check", false, "do not convert; report each rune that is replaced or refused")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: bestfit [-check] cp932|cp949 < utf8.txt > out.txt")
		flag.PrintDefaults()
	}
	flag.Parse()
	var enc *bestfit.Encoding
	switch flag.Arg(0) {
	case "cp932":
		enc = cp932.Encoding
	case "cp949":
		enc = cp949.Encoding
	default:
		flag.Usage()
		os.Exit(2)
	}
	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *check {
		for _, l := range enc.Lossy(string(in)) {
			if l.To == "" {
				fmt.Printf("byte %d: %c U+%04X refused\n", l.Offset, l.Rune, l.Rune)
			} else {
				fmt.Printf("byte %d: %c U+%04X -> %s\n", l.Offset, l.Rune, l.Rune, l.To)
			}
		}
		return
	}
	out, err := enc.NewEncoder().Bytes(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout.Write(out)
}
