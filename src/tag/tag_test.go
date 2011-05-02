package main

import (
	"./tag"
	"testing"
	"regexp"
	"fmt"
)

func check(doc *tag.Node, spec, expectedId string, t *testing.T) {
	fmt.Printf("Spec: %s\n", spec)
	ts := tag.ParseTagSpec(spec)
	if ts == nil {
		t.Error("Unparsabel " + spec)
		return
	} else {
		n := tag.FindTag(ts, doc)
		if n == nil {
			t.Error("Not found: " + spec)
			return
		} else {
			for _, a := range n.Attr {
				if a.Key == "id" {
					if a.Val == expectedId {
						return
					} else {
						t.Error("In " + spec + ": Expected id " + expectedId + " but got " + a.Val)
						return
					}
				}
			}
			t.Error("In " + spec + ": No id")
		}
	}
}

var SimpleHtml = `<!DOCTYPE html>
<html>
<body>
	<div id="div1">
		<p id="first">Hello World!</p>
		<p id="emptyp"></p>
		<p id="alphabet">abcdefghijklmnopqrtstuvwxyz</p>
		<p id="important" class="important wichtig high" title="FirstTitle" >Important</p>
		<p id="A" class="important high" ></p>
		<p id="B" class="aa bb" ></p>
		<p id="C" class="bb aa" ></p>
		<p id="D" class="xx yy" ></p>
		<h1 id="h1_1" lang="de">Berlin</h1>
		<h1 id="h1_2" lang="nl" title="AT">Amsterdam</h1>
		<h1 id="h1_2" lang="de" title="DT">Dortmund</h1>
		<h1 id="h1_3" class="head">Chicago</h1>
		<span id="SP1">Some aaaa text</span
	</div1>
	
</body>
</html>`


func TestBasics(t *testing.T) {
	doc := tag.ParseHtml(SimpleHtml)
	if doc == nil {
		t.Error("Unparsabel html.")
		t.FailNow()
	}

	check(doc, "div", "div1", t)
	check(doc, "p == Hello World!", "first", t)
	check(doc, "p == *xyz", "alphabet", t)
	check(doc, "p == abc*", "alphabet", t)
	check(doc, "p == abcd*wxyz", "alphabet", t)
	check(doc, "p class=important", "important", t)
	check(doc, "p class=high", "important", t)
	check(doc, "p class=important class=high", "important", t)
	check(doc, "p !title=Wrong == Important", "important", t)
	check(doc, "p !title=Title* == Important", "important", t)
	check(doc, "p !title=*First == Important", "important", t)
	check(doc, "p title=First*Title", "important", t)
	check(doc, "p title=*Title", "important", t)
	check(doc, "p title=First*", "important", t)
	check(doc, "p class=important !class=wichtig class=high", "A", t)
	check(doc, "p class=aa class=bb", "B", t)
	check(doc, "p class=bb class=aa", "B", t)
	check(doc, "p class=xx class=yy", "D", t)
	check(doc, "p class=yy class=xx", "D", t)

	check(doc, "h1 == Berlin", "h1_1", t)
	check(doc, "h1 !title", "h1_1", t)
	check(doc, "h1 !lang", "h1_3", t)
	check(doc, "h1 lang", "h1_1", t)
	check(doc, "h1 title", "h1_2", t)
	check(doc, "h1 !lang=nl", "h1_1", t)
	check(doc, "h1 !lang=de", "h1_2", t)
	check(doc, "h1 lang title", "h1_2", t)
	check(doc, "h1 lang !title=AT", "h1_1", t)

	check(doc, "span == /Some.*text/", "SP1", t)
	check(doc, "span == /Some [aeio]+ text/", "SP1", t)
	// check(doc, "", "", t)
}

func main() {
	tests := []testing.InternalTest{
		{"TestBasics", TestBasics},
	}
	testing.Main(regexp.MatchString, tests)
}
