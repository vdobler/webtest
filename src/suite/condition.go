package suite

import (
	"os"
	"time"
	"fmt"
	"strings"
	"strconv"
	"regexp"
	"dobler/webtest/tag"
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
	ts := tc.Spec.String()
	if strings.Contains(ts, "\n") {
		list := strings.Split(ts, "\n", -1)
		ts = "["
		for _, l := range list {
			ts += "\n\t\t\t" + l
		}
		ts += "\n\t\t]"
	}
	switch tc.Cond {
	case TagExpected:
		s = "\t" + ts
	case TagForbidden:
		s = "!\t" + ts
	case CountEqual:
		s = fmt.Sprintf("=%d\t%s", tc.Count, ts)
	case CountNotEqual:
		s = fmt.Sprintf("!=%d\t%s", tc.Count, ts)
	case CountLess:
		s = fmt.Sprintf("<%d\t%s", tc.Count, ts)
	case CountLessEqual:
		s = fmt.Sprintf("<=%d\t%s", tc.Count, ts)
	case CountGreater:
		s = fmt.Sprintf(">%d\t%s", tc.Count, ts)
	case CountGreaterEqual:
		s = fmt.Sprintf(">=%d\t%s", tc.Count, ts)
	default:
		fmt.Printf("No such case: %d\n", tc.Cond)
	}
	return
}


// Represent a condition like "!Content-Type ~= "text/html" where Key="Content-Type"
// Op="~=", Val="text/html" and Neg=true.  For tags Op contains the number of
// occurences of the tag. Key is "Text", "Bin" or "Tag" for body-testing.
// Line contains the line number in the source
type Condition struct {
	Key string
	Op  string
	Val string
	Neg bool
	Id  string
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
		trace("Converted '%s' and '%s' to %d and %d.", a,b,n,m)
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

func (cond *Condition) Fullfilled(v string) bool {
	ans := false
	switch cond.Op {
	case ".": // Empty operator: tests existance only.
		ans = (v != "")
	case "==":
		ans = (cond.Val == v)
	case "_=":
		ans = strings.HasPrefix(v, cond.Val)
	case "=_":
		ans = strings.HasSuffix(v, cond.Val)
	case "~=":
		ans = strings.Contains(v, cond.Val)
	case ">":
		rv, lv := toNumber(v, cond.Val, cond.Id)
		ans = rv > lv
	case ">=":
		rv, lv := toNumber(v, cond.Val, cond.Id)
		ans = rv >= lv
	case "<":
		rv, lv := toNumber(v, cond.Val, cond.Id)
		ans = rv < lv
	case "<=":
		rv, lv := toNumber(v, cond.Val, cond.Id)
		ans = rv <= lv
	case "/=":
		if rexp, err := regexp.Compile(cond.Val); err == nil {
			ans = (rexp.FindStringIndex(v) != nil)
		} else {
			error("Invalid regexp in condition '%s': %s", cond.String(), err.String())
		}
	default:
		error("Condition operator '%s' (%s) not implemented.", cond.Op, cond.Id)
	}
	if cond.Neg {
		ans = !ans
	}
	return ans
}

func (c *Condition) String() (s string) {
	if c.Neg {
		s = "!"
	} else {
		s = " "
	}
	s += c.Key
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
		vs = vs[:MaxConditionLen] + "..."
	}

	return fmt.Sprintf("%s (%s) '%s'", txt, c.Id, vs)
}

func (c *TagCondition) Info(txt string) string {
	vs := strings.Replace(strings.Replace(c.String(), "\n", "\\n", -1), "\t", " ", -1)
	if len(vs) > MaxConditionLen {
		vs = vs[:MaxConditionLen] + "..."
	}

	return fmt.Sprintf("%s (%s) '%s'", txt, c.Id, vs)
}
