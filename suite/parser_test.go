package suite

import (
	"testing"
)


func TestDequote(t *testing.T) {
	tests := [][]string{[]string{"abc", "abc"},
		[]string{"abc xyz", "abc xyz"},
		[]string{"\"abc\"", "abc"},
		[]string{"\"abc\"", "abc"},
		[]string{"\"abc xyz\"", "abc xyz"},
		[]string{"\"a\\x18b\"", "a\x18b"},
		[]string{"\"a\\tb\"", "a\tb"},
		[]string{"a\\tb", "a\\tb"},
	}
	for _, test := range tests {
		o, e := test[0], test[1]
		s, err := dequote(o)
		if err != nil {
			t.Errorf("Unexpected error: %s", err.String())
		} else if s != e {
			t.Errorf("dequote '%s' yielded '%s', expected '%s'", o, s, e)
		}
	}
}


func TestEndQuoteIndex(t *testing.T) {
	type eqt struct {
		line string
		n    int
	}

	tests := []eqt{eqt{"\"foo\" bar", 4},
		eqt{"\"foo\"", 4},
		eqt{"\"foo\"bar\" x", 8},
		eqt{"\"foo\\\" bar\"", 10},
	}
	for _, test := range tests {
		if i := endQuoteIndex(test.line); i != test.n {
			t.Errorf("Index of quote in '%s' was %d insted of %d.", test.line, i, test.n)
		}
	}

}


//  foo bar baz  -->  [foo, bar, baz]
//  foo "bar" baz --> [foo, bar, baz]
//  foo "bar baz" --> [foo, bar baz]
//  foo "bar\" baz\"" --> [foo, bar "baz"]

func TestStringList(t *testing.T) {
	type slt struct {
		line   string
		fields []string
	}

	tests := []slt{slt{"abc", []string{"abc"}},
		slt{"abc xyz", []string{"abc", "xyz"}},
		slt{"abc   xyz", []string{"abc", "xyz"}},
		slt{"abc   xyz ", []string{"abc", "xyz"}},
		slt{"abc   xyz   ", []string{"abc", "xyz"}},
		slt{"abc foo xyz", []string{"abc", "foo", "xyz"}},
		slt{"abc  foo   xyz", []string{"abc", "foo", "xyz"}},
		slt{"abc \"foo\" xyz", []string{"abc", "foo", "xyz"}},
		slt{"abc  \"foo\"   xyz", []string{"abc", "foo", "xyz"}},
		slt{"\"abc\" xyz", []string{"abc", "xyz"}},
		slt{"\"abc\"", []string{"abc"}},
		slt{"\"abc foo\" xyz", []string{"abc foo", "xyz"}},
		slt{"\"abc foo\"", []string{"abc foo"}},
		slt{"\"abc foo\" ", []string{"abc foo"}},
		slt{"\"a\\\"c\"", []string{"a\"c"}},
		slt{"\"ab\\\" c\"", []string{"ab\" c"}},
		slt{"\"foo\\tbar\"", []string{"foo\tbar"}},
	}

	for _, test := range tests {
		f, err := StringList(test.line)
		if err != nil {
			t.Errorf("Unexpected error: %s", err.String())
		} else {
			if len(f) != len(test.fields) {
				t.Errorf("Cut '%s' into %#v instead of %#v. %d %d ",
					test.line, f, test.fields, len(f), len(test.fields))
			} else {
				for i, s := range f {
					if s != test.fields[i] {
						t.Errorf("Field %d in cut of '%s' was '%s' expected '%s'",
							i, test.line, s, test.fields[i])
					}
				}
			}
		}
	}
}
