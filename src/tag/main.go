package main

import (
	"fmt"
	"html"
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
	  <a href="index.html" class="ext int special">My first Link</a>
	</p>
	<p>
	  <img src="bild.jpg" alt="i &lt; i+1" >
	  <div id="theDiv">
		<h2>Inside</h2>
		<p>
			<span class="code" title="V&amp;S">Go!</span> Rules
		</p>
	  </div>
	</p>
  </div>
</body>
</html>`



func main() {
	doc := tag.ParseHtml(html1)

	ts := tag.TagSpec{Name: "a", 
					  Content:  "My * Link",
					  Classes:  []string{"ext", "int"},
					  XClasses: []string{"wrong"},
					  XAttr:    []html.Attribute{html.Attribute{Key: "target", Val:tag.IGNORED}}}

	fmt.Printf("\nTagSpec:\n%s\n", ts.String())
	tag.FindTag(&ts, doc)
	
	fmt.Printf("\nTagSpec:\n%s\n", ts.String())
	ts = tag.TagSpec{Name: "div", 
					 Attr: []html.Attribute{html.Attribute{Key: "id", Val: "theDiv"}}}
	if div := tag.FindTag(&ts, doc); div != nil {
		fmt.Printf("%s\n", div.Html(0))
	}

	tsp := tag.ParseSimpleTagSpec("p class=ext !style id=myId !lang=en =D= Some Text")
	fmt.Printf("\n%s\n", tsp.String())
	
	tsp = tag.ParseTagSpec("p\ndiv id=theDiv")
	if node := tag.FindTag(tsp, doc); node!= nil {
		fmt.Printf("%s\n", node.Html(0))
	}
	
	fmt.Printf("\n%s\n", tsp.String())
	
	fmt.Printf("\n%s\n", doc.RealHtml())
	
}