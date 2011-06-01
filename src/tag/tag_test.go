package tag

import (
	"testing"
	_ "fmt"
	// "strings"
)

func MustParse(spec string, t *testing.T) *TagSpec {
	ts, err := ParseTagSpec(spec)
	if ts == nil || err != nil {
		t.Errorf("Unexpected unparsable tagspec '%s': %s", spec, err.String())
		t.FailNow()
		return nil
	}
	return ts
}

func check(doc *Node, spec, expectedId string, t *testing.T) {
	// fmt.Printf("Spec: %s\n", spec)
	ts := MustParse(spec, t)
	n := FindTag(ts, doc)
	if n == nil {
		t.Error("Not found: " + spec)
		return
	}

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


func checkN(doc *Node, spec string, t *testing.T) {
	// fmt.Printf("Spec: %s\n", spec)
	ts := MustParse(spec, t)
	if ts == nil {
		t.Error("Unparsabel " + spec)
		return
	}

	n := FindTag(ts, doc)
	if n != nil {
		t.Error("Found: " + spec)
	}
}


var testSimpleHtml = `<!DOCTYPE html>
<html>
<body>
	<div id="div1">
		<p id="first">Hello World!</p>
		<p id="emptyp"></p>
		<p id="alphabet">abcdefghijklmnopqrtstuvwxyz</p>
		<p id="important" class="important wichtig high" title="FirstTitle" >Important</p>
		<p id="A" class="important high" ></p>
		<p id="B" class="aa bb" >huhu</p>
		<p id="C" class="bb aa" ></p>
		<p id="D" class="xx yy" ></p>
		<h1 id="h1_1" lang="de">Berlin</h1>
		<h1 id="h1_2" lang="nl" title="AT">Amsterdam</h1>
		<h1 id="h1_2" lang="de" title="DT">Dortmund</h1>
		<h1 id="h1_3" class="head">Chicago</h1>
		<span id="SP1">Some aaaa text</span>
	</div>
	<h4 id="theH4">UTF-8 Umlaute äöüÄÖÜ Euro €</h4>
	<div id="div2" class="news">
		<h2>The <span class="red">new</span> Title</h2>
		<p>New York Rio Tokio</p>
	</div>
	<div id="div3" lang="it">123
		<p id="nested"> AA <span> BB </span> CC <em> DD <code> EE </code> FF </em> GG </p>
		456
	</div>
	<div id="div4"><p id="plu">Luzern</p></div>
	<div id="div5"><p id="pch"><span id="sch">Chiasso</span></p></div>
	<div id="deep"><p><div><p><span><div><p><span>Deeeeeep</span></p></div></span></p></div></p></div>
	<p id="LongText" class="LongText">This is a pretty long text.</p>
	<a href="http://some.sub.domain.org/fancy/path/here" id="a123"> Link deep down </a>
	<h3 id="emptyh3"> 	 
		&nbsp; &ensp; &emsp; &thinsp; 
	</h3>
</body>
</html>`


func TestBasics(t *testing.T) {
	doc, err := ParseHtml(testSimpleHtml)
	if err != nil {
		t.Error("Unparsabel html: " + err.String())
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

	check(doc, "p class=bb", "B", t)
	check(doc, "p class=bb == ", "C", t)
	check(doc, "h3 ==", "emptyh3", t)

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

	check(doc, "a href=/.*domain.org/.*/", "a123", t)

	checkN(doc, "p == Hello World", t)
	checkN(doc, "p !title=FirstTitle == Important", t)

	// check(doc, "", "", t)
}


func TestTextcontent(t *testing.T) {
	doc, err := ParseHtml(testSimpleHtml)
	if err != nil {
		t.Error("Unparsabel html: " + err.String())
		t.FailNow()
	}

	check(doc, "p == Luzern", "plu", t)
	check(doc, "p == Lu?ern", "plu", t)
	check(doc, "p == L?z?rn", "plu", t)
	check(doc, "p == Luze*", "plu", t)
	check(doc, "p == *zern", "plu", t)
	check(doc, "p == /Luzern/", "plu", t)
	check(doc, "p == /L.zern/", "plu", t)
	check(doc, "p == /^L.zern$/", "plu", t)

	check(doc, "p =D= Luzern", "plu", t)
	check(doc, "span == Chiasso", "sch", t)
	check(doc, "span =D= Chiasso", "sch", t)
	checkN(doc, "p == Chiasso", t)
	check(doc, "p =D= Chiasso", "pch", t)

	checkN(doc, "p == AA BB CC DD EE FF GG", t)
	check(doc, "p =D= AA BB CC DD EE FF GG", "nested", t)
	checkN(doc, "div == 123 AA * GG 456", t)
	check(doc, "div =D= 123 AA * GG 456", "div3", t)

	check(doc, "p class=LongText == This is a pretty long text.", "LongText", t)
	check(doc, "p class=LongText == This is a * long text.", "LongText", t)
	check(doc, "p class=LongText == This * long text.", "LongText", t)
	check(doc, "p class=LongText == This * a pretty * text.", "LongText", t)
	check(doc, "p class=LongText == This ?? ? pretty*text.", "LongText", t)
	check(doc, "p id=Lon*ext == This is a pretty long text.", "LongText", t)
	check(doc, "p id=?ong?ext == This * text.", "LongText", t)

	check(doc, "h4 == UTF-8 *", "theH4", t)
	check(doc, "h4 == UTF-8 Umlaute äöüÄÖÜ Euro €", "theH4", t)
	check(doc, "h4 == UTF-8 * äöüÄÖÜ Euro €", "theH4", t)
	check(doc, "h4 == *€", "theH4", t)
}

func TestNestedTags(t *testing.T) {
	doc, err := ParseHtml(testSimpleHtml)
	if err != nil {
		t.Error("Unparsabel html: " + err.String())
		t.FailNow()
	}
	check(doc, "div\n  p id=A", "div1", t)
	check(doc, "div\n  span == Some*text", "div1", t)
	check(doc, "div\n p\n  div\n   p =D= Deeeeeep", "deep", t)
	check(doc, "div class=news\n  h2\n    span class=red == new", "div2", t)
}

func TestCounting(t *testing.T) {
	doc, err := ParseHtml(testSimpleHtml)
	if err != nil {
		t.Error("Unparsabel html: " + err.String())
		t.FailNow()
	}

	counts := map[string]int{"html": 1, "body": 1, "div": 6, "p": 14, "h1": 4, "h2": 1, "span": 5}
	for q, n := range counts {
		ts := MustParse(q, t)
		m := CountTag(ts, doc)
		if m != n {
			t.Errorf("Found %d instances of %s but expected %d.", m, q, n)
		}
	}

	tsx := MustParse("div id=div1", t)

	root := FindTag(tsx, doc)
	if root == nil {
		t.Error("No div id=div1 found")
		t.FailNow()
	}
	counts = map[string]int{"html": 0, "body": 0, "div": 1, "p": 8, "h1": 4, "h2": 0, "span": 1}
	for q, n := range counts {
		ts := MustParse(q, t)
		m := CountTag(ts, root)
		if m != n {
			t.Errorf("Found %d instances of %s but expected %d.", m, q, n)
		}
	}
}
