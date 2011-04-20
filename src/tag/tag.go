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


func debug(m ...interface{}) {
	str := fmt.Sprint(m...)
	fmt.Print(str + "\n")
	// log.Print(m...)
}



// Our own node used for parsing
type Node struct {
	Parent *Node
	Name   string
	Attr   []html.Attribute
	Text   string
	Full   string
	Child  []*Node
	subs   []*Node
	class  []string
}

// String representation of a Node
func (n *Node) String() (s string) {
	s = "<" + n.Name + "> "
	if len(n.Child) > 0 {
		s += fmt.Sprintf("(%d childs) ", len(n.Child))
	}
	s += n.Text
	return
}

// Kind of "HTML" of a node
func (n *Node) Html(indent int) (s string) {
	in := ""
	for i := 0; i < indent; i++ {
		in += "  "
	}
	s = in + "<" + n.Name + ""
	for _, a := range n.Attr {
		s += " " + a.Key + "=\"" + a.Val + "\""
	}
	s += "> " + n.Text + "\n"
	for _, c := range n.Child {
		s += c.Html(indent + 1)
	}
	s += in + "</" + n.Name + ">\n"
	return
}

// Should produce original html
func (n *Node) RealHtml() (s string) {
	s = "<" + n.Name + ""
	for _, a := range n.Attr {
		s += " " + a.Key + "=\"" + html.EscapeString(a.Val) + "\""
	}
	if len(n.class) > 0 {
		s += " class=\""
		for i, c := range n.class {
			if i > 0 { s += " " }
			s += c
		}
		s += "\""
	}
	s += ">"
	for _, c := range n.subs {
		if c.Name == "-TEXT-" {
			s += html.EscapeString(c.Text)
		} else {
			s += c.RealHtml()
		}
	}
	switch n.Name {
	case "img", "meta", "br", "input":
		s += " />"
	default:
		s += "</" + n.Name + ">"
	}
	return
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
	return false
}

func wildcardMatches(s, exp string) bool {
	parts := strings.Split(exp, "*", 2)
	if strings.HasPrefix(s, parts[0]) && strings.HasSuffix(s, parts[1]) {
		return true
	}
	return false
}

// Dispatch "plain text", "/regular expression/" and "wildcard * search"
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


func ParseHtml(h string) (root *Node) {
	r := strings.NewReader(h)
	z := html.NewTokenizer(r)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return nil
		}
		if tt == html.StartTagToken {
			tok := z.Token()
			root = parse(tok, z, nil)
			debug("Constructed Html: \n" + root.Html(0))
			return root
		}
	}
	return nil
}

// Normalize whitespace in t: Replace tabs, newline, ... with spaces, collaps multiple spaces and trim.
func cleanText(t string) (s string) {
	s = strings.Replace(strings.Replace(strings.Replace(t, "\n", " ", -1), "\r", " ", -1), "\t", " ", -1)
	for strings.Contains(s, "    ") {
		s = strings.Replace(s, "    ", " ", -1)
	}
	for strings.Contains(s, "  ") {
		s = strings.Replace(s, "  ", " ", -1)
	}
	s = strings.Trim(s, " ")
	return
}

func parse(tok html.Token, z *html.Tokenizer, parent *Node) (node *Node) {
	if tok.Type != html.StartTagToken {
		fmt.Printf("parse() called on non start tag\n")
		return nil
	}
	// debug("parse on tok=" + tok.Data)
	node = new(Node)
	node.Parent = parent
	node.Name = tok.Data
	node.Attr = tok.Attr
	// var childs vector.Vector
	// var chld []*Node

	for done := false; !done; {
		tt := z.Next()
		if tt == html.ErrorToken {
			return
		}
		t := z.Token()

		// Some Tags are selfclosing, even if not written properly as "<tag />"
		if tt == html.StartTagToken {
			switch t.Data {
			case "img", "meta", "input", "br", "hr":
				tt = html.SelfClosingTagToken
			}
		}

		switch tt {
		case html.StartTagToken:
			// debug("STT " + t.Data)
			ch := parse(t, z, node)
			node.Child = append(node.Child, ch)
			node.subs = append(node.subs, ch)
			node.Full += " " + ch.Full
		case html.EndTagToken:
			// debug("ETT " + t.Data)
			if t.Data != node.Name {
				fmt.Printf("Tag " + node.Name + " closed by " + t.Data + "\n")
			}
			done = true
		case html.TextToken:
			// debug("TT")
			ct := " " + cleanText(t.Data)
			node.Text += ct
			node.Full += ct
			node.subs = append(node.subs, &Node{Parent: node, Name: "-TEXT-", Text: t.Data})
		case html.SelfClosingTagToken:
			// debug("SCTT " + t.Data)
			ch := Node{Name: t.Data, Attr: t.Attr}
			node.Child = append(node.Child, &ch)
			node.subs = append(node.subs, &ch)
		}
	}

	node.Text = strings.Trim(node.Text, " \n\t\r")
	node.Full = strings.Trim(node.Full, " \n\t\r")
	
	prepareClasses(node)
	
	fmt.Printf("Made Node: " + node.String() + "\n")
	return
}

func prepareClasses(node *Node) {
	// Extract classes to own field in node
	for i, a := range node.Attr {
		if a.Key == "class" {
			node.class = strings.Fields(a.Val)
			// Remove from Attr
			m := len(node.Attr) - 1
			for j:=i; j<m; j++ {
				node.Attr[j] = node.Attr[j+1]
			}
			node.Attr = node.Attr[:m]
			break
		}
	}
}

func FindTag(ts *TagSpec, node *Node) *Node {
	// debug("FindTag: " + node.String())
	if Matches(ts, node) {
		debug("Found!")
		return node
	}
	for _, child := range node.Child {
		if m := FindTag(ts, child); m != nil {
			return m
		}
	}
	debug("Not found")
	return nil
}
