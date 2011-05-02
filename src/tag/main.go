package main

import (
	"fmt"
	// "html"
	_ "strings"
	"./tag"
)

var html1 = `<html>
<head>
	<title>Hallo Welt</title>
	<meta name="keywords" content="Test, Html-Test, Web, Unittest">
</head>
<body>
  <h1>Der Titel</h1>
  <div>
    <p>
	  Some Text
	  <a href="index.html" class="ext int special" target="blank">My first Link</a>
	</p>
	<p>
	  <img src="bild.jpg" alt="i &lt; i+1" >
	  <div id="theDiv">
		<h2>Inside</h2>
		<p>
			<span class="code" title="V&amp;S">Go!</span> Rules
		</p>
	  </div>
	  <p id="myId" class="ext"> Some <i>Text</i> </p>
	</p>
  </div>
</body>
</html>`


func test(doc *tag.Node, spec string) {
	ts := tag.ParseTagSpec(spec)
	fmt.Printf("\n\n---------------------\nTagSpec:\n%s\n", ts.String())
	node := tag.FindTag(ts, doc)
	if node != nil {
		fmt.Printf("Found!\n%s\n", node.HtmlRep(0))
	} else {
		fmt.Printf("NOT FOUND!\n")
	}
	fmt.Printf("-----------------------\n")
}


func main() {
	doc := tag.ParseHtml(html1)

	test(doc, "a class=ext class=int !class=wrong target !title == My * Link")

	test(doc, "div id=theDiv")

	test(doc, "p class=ext !style id=myId !lang=en =D= Some Text")

	test(doc, "p\n  div id=theDiv")

	test(doc, "div\n  p\n    img\n    div\n      h2\n      p\n    p\n    h2\n      span")

}
