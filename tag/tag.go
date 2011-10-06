//
// Tag - Go package to facialiate testing tags in html documents.  Used by Webtest.
//
// Copyright 2011 Volker Dobler. All rights reserved.
//
package tag

/*
	This package helps testing for occurenc of tags in html/xml documents.

	Tags are described by a plaintext strings. The following examples show
	most of the possibilities

	  tagspec:
		tagname [ attr... ] [ contentOp content ]

	  attr:
	    [ class | attribute ]

	  class:
	  	[ '!' ] 'class' [ '=' fixed ]

	  attribute:
	  	[ '!' ] name [ '=' content ]

	  contentOp:
		[ '==' | '=D=' ]              '==' is normal matching of text content
		                               'wheras '=D=' is deep matching of nested
									   text content.

	  content:
		[ pattern | '/' regexp '/' ]   pattern may contain '*' and '?' and works
		                               like shell globing. regexp is what it is.

	Only specified classes, attributes and content is considered when finding
	tags in a html/xml document. E.g.:
	  "p lang=en"
	will match any p-tag with lang="en" regardless of any other classes, 
	attributes and content of the p-tag.

	Values for attributes may be ommitted: Such test just check wether the
	tag has the attribute (value does not matter).

	The difference between class and "normal" attribute testing is: Attributes
	may be specified only once and their value is optional wheras classes can
	be specified multiple times and must contain a value. Think of a tag like
	  <p class="important news wide">Some Text</p>
	As beeing something like
	  <p class="important" class="news" class="wide">Some Text</p>
	For finding tags.

	The texttual content of a tag is normalized: Any whitespace is collapsed
	(even &nbsp; and that like) and trimmed. E.g. the text content of
	  <p> This is &nbsp;	some text.   </p>
	is simply "This is some text.".
	If nested tags are present, their content is part of the normalized
	text content too. E.g. the deep text content of the p
	  <p>Some<big>important<span>and</span>useless</big>info</p>
	is "Some big important and useless info". Note that spaces are added
	around inner tags while the direct text content is just "Some info"

	To mach tag content use either "==" or "=D=".
	"==" matches direct content while "=D=" matches deep content.

	Some example

	  h3				match any h3 tag
	  h3 class			match if any class is present
	  h3 lang			match if any lang attribute is present
	  h3 lang=de		match if lang attribute is "de"
	  h3 lang=d?        would macth de, dx, d0, ...
	  h3 lang=d*e       would match de, de, dXe, d123e...
	  h3 class=a        match if tag has class a, e.g. class="x a b"
	  h3 !class			match if no class attribute is present
	  h3 !id			match if no id attribute is set
	  h3 title=			match if title is presnet but empty: title=""
	  h3 !lang=de		match if no lang is present or value is not de
	  h3 == Hello		match <h3>Hello</h3> as well as <h3>   Hello </h3>
	  h3 == H?llo*      would match stuff like "Hallo du da..."
	  h3 == /(cat|dog)/	regexp either cat or dog

*/

import (
	"fmt"
	"html"
	"log"
	"os"
)

var LogLevel int = 2 // 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace
var logger *log.Logger

func init() {
	logger = log.New(os.Stderr, "Tag     ", log.Ldate|log.Ltime)
}

func error(f string, m ...interface{}) {
	if LogLevel >= 1 {
		logger.Print("*ERROR* " + fmt.Sprintf(f, m...))
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
func supertrace(f string, m ...interface{}) {
	if LogLevel >= 6 {
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
func containsAttr(attr []html.Attribute, name string, cntnt Content) bool {
	for _, at := range attr {
		if at.Key == name {
			if cntnt == nil || cntnt.Matches(at.Val) {
				return true
			} else {
				return false
			}
		}
	}
	return false
}

// Check if ts matches the tag node
func Matches(ts *TagSpec, node *Node) bool {
	supertrace("Trying node: " + node.String())

	// Tag Name
	if node.Name != ts.Name {
		return false
	}

	if LogLevel != 5 {
		debug("Trying node: " + node.String())
	}
	// Tag Attributes
	for name, cntnt := range ts.Attr {
		debug("  Checking needed attribute '%s' = %v", name, cntnt)
		if !containsAttr(node.Attr, name, cntnt) {
			debug("    --> missing")
			return false
		}
	}
	for name, cntnt := range ts.XAttr {
		debug("  Checking forbidden attribute %s", name)
		if containsAttr(node.Attr, name, cntnt) {
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
	if ts.Content != nil {
		var nc string
		if ts.Deep {
			nc = node.Full
		} else {
			nc = node.Text
		}

		debug("  Checking for content %#v", nc)
		if !ts.Content.Matches(nc) {
			debug("    --> mismatch")
			return false
		}
	}

	// Sub Tags
	ci := 0 // next child to test
	numSubs, numChilds := len(ts.Sub), len(node.Child)
	for si := 0; si < numSubs; si++ {
		debug("  Checking subnode %d", si)
		found := false
		remaining := numSubs - si - 1
		for ; ci < numChilds-remaining; ci++ { // only test childs which are not needed for remaining subspecs
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

// Find the first tag under node which matches the given TagSpec ts.
// If node matches, node will be returned. If no match is found nil is returned.
func FindTag(ts *TagSpec, root *Node) *Node {
	debug("FindTag: " + ts.String())
	node := findTag(ts, root)

	return node
}

// The real work part of FindTag.
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

// Find all non-nested tags under node which matches the given TagSpec ts.
func FindAllTags(ts *TagSpec, node *Node) []*Node {
	debug("FindAllTags: " + ts.String())
	list := make([]*Node, 0, 10)
	findAllTags(ts, node, &list)
	return list
}

// The real work part of FindAllTags.
func findAllTags(ts *TagSpec, node *Node, lp *[]*Node) {
	if Matches(ts, node) {
		*lp = append(*lp, node)
		return
	}
	for _, child := range node.Child {
		findAllTags(ts, child, lp)
	}
}

// Find the first tag under node which matches the given TagSpec ts.
// If node matches, node will be returned. If no match is found nil is returned.
func CountTag(ts *TagSpec, node *Node) (n int) {
	debug("CountTag: " + ts.String())
	return countTag(ts, node)
}

// The real work part of CountTag.
func countTag(ts *TagSpec, node *Node) (n int) {
	if Matches(ts, node) {
		return 1
	}
	for _, child := range node.Child {
		n += countTag(ts, child)
	}
	return
}
