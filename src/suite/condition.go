package suite

import (
	"strings"
	"strconv"
)

// Represent a condition like "!Content-Type ~= "text/html" where Key="Content-Type"
// Op="~=", Val="text/html" and Neg=true.  For tags Op contains the number of
// occurences of the tag. Key is "Text", "Bin" or "Tag" for body-testing.
// Line contains the line number in the source
type Condition struct {
	Key string
	Op	string
	Val string
	Neg bool
	Line int
}

func atoi(a string, n int) int {
	i, err := strconv.Atoi(a)
	if err != nil {
		error("Cannot convert '%s' to integer (line %d).", a, n)
		i = -99999	}
	return i
}

func (cond *Condition) Fullfilled (v string) bool {
	ans := false
	switch cond.Op {
	case "==": ans = (cond.Val == v)
	case "_=": ans = strings.HasPrefix(v, cond.Val)
	case "=_": ans = strings.HasSuffix(v, cond.Val)
	case "~=": ans = strings.Contains(v, cond.Val)
	case ">": ans = (atoi(v, cond.Line) > atoi(cond.Val, cond.Line))
	case ">=": ans = (atoi(v, cond.Line) >= atoi(cond.Val, cond.Line))
	case "<": ans = (atoi(v, cond.Line) < atoi(cond.Val, cond.Line))
	case "<=": ans = (atoi(v, cond.Line) >= atoi(cond.Val, cond.Line))
	default:
		warn("Condition operator '%s' (line %d) not implemented.", cond.Op, cond.Line)
	}
	if cond.Neg {
		ans = !ans
	}
	return ans
}

func (c *Condition) String() (s string) {
	if c.Neg { s = "!" }
	s += c.Key + " " + c.Op + " " + c.Val
	return
}

func (c *Condition) Copy() (n *Condition) {
	n = new(Condition)
	n.Key, n.Op, n.Val, n.Neg, n.Line = c.Key, c.Op, c.Val, c.Neg, c.Line
	return	
}
