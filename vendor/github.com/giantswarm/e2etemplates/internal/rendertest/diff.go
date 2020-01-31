package rendertest

import (
	"fmt"
	"strings"
	"unicode"
)

func Diff(a, b string) (int, string) {
	aSplit := strings.Split(a, "\n")
	bSplit := strings.Split(b, "\n")

	length := len(aSplit)
	if len(bSplit) < length {
		length = len(bSplit)
	}

	for i := 0; i < length; i++ {
		a := strings.TrimRightFunc(aSplit[i], unicode.IsSpace)
		b := strings.TrimRightFunc(bSplit[i], unicode.IsSpace)
		if a != b {
			return i + 1, fmt.Sprintf("a: %q b: %q", a, b)
		}
	}

	if len(aSplit) > len(bSplit) {
		return len(bSplit) + 1, fmt.Sprintf("a: %q b: EOF", aSplit[len(bSplit)])
	}
	if len(aSplit) < len(bSplit) {
		return len(aSplit) + 1, fmt.Sprintf("a: EOF b: %q", bSplit[len(aSplit)])
	}

	return 0, ""
}
