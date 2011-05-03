package suite

import (
	// "./suite"
	"testing"
	// "regexp"
	"fmt"
	"http"
	"os"
	"strconv"
	"strings"
)

var (
	port = ":54123"
	host = "http://localhost"
	theSuite *Suite
)

var suiteTmpl = `
----------------------
Global
----------------------
GET http://wont.use
RESPONSE
	StatusCode	== 200
CONST
	URL	 %s%s

----------------------
Basic Test
----------------------
GET ${URL}/html.html
RESPONSE
	 Fancy-Header  == Important Value
	!Fancy-Header  == Wrong
	 Fancy-Header  ~= Important Value
	 Fancy-Header  ~= Value
	 Fancy-Header  _= Important
	 Fancy-Header  _= Important Value
	!Fancy-Header  _= Value
	!Fancy-Header  =_ Important
	 Fancy-Header  =_ Important Value
	 Fancy-Header  =_ Value
	 Fancy-Header  /= Important.*
	 Fancy-Header  /= ^Imp.*lue$
	!Fancy-Header  /= Wrong
	!Fancy-Header  /= ^Wro.*ng$
BODY
	 Txt  ~= Braunschweig
	 # 5765696c6572 = hex(Weiler)
	 Bin  ~= 5765696c6572
	 Tag title == Dummy HTML 1
	 Tag p class=a
	!Tag p class=c
	!Tag p == Wrong.*
	
----------------------
Binary Test 1
----------------------
GET ${URL}/bin.bin
RESPONSE
	Content-Type  ==  application/data
BODY
	Txt  ~=  Hallo Welt!
	Bin  ==  010748616c6c6f2057656c74210aFFFE
	Bin  _=  01074861
	Bin  =_  FFFE
	Bin  ~=  48616c

----------------------
Binary Test 2
----------------------
GET ${URL}/bin.bin
PARAM
	sc  401
RESPONSE
	StatusCode    ==  401
	Content-Type  ==  application/data
BODY
	Txt  ~=  Hallo Welt!

----------------------
Sequence Test
----------------------
GET ${URL}/html.html
SEQ
	name   Anna Berta Claudia "Doris Dagmar" Emmely
PARAM
	text ${name}
BODY
	Txt  ~=  ${name}
SETTING
	Repeat	3

----------------------
Random Test
----------------------
GET ${URL}/html.html
RAND
	name   Anna Berta Claudia "Doris Dagmar" Emmely
PARAM
	text ${name}
BODY
	Txt  ~=  ${name}
SETTING
	Repeat	3
`

func TestServer(t *testing.T) {
	go StartHandlers(port, t)
}

func StartHandlers(addr string, t *testing.T) (err os.Error) {
	http.Handle("/html.html", http.HandlerFunc(htmlHandler))
	http.Handle("/bin.bin", http.HandlerFunc(binHandler))
	http.Handle("/404.html", http.NotFoundHandler())
    err = http.ListenAndServe(addr, nil)
    if err != nil {
        fmt.Printf("Cannot run test server on port %s.\nFAIL\n", addr)
		os.Exit(1)
    }
	return
}

var htmlPat = `<!DOCTYPE html>
<html><head><title>Dummy HTML 1</title></head>
<body>
	<h1>Dummy Document for html based testst</h1>
	<p class="a">Some fancy text Braunschweig Weiler</p>
	%s
	<div>
		<input type="text" name="q" value="Search" />
	</div>
	<p class="b">Stupid stuff here.</p>
</body>
`

func htmlHandler(w http.ResponseWriter, req *http.Request) {
	w.SetHeader("Content-Type", "text/html; charset=utf-8")
	w.SetHeader("Fancy-Header", "Important Value")
	w.WriteHeader(200)
	t := req.FormValue("text")
	body := fmt.Sprintf(htmlPat, t)
	w.Write([]byte(body))
	w.Flush()
}

func binHandler(w http.ResponseWriter, req *http.Request) {
	w.SetHeader("Content-Type", "application/data")
	w.SetHeader("Fancy-Header", "Arbitary Value")
	var c int = 200
	if sc := req.FormValue("sc"); sc != "" {
		var e os.Error
		c, e = strconv.Atoi(sc)
		if e != nil {
			c = 500
		}
	}
	w.WriteHeader(c)
	w.Write([]byte("\001\007Hallo Welt!\n\377\376"))
	w.Flush()
}

func passed(test *Test, t *testing.T) bool {
 	if !strings.HasPrefix(test.Status(), "PASSED") {
		t.Logf("Result from test %s:\n%s", test.Title, test.Result)
		t.Fail()
		return false
	}
	return true
}


func TestParsing(t *testing.T) {
	suiteText := fmt.Sprintf(suiteTmpl, host, port)
	p := NewParser(strings.NewReader(suiteText))
	var err os.Error
	theSuite, err = p.ReadSuite() 
	if err != nil {
		t.Fatalf("Cannot read suite: %s", err.String())
	}
}



func TestBasic(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(1)
	passed(&theSuite.Test[1], t)
}

func TestBin(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(2)
	passed(&theSuite.Test[2], t)
	theSuite.RunTest(3)
	passed(&theSuite.Test[3], t)
}



func TestSequence(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(4)
	passed(&theSuite.Test[4], t)
}

func TestRandom(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(5)
	passed(&theSuite.Test[5], t)
}



