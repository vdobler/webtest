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

var Debug bool

func debug(m ...interface{}) {
	if Debug {
		str := fmt.Sprint(m...)
		fmt.Print(str + "\n")
	}
	// log.Print(m...)
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

	// Tag Name
	if node.Name == "*" {
		return true
	}
	if node.Name != ts.Name {
		return false
	}

	// Tag Attributes
	for _, a := range ts.Attr {
		debug("  Checking needed attribute ", a)
		if !containsAttr(a, node.Attr) {
			debug("    --> missing")
			return false
		}
	}
	for _, a := range ts.XAttr {
		debug("  Checking forbidden attribute ", a)
		if containsAttr(a, node.Attr) {
			debug("    --> present")
			return false
		}
	}

	// Classes
	for _, c := range ts.Classes {
		debug("  Checking needed class " + c)
		if !containsClass(c, node.class) {
			debug("    --> missing")
			return false
		}
	}
	for _, c := range ts.XClasses {
		debug("  Checking forbidden class " + c)
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

		if !textMatches(nc, ts.Content) {
			return false
		}
	}

	// Sub Tags
	ci := 0 // next child to test
	for si := 0; si < len(ts.Sub); si++ {
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

	return true
}

func regexpMatches(s, exp string) bool {
	return false  // TODO: implement
}

func wildcardMatches(s, exp string) bool {
	parts := strings.Split(exp, "*", 2)
	if strings.HasPrefix(s, parts[0]) && strings.HasSuffix(s, parts[1]) {
		return true
	}
	return false
}

// Dispatch "plain text", "/regular expression/" and "wildcard * search"
// to the appropriate functions
func textMatches(s, exp string) bool {
	if exp == "" {
		return true
	}

	if exp[0] == '/' && exp[len(exp)-1] == '/' {
		return regexpMatches(s, exp[1:len(exp)-2])
	} else if strings.Index(exp, "*") >= 0 {
		return wildcardMatches(s, exp)
	}

	return exp == s
}

// Find the first tag under node which matches the given TagSpec ts.
// If node matches, node will be returned. If no match is found nil is returned.
func FindTag(ts *TagSpec, node *Node) *Node {
	// debug("FindTag: " + node.String())
	if Matches(ts, node) {
		// debug("Found!")
		return node
	}
	for _, child := range node.Child {
		if m := FindTag(ts, child); m != nil {
			return m
		}
	}
	// debug("Not found")
	return nil
}
