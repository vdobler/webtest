package suite

import (
	"fmt"
	"testing"
)


func pr(s string) Range {
	r, err := parseRange(s)
	if err != nil {
		fmt.Printf("%s unparsable: %s\n", s, err.String())
	}
	return r
}

func TestRangedParsing(t *testing.T) {
	r, err := parseRange("[2:5]")
	if err != nil || !r.Low || !r.High || r.N != 2 || r.M != 5 {
		t.Errorf("[2:5]")
	}
	r, err = parseRange("[234:567]")
	if err != nil || !r.Low || !r.High || r.N != 234 || r.M != 567 {
		t.Errorf("[234:567]")
	}
	r, err = parseRange("[:5]")
	if err != nil || r.Low || !r.High || r.M != 5 {
		t.Errorf("[:5]")
	}
	r, err = parseRange("[2:]")
	if err != nil || !r.Low || r.High || r.N != 2 {
		t.Errorf("[2:]")
	}
	r, err = parseRange("[:]")
	if err != nil || r.Low || r.High {
		t.Errorf("[:]")
	}
	r, err = parseRange("")
	if err != nil || r.Low || r.High {
		t.Errorf("")
	}

	r, err = parseRange("[2:5")
	if err == nil {
		t.Errorf("[2:5")
	}
	r, err = parseRange("2:5")
	if err == nil {
		t.Errorf("2:5")
	}
	r, err = parseRange("2:5]")
	if err == nil {
		t.Errorf("2:5]")
	}
	r, err = parseRange("[2:5:7]")
	if err == nil {
		t.Errorf("[2:5:7]")
	}
	r, err = parseRange("[2:x]")
	if err == nil {
		t.Errorf("[2:x]")
	}
}


func TestRangedCondition(t *testing.T) {
	var s = `This is a
multilined
test text
for internal use.`

	for _, c := range []Condition{Condition{Op: "~=", Val: "This", Range: pr("")},
		Condition{Op: "~=", Val: "test", Range: pr("[2:]")},
		Condition{Op: "~=", Val: "multilined", Range: pr("[:-2]")},
		Condition{Op: "=_", Val: "text", Range: pr("[2:3]")},
		Condition{Op: "=_", Val: "test text", Range: pr("[1:-1]")},
		Condition{Op: "=_", Val: "xxx", Neg: true, Range: pr("[7:3]")},
		Condition{Op: "=_", Val: "xxx", Neg: true, Range: pr("[7:6]")},
		Condition{Op: "=_", Val: "xxx", Neg: true, Range: pr("[-7:30]")},
	} {
		if !(&c).Fullfilled(s) {
			t.Errorf("Condition %s did not match", c.String())
		}
	}
}
