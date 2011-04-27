package suite

import (
	"fmt"
	"strings"
	"strconv"
	"regexp"
)

const MaxConditionLen = 40 // How many charaters of the condition are displayed in a passed/failed report

// Represent a condition like "!Content-Type ~= "text/html" where Key="Content-Type"
// Op="~=", Val="text/html" and Neg=true.  For tags Op contains the number of
// occurences of the tag. Key is "Text", "Bin" or "Tag" for body-testing.
// Line contains the line number in the source
type Condition struct {
	Key  string
	Op   string
	Val  string
	Neg  bool
	Line int
}

func atoi(s string, line, fallback int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		error("Cannot convert '%s' to integer (line %d).", s, line)
		i = fallback
	}
	return i
}

func (cond *Condition) Fullfilled(v string) bool {
	ans := false
	switch cond.Op {
	case "==":
		ans = (cond.Val == v)
	case "_=":
		ans = strings.HasPrefix(v, cond.Val)
	case "=_":
		ans = strings.HasSuffix(v, cond.Val)
	case "~=":
		ans = strings.Contains(v, cond.Val)
	case ">":
		ans = (atoi(v, cond.Line, 0) > atoi(cond.Val, cond.Line, 0))
	case ">=":
		ans = (atoi(v, cond.Line, 0) >= atoi(cond.Val, cond.Line, 0))
	case "<":
		ans = (atoi(v, cond.Line, 0) < atoi(cond.Val, cond.Line, 0))
	case "<=":
		ans = (atoi(v, cond.Line, 0) >= atoi(cond.Val, cond.Line, 0))
	case "/=":
		if rexp, err := regexp.Compile(cond.Val); err != nil {
			ans = (rexp.FindStringIndex(v) != nil)
		} else {
			error("Invalid regexp in condition '%s': %s", cond.String(), err.String())
		}
	default:
		error("Condition operator '%s' (line %d) not implemented.", cond.Op, cond.Line)
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
	s += c.Key + " " + c.Op + " " + c.Val
	return
}

func (c *Condition) Info(txt string, align bool) string {
	vs := c.String()
	if i := strings.Index(vs, "\n"); i != -1 {
		vs = vs[:i]
	}

	if len(vs) > MaxConditionLen {
		vs = vs[:MaxConditionLen] + "..."
	}

	return fmt.Sprintf("%s (line %d) '%s'", txt, c.Line, vs)
}
