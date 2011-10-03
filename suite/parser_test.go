package suite

import (
	"fmt"
	"strings"
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

func TestSuiteParsing(t *testing.T) {
	LogLevel = 3
	suites := []string{
		`# Minimal suite
---------------------
Minimal Suite
---------------------
GET http://localhost:54123/
`,
		`# All Sections
---------------------
All Sections
---------------------
GET http://localhost:54123/
CONST
	a b
RAND
	c d
SEQ
	d e
HEADER
	MyHeader  abc
RESPONSE
	Status-Code  == 200
BODY
	Txt  ~=  Hallo
TAG
	a href  ==  Hallo
SEND-COOKIE
	a:localhost:/:Secure  := b
SET-COOKIE
	a:localhost:/MaxAge  > 5
SETTING
	Sleep 1
BEFORE
	bash -c echo
AFTER
	bash -c echo
`,
		`# Variants
---------------------
Variants
---------------------
GET http://localhost:54123/
CONST
	a b
	bbb ccc
	xy foo bar
	z foo "bar" baz
	zz  " foo "
SEQ
	c d
	d e fff 123 "Hallo" "Hallo Welt" xyz
HEADER
	MyHeader  abc
RESPONSE
	A  == 200
	B  /= 200
	C  ~= 200
	D  _= 200
	E  =_ 200
	!A  == 200
	!B  /= 200
	!C  ~= 200
	!D  _= 200
	!E  =_ 200
	! A  == 200
	! B  /= 200
	! C  ~= 200
	! D  _= 200
	! E  =_ 200
BODY
	Txt  ~=  Hallo
	Txt[1:2]  ~=  Hallo
	Txt[999:888]  ~=  Hallo
	Txt[-12:-45]  ~=  Hallo
	Txt[999:]  ~=  Hallo
	Txt[:-45]  ~=  Hallo
	!Txt  ~=  Hallo
	!Txt[1:2]  ~=  Hallo
	!Txt[999:888]  ~=  Hallo
	!Txt[-12:-45]  ~=  Hallo
	!Txt[999:]  ~=  Hallo
	!Txt[:-45]  ~=  Hallo
	! Txt  ~=  Hallo
	! Txt[1:2]  ~=  Hallo
	! Txt[999:888]  ~=  Hallo
	! Txt[-12:-45]  ~=  Hallo
	! Txt[999:]  ~=  Hallo
	! Txt[:-45]  ~=  Hallo
TAG
	a href  ==  Hallo
	=3 a href  ==  Hallo
	>3 a href  ==  Hallo
	<3 a href  ==  Hallo
	>=3 a href  ==  Hallo
	<=3 a href  ==  Hallo
	=321 a href  ==  Hallo
	>321 a href  ==  Hallo
	<321 a href  ==  Hallo
	>=321 a href  ==  Hallo
	<=321 a href  ==  Hallo

	!a href  ==  Hallo
	!=3 a href  ==  Hallo
	!>3 a href  ==  Hallo
	!<3 a href  ==  Hallo
	!>=3 a href  ==  Hallo
	!<=3 a href  ==  Hallo
	!=321 a href  ==  Hallo
	!>321 a href  ==  Hallo
	!<321 a href  ==  Hallo
	!>=321 a href  ==  Hallo
	!<=321 a href  ==  Hallo

	! a href  ==  Hallo
	! =3 a href  ==  Hallo
	! >3 a href  ==  Hallo
	! <3 a href  ==  Hallo
	! >=3 a href  ==  Hallo
	! <=3 a href  ==  Hallo
	! =321 a href  ==  Hallo
	! >321 a href  ==  Hallo
	! <321 a href  ==  Hallo
	! >=321 a href  ==  Hallo
	! <=321 a href  ==  Hallo

	[
		div class=teaser
			h1 !class=buggy == *Title*
	]
	![
		div class=teaser
			h1 !class=buggy == *Title*
	]
	! [
		div class=teaser
			h1 !class=buggy == *Title*
	]
	=5[
		div class=teaser
			h1 !class=buggy == *Title*
	]
	=5 [
		div class=teaser
			h1 !class=buggy == *Title*
	]
	!=5[
		div class=teaser
			h1 !class=buggy == *Title*
	]
	!=5 [
		div class=teaser
			h1 !class=buggy == *Title*
	]
	! =5 [
		div class=teaser
			h1 !class=buggy == *Title*
	]

SEND-COOKIE
	a:localhost:/:Secure  :=  b
	a:localhost:/:Secure  :=  berta und emil
	a:localhost:/:Secure  :=  " berta und emil "
	name  :=  b
	name:domain.org  :=  b
	name:/some/path  :=  b

SET-COOKIE
	a:localhost:/MaxAge  > 5
SETTING
	Sleep 1
BEFORE
	bash -c echo
	bash -c "echo Hallo Welt > _file"
`,
	}

	for i, s := range suites {
		// Test initial parsing
		p := NewParser(strings.NewReader(s), fmt.Sprintf("Suite %d", i))
		suite, err := p.ReadSuite()
		if err != nil {
			t.Errorf("Cannot parse suite %d: %s", i, err.String())
			continue
		}

		// Parsing, string(), re-parsing, string() should be idempotent
		st := suite.Test[0].String()
		p = NewParser(strings.NewReader(st), fmt.Sprintf("Suite %d reparsed", i))
		suite, err = p.ReadSuite()
		if err != nil {
			t.Errorf("Cannot re-parse suite %d: %s", i, err.String())
			continue
		}
		stt := suite.Test[0].String()
		if stt != st {
			t.Errorf("Parsing of suite %d not idempotent.", i)
			fmt.Printf("1.Pass:\n%s\n2. Pass:\n%s\n", st, stt)
		}
	}

}
