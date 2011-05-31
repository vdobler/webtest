package tag

import (
	"os"
	"strings"
	"regexp"
)

// Interface for content values of tags and attributes.
// A nil content measn ignored value of content and can be distinguisehd from
// an empty content.
type Content interface {
	Matches(string) bool // Check if the given string matches the desired content
	String() string      // String representation of Content.
}

type Attribute struct {
	Name  string
	Value Content
}

// The simplest type of content: a plain string
type StringContent struct {
	Value string
}

func (sc StringContent) Matches(s string) bool {
	return s == sc.Value
}
func (sc StringContent) String() string {
	return sc.Value
}

// Shell like globbing content: Use * and ? (special meaning can be masked with \)
type PatternContent struct {
	Pattern string
}

func (pc PatternContent) Matches(s string) bool {
	m, _ := Match(pc.Pattern, s)
	return m
}
func (pc PatternContent) String() string {
	return pc.Pattern
}

// Regular expression content.
type RegexpContent struct {
	Regexp *regexp.Regexp
}

func (rc RegexpContent) Matches(s string) bool {
	return rc.Regexp.FindStringIndex(s) != nil
}
func (rc RegexpContent) String() string {
	return "/" + rc.Regexp.String() + "/"
}

// Factory to generate a Content object from its string representation. The following are distingusihed:
//   - strings with * or ? characters (e.g. "some*vlue") --> Patterm
//   - starts and exnds with / (e.g. "/the (cat|dog) .*/") --> Regexp
//   - all other  --> Fixed String
func MakeContent(cntnt string) (Content, os.Error) {
	if len(cntnt) > 2 && cntnt[0] == '/' && cntnt[len(cntnt)-1] == '/' {
		if rexp, err := regexp.Compile(cntnt[1 : len(cntnt)-1]); err == nil {
			return RegexpContent{rexp}, nil
		} else {
			error("Malformed regular expression: %s", err.String())
			return nil, err
		}
	} else if strings.Index(cntnt, "*") != -1 || strings.Index(cntnt, "?") != -1 {
		if _, err := Match(cntnt, cntnt); err != nil {
			error("Malformed regular expression: %s", err.String())
			return nil, err
		}
		return PatternContent{cntnt}, nil
	}

	return StringContent{cntnt}, nil
}

// TagSpec describes a specific tag
type TagSpec struct {
	// The tag name (lowercase)
	Name string
	// The text content of this tag
	Content Content
	// Deep content match
	Deep bool
	// Needed classes
	Classes []string
	// Forbidden classes
	XClasses []string
	// Needed attributes (with optional values)
	Attr map[string]Content
	// Forbidden Attributes
	XAttr map[string]Content
	// Sub specs
	Sub []*TagSpec
}


// Make a deep copy of ts which does not share data with ts.
func (ts *TagSpec) DeepCopy() *TagSpec {
	cp := new(TagSpec)
	cp.Name, cp.Content, cp.Deep = ts.Name, ts.Content, ts.Deep
	cp.Classes, cp.XClasses = ts.Classes, ts.XClasses
	cp.Attr, cp.XAttr = ts.Attr, ts.XAttr
	cp.Sub = make([]*TagSpec, len(ts.Sub))
	for i, s := range ts.Sub {
		cp.Sub[i] = s.DeepCopy()
	}

	return cp
}


// Yield a string representation of a TagSpec
func (ts *TagSpec) String() string {
	return ts2str(ts, 0)
}

// Real work part of TagSpec.String.
func ts2str(ts *TagSpec, indent int) string {
	ind := strings.Repeat("  ", indent)
	s := ind + ts.Name

	// attributes
	for name, cntnt := range ts.Attr {
		s += " " + name
		if cntnt != nil {
			s += "=" + cntnt.String()
		}
	}
	for name, cntnt := range ts.XAttr {
		s += " !" + name
		if cntnt != nil {
			s += "=" + cntnt.String()
		}
	}

	// classes
	for _, c := range ts.Classes {
		s += " class=" + c
	}
	for _, c := range ts.XClasses {
		s += " !class=" + c
	}

	// content
	if ts.Content != nil {
		if ts.Deep {
			s += " =D= "
		} else {
			s += " == "
		}
		s += ts.Content.String()
	}

	// sub tags
	for _, c := range ts.Sub {
		s += "\n" + ts2str(c, indent+1)
	}
	return s
}


func trim(s string) string {
	return strings.Trim(s, " \t")
}


// Decompose texttual tag specification spec into a TagSpec.
// Returns nil on error.  TODO: proper error handling (e.g. making sure regepx and patterns are okay).
func ParseSimpleTagSpec(spec string) (ts *TagSpec, err os.Error) {
	// fmt.Printf("Parsing: " + spec)
	ts = new(TagSpec)
	ts.Attr = make(map[string]Content)
	ts.XAttr = make(map[string]Content)
	spec = trim(spec)

	var cntnt string
	// TODO: swicth from "==" to "===" and split on " ==== " and " =D= " to safeguard 
	if strings.Index(spec, " == ") != -1 {
		ts.Deep = false
		p := strings.Split(spec, " == ", 2)
		spec, cntnt = trim(p[0]), trim(p[1])
		ts.Content, err = MakeContent(cntnt)
	} else if strings.Index(spec, " =D= ") != -1 {
		ts.Deep = true
		p := strings.Split(spec, " =D= ", 2)
		spec, cntnt = trim(p[0]), trim(p[1])
		ts.Content, err = MakeContent(cntnt)
	} else {
		ts.Content = nil
	}
	if err != nil {
		return
	}

	f := strings.Fields(spec)
	if len(f) == 0 {
		return nil, os.ErrorString("No tag given in tagspec.")
	}
	ts.Name = f[0]

	for i := 1; i < len(f); i++ {
		atr := f[i]
		var expected bool = true
		var val string
		var cntnt Content
		if atr[0] == '!' {
			atr = atr[1:]
			expected = false
		}
		if strings.Index(atr, "=") != -1 {
			p := strings.Split(atr, "=", 2)
			atr = trim(p[0])
			val = trim(p[1])
			cntnt, err = MakeContent(val) // err later (dont err for classes)
		}
		if atr == "class" {
			if expected {
				ts.Classes = append(ts.Classes, val)
			} else {
				ts.XClasses = append(ts.XClasses, val)
			}
		} else {
			if err != nil {
				return nil, err
			}
			// TODO: fail if overwriting existing attribute
			if expected {
				ts.Attr[atr] = cntnt
			} else {
				ts.XAttr[atr] = cntnt
			}
		}

	}
	return
}

// Returns the number of leading spaces in s.
func indentDepth(s string) (d int) {
	d = 0
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			d++
		} else if s[i] == '\t' {
			d += 4
		} else {
			break
		}
	}
	// trace("Indent depth of '%#v' == %d.", s, d)
	return
}

func ParseTagSpec(spec string) (ts *TagSpec, err os.Error) {
	trace("Parsing TagSpec: %s", spec)
	lines := strings.Split(spec, "\n", -1)
	ts, err = ParseSimpleTagSpec(lines[0])
	if err != nil {
		return
	}
	if len(lines) > 1 {
		ind := indentDepth(lines[1])
		// fmt.Printf("Have subs: indent >= %d\n", ind)
		var sub *TagSpec
		for i := 1; i < len(lines); {
			ss := lines[i]
			i++
			for ; i < len(lines) && indentDepth(lines[i]) > ind; i++ {
				ss += "\n" + lines[i]
			}
			sub, err = ParseTagSpec(ss)
			if err != nil {
				return
			}
			ts.Sub = append(ts.Sub, sub)
		}
	}
	return
}
