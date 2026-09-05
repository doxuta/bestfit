package bestfit_test

import (
	"errors"
	"fmt"

	"github.com/doxuta/bestfit"
	"github.com/doxuta/bestfit/cp932"
	"github.com/doxuta/bestfit/cp949"
	"golang.org/x/text/encoding/japanese"
)

func Example() {
	_, err := japanese.ShiftJIS.NewEncoder().String("東京〜大阪")
	fmt.Println("x/text:", err)

	out, _ := cp932.Encoding.NewEncoder().String("東京〜大阪")
	fmt.Printf("bestfit: % X\n", out)

	won, _ := cp949.Encoding.NewEncoder().String("₩12,000")
	fmt.Printf("cp949: %q\n", won)
	// Output:
	// x/text: encoding: rune not supported by encoding.
	// bestfit: 93 8C 8B 9E 81 60 91 E5 8D E3
	// cp949: "\\12,000"
}

func ExampleUnsupportedError() {
	_, err := cp932.Encoding.NewEncoder().String("価格〜😀")
	var ue *bestfit.UnsupportedError
	if errors.As(err, &ue) {
		fmt.Printf("%c at byte %d\n", ue.Rune, ue.Offset)
	}
	// Output: 😀 at byte 9
}

func ExampleEncoding_Lossy() {
	for _, l := range cp932.Encoding.Lossy("café〜😀") {
		fmt.Printf("byte %d: %c -> %q\n", l.Offset, l.Rune, l.To)
	}
	// Output:
	// byte 3: é -> "e"
	// byte 5: 〜 -> "～"
	// byte 8: 😀 -> ""
}

func ExampleTable() {
	// Add your own rows in front of the shipped ones.
	mine := bestfit.Table{'—': "―", '•': "・"}
	e := bestfit.New(japanese.ShiftJIS, mine, cp932.JIS, cp932.Windows)
	out, _ := e.NewEncoder().String("A—B•C")
	fmt.Printf("% X\n", out)
	// Output: 41 81 5C 42 81 45 43
}
