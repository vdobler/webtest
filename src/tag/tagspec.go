package tag

import (
	"fmt"
	// "bufio"
	// "os"
	"html"
	// "log"
	"strings"
	// "container/vector"
)

// The "value" if the value of an attribute is of no concern 
const IGNORED = "._IGN_."


// TagSpec describes a specific tag
type TagSpec struct {
	// The tag name (lowercase)
	Name string
	// The text content of this tag
	Content string
	// Deep content match
	Deep bool
	// Needed classes
	Classes []string
	// Forbidden classes
	XClasses []string
	// Needed attributes (with optional values)
	Attr []html.Attribute
	// Forbidden Attributes
	XAttr []html.Attribute
	// Sub specs
	Sub []*TagSpec
}


// Yield a string representation of a TagSpec
func (ts *TagSpec) String() string {
	return ts2str(ts, 0)
}


func ts2str(ts *TagSpec, indent int) string {
	ind := strings.Repeat("  ", indent)
	s := ind + ts.Name
	for _, a := range ts.Attr {
		k, v := a.Key, a.Val
		s += " " + k
		if v != IGNORED {
			s += "=" + v
		}
	}
	for _, a := range ts.XAttr {
		k, v := a.Key, a.Val
		s += " !" + k
		if v != IGNORED {
			s += "=" + v
		}
	}

	for _, c := range ts.Classes {
		s += " class=" + c
	}
	for _, c := range ts.XClasses {
		s += " !class=" + c
	}

	if ts.Content != "" {
		if ts.Deep {
			s += " =D= "
		} else {
			s += " == "
		}
		s += ts.Content
	}

	for _, c := range ts.Sub {
		s += "\n" + ts2str(c, indent+1)
	}
	return s
}


func trim(s string) string {
	return strings.Trim(s, " \t")
}


func ParseSimpleTagSpec(spec string) (ts *TagSpec) {
	fmt.Printf("Parsing: " + spec)
	ts = new(TagSpec)
	spec = trim(spec)

	if strings.Index(spec, "==") != -1 {
		ts.Deep = false
		p := strings.Split(spec, "==", 2)
		spec, ts.Content = trim(p[0]), trim(p[1])
	} else if strings.Index(spec, "=D=") != -1 {
		ts.Deep = true
		p := strings.Split(spec, "=D=", 2)
		spec, ts.Content = trim(p[0]), trim(p[1])
	}

	f := strings.Fields(spec)
	if len(f) == 0 {
		fmt.Printf("Wrong tagspec! will use XXX-tag\n")
		ts.Name = "XXX"
		return
	}
	ts.Name = f[0]

	for i := 1; i < len(f); i++ {
		atr := f[i]
		var expected bool = true
		var val string = IGNORED
		if atr[0] == '!' {
			atr = atr[1:]
			expected = false
		}
		if strings.Index(atr, "=") != -1 {
			p := strings.Split(atr, "=", 2)
			atr = p[0]
			val = p[1]
		}
		if atr == "class" {
			if expected {
				ts.Classes = append(ts.Classes, val)
			} else {
				ts.XClasses = append(ts.XClasses, val)
			}
		} else {
			if expected {
				ts.Attr = append(ts.Attr, html.Attribute{Key: atr, Val: val})
			} else {
				ts.XAttr = append(ts.XAttr, html.Attribute{Key: atr, Val: val})
			}
		}

	}
	return
}


func ParseTagSpec(spec string) (ts *TagSpec) {
	// TODO: Implement correct behaviour
	lines := strings.Split(spec, "\n", -1)
	ts = ParseSimpleTagSpec(lines[0])
	for i := 1; i < len(lines); i++ {
		ts.Sub = append(ts.Sub, ParseSimpleTagSpec(lines[i]))
	}
	return
}
