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

const (
	TEXT_NODE = "-TXT-"
)

// Node represents (kinda) node in the DOM of the parsed html.
// Main differences are Node.Text which contains the (unescaped) text 
// content of the node whereas Child does not contain TextNodes.
type Node struct {
	Parent *Node
	Name   string
	Attr   []html.Attribute
	Text   string
	Full   string
	Child  []*Node
	subs   []*Node   // subs contains real child tags _and_ text childs (where Name=="-TXT-")
	class  []string  
}

// String "representation" of a Node. Usefull for debuging purpose.
func (n *Node) String() (s string) {
	s = n.HtmlRep(-1)
	if len(n.Child) > 0 {
		s += fmt.Sprintf(" [%d children]", len(n.Child))
	}
	return
}

// Kind of "HTML" of a node. Kind of as no closing tags are printed
// and special charaters are not escaped.  If indent < 0 than no
// child nodes will be output. Otherwise indent is the indentation
// used as a prefix to every line generated.
func (n *Node) HtmlRep(indent int) (s string) {
	var in string
	if indent > 0 {
		in = strings.Repeat("  ", indent)
	}
	s = in + "<" + n.Name + ""
	for _, a := range n.Attr {
		s += " " + a.Key + "=\"" + a.Val + "\""
	}
	if len(n.class) > 0 {
		s += " class=\""
		for i, c := range n.class {
			if i > 0 { s += " " }
			s += c
		}
		s += "\""
	}
	s += "> " + n.Text
	if indent >= 0 {
		for _, c := range n.Child {
			s += "\n" + c.HtmlRep(indent + 1)
		}
	}
	return
}

// Should produce original html as seen by parser.
// Two differences: a) Selfclosing tags are selfclosing in the output
// and b) attribute values and text is escaped.
func (n *Node) Html() (s string) {
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
		if c.Name == TEXT_NODE {
			s += html.EscapeString(c.Text)
		} else {
			s += c.Html()
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


// Parse the given html and return the root node of the document.
// Parsing starts at the first StartToken and will ignore
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
			debug("Constructed Html: \n" + root.HtmlRep(0))
			return root
		}
	}
	return nil
}

// Normalize whitespace in t: 
//  - replace tabs, newline, ... with spaces, 
//  - collaps multiple spaces
//  - trim result.
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
			node.subs = append(node.subs, &Node{Parent: node, Name: TEXT_NODE, Text: t.Data})
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
	
	// debug("Made Node: " + node.String() + "\n")
	// fmt.Printf("Made node: %s\n", node.String())
	return
}

// Extract classes to own field in node and remove from Attr.
func prepareClasses(node *Node) {
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
