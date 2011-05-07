package tag

import (
	"fmt"
	"bytes"
	"os"
	"html"
	"xml"
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
	subs   []*Node // subs contains real child tags _and_ text childs (where Name=="-TXT-")
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
			if i > 0 {
				s += " "
			}
			s += c
		}
		s += "\""
	}
	s += "> " + n.Text
	/* *********************************
	if n.Full != "" {
		s += "  [[" + n.Full + "]]"
	}
	********************************** */
	if indent >= 0 {
		for _, c := range n.Child {
			s += "\n" + c.HtmlRep(indent+1)
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
			if i > 0 {
				s += " "
			}
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

// Extract classes to own Class field in node and remove from Attr.
func prepareClasses(node *Node) {
	for i, a := range node.Attr {
		if a.Key == "class" {
			node.class = strings.Fields(a.Val)
			// Remove from Attr
			m := len(node.Attr) - 1
			for j := i; j < m; j++ {
				node.Attr[j] = node.Attr[j+1]
			}
			node.Attr = node.Attr[:m]
			break
		}
	}
}

// Normalize whitespace in t: 
//  - replace each tab, newline and cariage return with a single space, 
//  - collaps multiple spaces to one
//  - trim result.
func cleanText(t string) (s string) {
	s = strings.Replace(strings.Replace(strings.Replace(t, "\n", " ", -1), "\r", " ", -1), "\t", " ", -1)
	for strings.Contains(s, "        ") {
		s = strings.Replace(s, "        ", " ", -1)
	}
	for strings.Contains(s, "    ") {
		s = strings.Replace(s, "    ", " ", -1)
	}
	for strings.Contains(s, "  ") {
		s = strings.Replace(s, "  ", " ", -1)
	}
	s = strings.Trim(s, " ")
	return
}


// Parse the given html and return the root node of the document.
// Parsing starts at the first StartToken and will ignore
func ParseHtml(h string) (root *Node, err os.Error) {
	debug(h)
	r := strings.NewReader(h)
	parser := xml.NewParser(r)
	parser.Strict = false
	parser.AutoClose = xml.HTMLAutoClose
	parser.Entity = xml.HTMLEntity
	for {
		var tok xml.Token
		tok, err = parser.Token()
		if err != nil {
			fmt.Printf("XXXXXXXXXXX\n")
			return
		}
		switch tok.(type) {
		case xml.StartElement:
			debug("Starting parsing from %v", tok)
			root, err = parse(tok, parser, nil)
			trace("=========== Parser ==========\nConstructed Structure: \n" + root.HtmlRep(0))
			trace("\n----------------------------\nRe-Constructed Html: \n" + root.Html() + "\n===============================")
			return
		}
	}
	return
}


func parse(tok xml.Token, parser *xml.Parser, parent *Node) (node *Node, err os.Error) {
	node = new(Node)
	node.Parent = parent
	st, _ := tok.(xml.StartElement)
	node.Name = st.Name.Local
	trace("parsing tag %s", node.Name)
	node.Attr = []html.Attribute{}
	for _, attr := range st.Attr {
		a := html.Attribute{Key: attr.Name.Local, Val: attr.Value}
		node.Attr = append(node.Attr, a)
	}

	// var childs vector.Vector
	// var chld []*Node

	for done := false; !done; {
		var tok xml.Token
		tok, err = parser.Token()
		if err != nil {
			if err == os.EOF {
				err = nil
				break
			}
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			ch, err := parse(t, parser, node)
			if err != nil {
				return
			}
			node.Child = append(node.Child, ch)
			node.subs = append(node.subs, ch)
			if node.Full != "" {
				node.Full += " "
			}
			node.Full += ch.Full
		case xml.EndElement:
			if t.Name.Local != node.Name {
				fmt.Printf("Tag " + node.Name + " closed by " + t.Name.Local + "\n")
			}
			done = true
		case xml.CharData:
			b := bytes.NewBuffer([]byte(t))
			s := b.String()
			ct := " " + cleanText(s)
			node.Text += ct
			node.Full += ct
			node.subs = append(node.subs, &Node{Parent: node, Name: TEXT_NODE, Text: s})
		case xml.Comment, xml.Directive, xml.ProcInst:
			// skip
		default:
			fmt.Printf("Very strange:\nType = %t\n Value = %#v\n", tok, tok)
		}
	}

	node.Text = strings.Trim(node.Text, " \n\t\r")
	node.Full = cleanText(node.Full)

	prepareClasses(node)

	trace("Made Node: " + node.String() + "\n")
	return
}
