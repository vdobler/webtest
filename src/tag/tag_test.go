package tag

import (
	"testing"
	_ "fmt"
	"strings"
)

func check(doc *Node, spec, expectedId string, t *testing.T) {
	// fmt.Printf("Spec: %s\n", spec)
	ts := ParseTagSpec(spec)
	if ts == nil {
		t.Error("Unparsabel " + spec)
		return
	}
	
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
	ts := ParseTagSpec(spec)
	if ts == nil {
		t.Error("Unparsabel " + spec)
		return
	}
	
	n := FindTag(ts, doc)
	if n != nil {
		t.Error("Found: " + spec)
	}
}

var structureHtml = `<html>
<body>
	<h1> A </h1>
	<p> B
		<span> C </span>
		D
	</p>
	<h2> E </h2>
	<div>
		<p> F </p>
		<p> G </p>
	</div>
</body>
</html>
`

var brokenHtml1 = `<!DOCTYPE html>
<html>
<body>
	<div id="div1">
		<span id="SP1">Some aaaa text</span>
	</wrong>
	<p>Some Text</p>
</body>
</html>`

var brokenHtml2 = `<!DOCTYPE html>
<html>
<body>
	<div id="div1">
		<span id="SP1>Some aaaa text</span>
	</div>
	<p>Some Text</p>
</body>
</html>`

func TestParsing(t *testing.T) {
	LogLevel = 4
	doc := ParseHtml(structureHtml)
	if doc == nil {
		t.Error("Unparsabel html.")
		t.FailNow()
	}
	exp := []string{"html", "body", "h1", "p", "span", "h2", "div", "p", "p"}
	lines := strings.Split(doc.HtmlRep(0), "\n", -1)
	for i, tag := range exp {
		tag = "<" + tag + ">"
		ls := strings.Trim(lines[i], " \t")
		if ! strings.HasPrefix(ls, tag) {
			t.Error("Expected %s on line %d but got %s.", tag, i, ls)
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
		<span id="SP1">Some aaaa text</span>
	</div>
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
</body>
</html>`


func TestTextcontent(t *testing.T) {
	doc := ParseHtml(SimpleHtml)
	if doc == nil {
		t.Error("Unparsabel html.")
		t.FailNow()
	}
	
	check(doc, "p == Luzern", "plu", t)
	check(doc, "p =D= Luzern", "plu", t)
	check(doc, "span == Chiasso", "sch", t)
	check(doc, "span =D= Chiasso", "sch", t)
	checkN(doc, "p == Chiasso", t)
	check(doc, "p =D= Chiasso", "pch", t)
	
	checkN(doc, "p == AA BB CC DD EE FF GG", t)
	check(doc, "p =D= AA BB CC DD EE FF GG", "nested", t)
	checkN(doc, "div == 123 AA * GG 456", t)
	check(doc, "div =D= 123 AA * GG 456", "div3", t)
}


func BenchmarkParsing(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ParseHtml(SimpleHtml)
	}
}


func TestBasics(t *testing.T) {
	doc := ParseHtml(SimpleHtml)
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

