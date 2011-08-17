package suite

import (
	"testing"
)



func TestDequote(t *testing.T) {
	if dequote("abc") != "abc" {
		t.Error(dequote("abc"))
	}

	if dequote("abc xyz") != "abc xyz" {
		t.Error(dequote("abc xyz"))
	}

	if dequote("\"abc\"") != "abc" {
		t.Error(dequote("\"abc\""))
	}

	if dequote("\"abc xyz\"") != "abc xyz" {
		t.Error(dequote("\"abc xyz\""))
	}

}


func TestStringList(t *testing.T) {
	type slt struct {
		line string 
		fields []string
	}

	tests := []slt{slt{"abc", []string{"abc"}},
		slt{"abc xyz", []string{"abc", "xyz"}},
		slt{"abc 1 xyz", []string{"abc", "1", "xyz"}},
		slt{"abc \"1\" xyz", []string{"abc", "\"1\"", "xyz"}},
		slt{"abc \"foo\" xyz", []string{"abc", "\"foo\"", "xyz"}},
		slt{"abc \"foo bar\" xyz", []string{"abc", "foo bar", "xyz"}},
	}

	for _, test := range tests {
		f := StringList(test.line)
		if len(f) != len(test.fields) {
			t.Errorf("Cut '%s' into %#v instead of %#v. %d %d ", test.line, f, test.fields, len(f), len(test.fields))
		} else {
			for i, s := range f {
				if s != test.fields[i] {
					t.Errorf("Field %d in cut of '%s' was %s", i, test.line, s)
				}
			}
		}

	}
}