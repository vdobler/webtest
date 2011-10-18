package suite

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vdobler/webtest/tag"
)

const MaxConditionLen = 45 // How many charaters of the condition are displayed in a passed/failed report

const (
	TagExpected = iota
	TagForbidden
	CountEqual
	CountNotEqual
	CountLess
	CountLessEqual
	CountGreater
	CountGreaterEqual
)

type TagCondition struct {
	Spec  tag.TagSpec
	Cond  int    // one from TagExpected, ... CountGreaterEqual		
	Count int    // used for Count... Cond only
	Id    string // used for error reporting if failed
}

func (tc *TagCondition) String() (s string) {
	switch tc.Cond {
	case TagExpected:
		// s = "  "
	case TagForbidden:
		s = "!" // "! "
	case CountEqual:
		s = fmt.Sprintf("=%d  ", tc.Count)
	case CountNotEqual:
		s = fmt.Sprintf("!=%d  ", tc.Count)
	case CountLess:
		s = fmt.Sprintf("<%d  ", tc.Count)
	case CountLessEqual:
		s = fmt.Sprintf("<=%d  ", tc.Count)
	case CountGreater:
		s = fmt.Sprintf(">%d  ", tc.Count)
	case CountGreaterEqual:
		s = fmt.Sprintf(">=%d  ", tc.Count)
	default:
		fmt.Printf("No such case: %d\n", tc.Cond)
	}

	pf := strings.Repeat(" ", len(s))
	ts := tc.Spec.String()
	if strings.Contains(ts, "\n") {
		list := strings.Split(ts, "\n")
		ts = "["
		for _, l := range list {
			ts += "\n\t\t" + l
		}
		ts += "\n\t" + pf + "]"
	}
	s += ts

	return
}

// Represent a condition like "!Content-Type ~= "text/html" where Key="Content-Type"
// Op="~=", Val="text/html" and Neg=true.  Key is "Text" or "Bin" body-testing.
// Id contains an identifier to the source
type Condition struct {
	Key   string
	Op    string
	Val   string
	Neg   bool
	Id    string
	Range Range
}

type Range struct {
	Low, High bool // which is limited?
	N, M      int  // lower limit and upper limit (
}

func (r Range) String() (s string) {
	if r.Low || r.High {
		s += "["
		if r.Low {
			s += fmt.Sprintf("%d", r.N)
		}
		s += ":"
		if r.High {
			s += fmt.Sprintf("%d", r.M)
		}
		s += "]"
	}
	return
}

// Represent a condition in a logfile
type LogCondition struct {
	Path string // path to the logfile
	Op   string // operator: ~= (contains); /= (regexp match); _= (line start); =_ (line end)
	Val  string // value/patern
	Neg  bool   // negation
	Id   string // reference to source
}

// String represnetation of condition c.
func (c *LogCondition) String() (s string) {
	if c.Neg {
		s = "!"
	}
	s += c.Path + "  " + c.Op + "  " + c.Val
	return
}

func atoi(s, line string, fallback int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		error("Cannot convert '%s' to integer (line %d).", s, line)
		i = fallback
	}
	return i
}

func toNumber(a, b, line string) (n, m int64) {
	var ae, be os.Error
	// Plain numbers
	n, ae = strconv.Atoi64(a)
	m, be = strconv.Atoi64(b)
	if ae == nil && be == nil {
		trace("Converted '%s' and '%s' to %d and %d.", a, b, n, m)
		return
	}

	// Timestamps according to RFC1123
	var at, bt *time.Time
	at, ae = time.Parse(time.RFC1123, a)
	bt, be = time.Parse(time.RFC1123, b)
	if ae == nil && be == nil {
		n, m = at.Seconds(), bt.Seconds()
		return
	}

	// Something is wrong
	error("Unable to convert both '%s' and '%s' to number with same method on line %s.", a, b, line)
	n, m = int64(len(a)), int64(len(b))
	return
}

// Retunr first n (if n>0) or last n (if n<0) of s.
func snippet(s string, n int) (snip string) {
	if n > 0 {
		if n > len(s) {
			n = len(s)
		}
		snip = s[:n]
	} else {
		n = -n
		if n > len(s) {
			n = len(s)
		}
		snip = s[len(s)-n:]
	}
	return
}

// Check whether v fullfills the condition cond.
func (cond *Condition) Fullfilled(v string) (ans bool, was string) {
	if cond.Range.Low || cond.Range.High {
		vv := strings.Split(v, "\n")
		low, high := 0, len(vv)
		if cond.Range.Low {
			low = bound(cond.Range.N, high)
		}
		if cond.Range.High {
			high = bound(cond.Range.M, high)
		}
		if high < low {
			high = low
		}
		vv = vv[low:high]
		v = strings.Join(vv, "\n")
	}

	switch cond.Op {
	case ".": // Empty operator: tests existance only.
		ans = (v != "")
		was = snippet(v, 20)
	case "==":
		ans = (cond.Val == v)
		was = v
	case "_=":
		ans = strings.HasPrefix(v, cond.Val)
		was = snippet(v, len(cond.Val))
	case "=_":
		ans = strings.HasSuffix(v, cond.Val)
		was = snippet(v, -len(cond.Val))
	case "~=":
		ans = strings.Contains(v, cond.Val)
		was = snippet(v, 10) + "[...]" + snippet(v, -10)
	case ">":
		rv, lv := toNumber(v, cond.Val, cond.Id)
		ans = rv > lv
		was = snippet(v, 10)
	case ">=":
		rv, lv := toNumber(v, cond.Val, cond.Id)
		ans = rv >= lv
		was = snippet(v, 10)
	case "<":
		rv, lv := toNumber(v, cond.Val, cond.Id)
		ans = rv < lv
		was = snippet(v, 10)
	case "<=":
		rv, lv := toNumber(v, cond.Val, cond.Id)
		ans = rv <= lv
		was = snippet(v, 10)
	case "/=":
		if rexp, err := regexp.Compile(cond.Val); err == nil {
			ans = (rexp.FindStringIndex(v) != nil)
		} else {
			error("Invalid regexp in condition '%s': %s", cond.String(), err.String())
		}
		was = snippet(v, 10) + "[...]" + snippet(v, -10)
	default:
		panic(fmt.Sprintf("Condition operator '%s' (%s) not implemented.", cond.Op, cond.Id))
	}
	if cond.Neg {
		ans = !ans
	}
	return
}

// Convert hex string (e.g. "a0 34 df 71 bc") into byte slice.
func hexToBytes(hex string) []byte {
	n := len(hex) / 2
	b := make([]byte, n, n)
	var c byte
	for i := 0; i < n; i++ {
		fmt.Sscanf(hex[2*i:2*i+2], "%x", &c) // Input sanitisation and error handling happens during parsing
		b[i] = c
	}
	supertrace("hexToBytes('%s') --> %#v", hex, b)
	return b
}

func bound(b, n int) int {
	if b >= 0 {
		if b <= n {
			return b
		}
		return n
	} else {
		if n+b <= n && n+b >= 0 {
			return n + b
		}
		return 0
	}
	return 0
}

// Check whether v fullfills the binary condition cond.
func (cond *Condition) BinFullfilled(v []byte) bool {
	ans := false
	val := hexToBytes(cond.Val)

	low, high := 0, len(v)
	if cond.Range.Low {
		low = bound(cond.Range.N, high)
	}
	if cond.Range.High {
		high = bound(cond.Range.M, high)
	}
	if high < low {
		high = low
	}
	v = v[low:high]

	switch cond.Op {
	case ".": // Empty operator: tests existance only.
		ans = (len(v) > 0)
	case "==":
		ans = (bytes.Compare(v, val) == 0)
	case "_=":
		ans = bytes.HasPrefix(v, val)
	case "=_":
		ans = bytes.HasSuffix(v, val)
	case "~=":
		ans = (bytes.Index(v, val) != -1)
	default:
		error("Condition operator '%s' (%s) not implemented.", cond.Op, cond.Id)
	}
	if cond.Neg {
		ans = !ans
	}
	return ans
}

// String represnetation of condition c.
func (c *Condition) String() (s string) {
	if c.Neg {
		s = "!"
	}
	s += c.Key
	s += c.Range.String()
	if c.Op != "." {
		s += " " + c.Op + " " + c.Val
	}
	return
}

func (c *Condition) Info(txt string) string {
	vs := c.String()
	if i := strings.Index(vs, "\n"); i != -1 {
		vs = vs[:i]
	}

	if len(vs) > MaxConditionLen {
		vs = vs[:MaxConditionLen-8-3] + "..." + vs[len(vs)-8:]
	}

	return fmt.Sprintf("%s (%s) '%s'", txt, c.Id, trim(vs))
}

func (c *TagCondition) Info(txt string) string {
	vs := strings.Replace(strings.Replace(c.String(), "\n", "\\n", -1), "\t", " ", -1)
	if len(vs) > MaxConditionLen {
		vs = vs[:MaxConditionLen-8-3] + "..." + vs[len(vs)-8:]
	}

	return fmt.Sprintf("%s (%s) '%s'", txt, c.Id, vs)
}
