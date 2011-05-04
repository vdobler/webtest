package tag

import (
	"fmt"
	"regexp"
	"os"
	"html"
	"log"
	"strings"
)

var LogLevel int = 3 // 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace
var logger *log.Logger

func init() {
	logger = log.New(os.Stderr, "Tag ", log.Ldate | log.Ltime) 
}

func error(f string, m ...interface{}) {
	if LogLevel >= 1 {
		logger.Panic("*ERROR* " + fmt.Sprintf(f, m...))
	}
}
func warn(f string, m ...interface{}) {
	if LogLevel >= 2 {
		logger.Print("*WARN * " + fmt.Sprintf(f, m...))
	}
}
func info(f string, m ...interface{}) {
	if LogLevel >= 3 {
		logger.Print("*INFO * " + fmt.Sprintf(f, m...))
	}
}
func debug(f string, m ...interface{}) {
	if LogLevel >= 4 {
		logger.Print("*DEBUG* " + fmt.Sprintf(f, m...))
	}
}
func trace(f string, m ...interface{}) {
	if LogLevel >= 5 {
		logger.Print("*TRACE* " + fmt.Sprintf(f, m...))
	}
}

// Check if cl is presnet in classes.
func containsClass(cl string, classes []string) bool {
	for _, c := range classes {
		if c == cl {
			return true
		}
	}
	return false
}

// Check if a is contained in attr.
func containsAttr(a html.Attribute, attr []html.Attribute) bool {
	k, v := a.Key, a.Val
	for _, at := range attr {
		if k == at.Key {
			if v == IGNORED || textMatches(at.Val, v) {
				return true
			}
		}
	}
	return false
}


// Check if ts matches the token node
func Matches(ts *TagSpec, node *Node) bool {
	debug("Trying node: " + node.String())
	
	// Tag Name
	if ts.Name == "*" {
		return true
	}
	if node.Name != ts.Name {
		return false
	}

	// Tag Attributes
	for _, a := range ts.Attr {
		debug("  Checking needed attribute %s", a)
		if !containsAttr(a, node.Attr) {
			debug("    --> missing")
			return false
		}
	}
	for _, a := range ts.XAttr {
		debug("  Checking forbidden attribute %s", a)
		if containsAttr(a, node.Attr) {
			debug("    --> present")
			return false
		}
	}

	// Classes
	for _, c := range ts.Classes {
		debug("  Checking needed class %s", c)
		if !containsClass(c, node.class) {
			debug("    --> missing")
			return false
		}
	}
	for _, c := range ts.XClasses {
		debug("  Checking forbidden class %s", c)
		if containsClass(c, node.class) {
			debug("    --> present")
			return false
		}
	}

	// Content
	if ts.Content != "" {
		nc := node.Text
		if ts.Deep {
			nc = node.Full
		}
		debug("  Checking for content " + nc)
		if !textMatches(nc, ts.Content) {
			debug("    --> mismatch")
			return false
		}
	}

	// Sub Tags
	ci := 0 // next child to test
	for si := 0; si < len(ts.Sub); si++ {
		debug("  Checking subnode %d", si)
		var found bool = false
		for ; ci < len(node.Child); ci++ {
			if found = Matches(ts.Sub[si], node.Child[ci]); found {
				break
			}
		}
		if !found {
			return false
		}
	}

	debug("==> found")
	return true
}

// TODO: Compile just once (during parsing/tagspec construction
func regexpMatches(s, exp string) bool {
	// fmt.Printf("Regexp Match '%s' :: '%s'\n", s, exp)
	if rexp, err := regexp.Compile(exp); err == nil {
		return (rexp.FindStringIndex(s) != nil)
	} else {
		fmt.Printf("Invalid regexp '%s': %s\n", exp, err.String())
	}
	trace("    --> regexp mismatch")
	return false
}

func wildcardMatches(s, exp string) bool {
	parts := strings.Split(exp, "*", 2)
	if strings.HasPrefix(s, parts[0]) && strings.HasSuffix(s, parts[1]) {
		return true
	}
	trace("    --> wildcard mismatch")
	return false
}

// Dispatch "plain text", "/regular expression/" and "wildcard * search"
// to the appropriate functions
func textMatches(s, exp string) bool {
	trace("textMatches: got '%s' expected '%s'", s, exp)
	if exp == "" {
		return true
	}

	if exp[0] == '/' && exp[len(exp)-1] == '/' {
		return regexpMatches(s, exp[1:len(exp)-1])
	} else if strings.Index(exp, "*") >= 0 {
		return wildcardMatches(s, exp)
	}
	
	if exp != s {
		trace("    --> compare mismatch")
		return false
	}
	return true
}

// Find the first tag under node which matches the given TagSpec ts.
// If node matches, node will be returned. If no match is found nil is returned.
func FindTag(ts *TagSpec, node *Node) *Node {
	debug("FindTag: " + ts.String())
	return findTag(ts, node)
}

func findTag(ts *TagSpec, node *Node) *Node {
	if Matches(ts, node) {
		return node
	}
	for _, child := range node.Child {
		if m := findTag(ts, child); m != nil {
			return m
		}
	}
	return nil
}
