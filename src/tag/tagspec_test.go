package tag

import (
	"testing"
	"fmt"
	// "strings"
)


func TestSimpleParsing(t *testing.T) {
	examples := []string{"a", "h3", "h1 class=xyz", "a href=/domain.org/path", "p == ABC xyz"}

	for _, spec := range examples {
		ts := MustParse(spec, t)
		if ts == nil {
			t.Error("Unparsabel " + spec)
			continue
		}
		if spec != ts.String() {
			t.Error("Parsed '" + ts.String() + "' != '" + spec + "'")
		}
	}

}

func TestNestedParsing(t *testing.T) {
	examples := [][2]string{[2]string{"p\n a", "p\n  a"},
		[2]string{"p\n\ta", "p\n  a"},
		[2]string{"div\n p\n p\n  span\n p", "div\n  p\n  p\n    span\n  p"},
		[2]string{"li\n div\n p class=x", "li\n  div\n  p class=x"},
		// [2]string{"", ""},
	}

	for _, spec := range examples {
		ts := MustParse(spec[0], t)
		if ts == nil {
			t.Error("Unparsabel " + spec[0])
			continue
		}
		if spec[1] != ts.String() {
			t.Error(fmt.Sprintf("Parsed '%#v' != '%#v'\n", ts.String(), spec[1]))
		}
	}
}
