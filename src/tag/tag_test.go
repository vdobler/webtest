package tag

import (
	"fmt"
	// "bufio"
	// "os"
	"html"
	// "log"
	// "strings"
	"testing"
)

var html1 = `<!DOCTYPE html>
<html>
<head>
  <title id=\"hund\" class=\"imp new\"target=\"\">Hallo Welt</title>
  <meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\"/>
</head>
<body>
  <h1 class=\"main\">Der Titel</h1>
  <div id=\"myId\">
    <p>
	  <a href=\"index.html\" target=\"blank\">Open <span>New</span> Window<a>
	</p>
	<p>
	  <a href=\"index.html\">No Window<a>
	</p>
  </div>
</body>
</html>`



func TestMatch(t *testing.T) {
	ts := TagSpec{Name: "title", 
				  Content: "Hallo Welt", 
				  Classes: []string{"imp", "new"},
				  XClasses: []string{"old"},
				  Attr: []html.Attribute{html.Attribute{Key:"id", Val: "hund"}},
				  XAttr: []html.Attribute{html.Attribute{Key:"targett", Val: IGNORED}, html.Attribute{Key:"name", Val: "vod"}} }
	fmt.Printf("TagSpec : " + ts.String() + "\n")


}