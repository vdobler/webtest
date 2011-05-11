package suite

import (
	// "./suite"
	"testing"
	"time"
	"fmt"
	"http"
	"os"
	"strconv"
	"strings"
)

var (
	port     = ":54123"
	host     = "http://localhost"
	theSuite *Suite
)

// keep server a life for n seconds after last testcase to allow manual testt the test server...
var testserverStayAlive int64 = 0


var suiteTmpl = `
----------------------
Global
----------------------
GET http://wont.use
RESPONSE
	Status-Code	== 200
CONST
	URL	 %s%s

# Test no 1
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
TAG
	 title == Dummy HTML 1
	 p class=a
	!p class=c
	!p == Wrong.*
	
# Test no 2
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

# Test no 3
----------------------
Binary Test 2
----------------------
GET ${URL}/bin.bin
PARAM
	sc  401
RESPONSE
	Status-Code    ==  401
	Content-Type  ==  application/data
BODY
	Txt  ~=  Hallo Welt!

# Test no 4
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

# Test no 5
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
	
# Test no 6
-------------------------
Plain Post (no Redirect)
-------------------------
POST ${URL}/post
RESPONSE
	Final-Url	==	${URL}/post
BODY
	Txt  ~=  Post Page 

# Test no 7
-------------------------
Post (with Redirect)
-------------------------
POST ${URL}/post
PARAM
	q		html.html
RESPONSE
	Final-Url	==	${URL}/html.html
TAG
	h1 == Dummy Document *
	p class=a == *Braunschweig Weiler

	
# Test no 8
------------------------
Too slow
------------------------
GET ${URL}/html.html
PARAM
	sleep	110
SETTING
	Max-Time	100
`

func TestServer(t *testing.T) {
	go StartHandlers(port, t)
}

func StartHandlers(addr string, t *testing.T) (err os.Error) {
	http.Handle("/html.html", http.HandlerFunc(htmlHandler))
	http.Handle("/bin.bin", http.HandlerFunc(binHandler))
	http.Handle("/post", http.HandlerFunc(postHandler))
	http.Handle("/404.html", http.NotFoundHandler())
	fmt.Printf("\n\nRunning test server on %s\n\n", addr)
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
		<form action="/post" method="post">
			Go to: <input type="text" name="q" value="Search" />
			<input type="submit" />
		</form>
	</div>
	<p class="b">Stupid stuff here.</p>
</body>
`

func htmlHandler(w http.ResponseWriter, req *http.Request) {
	w.SetHeader("Content-Type", "text/html; charset=utf-8")
	w.SetHeader("Fancy-Header", "Important Value")
	w.WriteHeader(200)
	t := req.FormValue("text")
	s := req.FormValue("sleep")
	if ms, err := strconv.Atoi(s); err == nil {
		time.Sleep(1000000 * int64(ms))
	}
	body := fmt.Sprintf(htmlPat, t)
	w.Write([]byte(body))
	w.Flush()
}

func postHandler(w http.ResponseWriter, req *http.Request) {
	w.SetHeader("Content-Type", "text/html; charset=utf-8")
	t := req.FormValue("q")
	if req.Method != "POST" {
		fmt.Printf("========= called /post with GET! ========\n")
	}
	if t != "" {
		w.SetHeader("Location", host+port+"/"+t)
		w.WriteHeader(302)
	} else {
		w.WriteHeader(200)
		body := "<html><body><h1>Post Page</h1><p>t = " + t + "</p></body></html>"
		w.Write([]byte(body))
	}
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

func failed(test *Test, t *testing.T) bool {
	if strings.HasPrefix(test.Status(), "PASSED") {
		t.Logf("Test %s expected to fail but passed.\n%s", test.Title, test.Result)
		t.Fail()
		return false
	}
	return true
}

func TestParsing(t *testing.T) {
	suiteText := fmt.Sprintf(suiteTmpl, host, port)
	p := NewParser(strings.NewReader(suiteText), "suiteText")
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
	theSuite.RunTest(0)
	passed(&theSuite.Test[0], t)
}

func TestBin(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(1)
	passed(&theSuite.Test[1], t)
	theSuite.RunTest(2)
	passed(&theSuite.Test[2], t)
}


func TestSequence(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(3)
	passed(&theSuite.Test[3], t)
}

func TestRandom(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(4)
	passed(&theSuite.Test[4], t)
}

func TestPlainPost(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(5)
	passed(&theSuite.Test[5], t)
}

func TestPost(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(6)
	passed(&theSuite.Test[6], t)
}

func TestTooSlow(t *testing.T) {
	if theSuite == nil {
		t.Fatal("No Suite.")
	}
	theSuite.RunTest(7)
	failed(&theSuite.Test[7], t)
}


func TestStayAlife(t *testing.T) {
	fmt.Printf("Will stay alive for %d seconds.\n", testserverStayAlive)
	time.Sleep(1000000000 * testserverStayAlive)
}



var backgroundSuite = `
----------------------
html
----------------------
GET ${URL}/html.html
SETTING
	Repeat 5
	Sleep  1
	
----------------------
bin
----------------------
GET ${URL}/bin.bin
SETTING
	Repeat 5
	Sleep  1
`

var stressSuite = `
----------------------
Basic
----------------------
GET ${URL}/html.html
RESPONSE
	Status-Code  ==  200
BODY
	 Txt  ~= Braunschweig
TAG
	 title == Dummy HTML 1
	 p class=a
	
----------------------
Binary
----------------------
GET ${URL}/bin.bin
RESPONSE
	Content-Type  ==  application/data
BODY
	Txt  ~=  Hallo Welt!
`

func testPrintStResult(txt string, result StressResult) {
	fmt.Printf("%s: Response Time %5d / %5d / %5d (min/avg/max). Status %2d / %2d / %2d (err/pass/fail). %2d / %2d (tests/checks).\n",
				txt, result.MinRT, result.AvgRT, result.MaxRT, result.Err, result.Pass, result.Fail, result.N, result.Total)
}

func TestStresstest(t *testing.T) {
	LogLevel = 1
	bgText := strings.Replace(backgroundSuite, "${URL}", host+port, -1)
	suiteText := strings.Replace(stressSuite, "${URL}", host+port, -1)
	p := NewParser(strings.NewReader(bgText), "background")
	var background *Suite
	var err os.Error
	background, err = p.ReadSuite()
	if err != nil {
		t.Fatalf("Cannot read suite: %s", err.String())
	}
	p = NewParser(strings.NewReader(suiteText), "suite")
	var suite *Suite
	suite, err = p.ReadSuite()
	if err != nil {
		t.Fatalf("Cannot read suite: %s", err.String())
	}
	
	LogLevel = 3
	r0 := suite.Stresstest(background, 0, 3, 100)
	r10 := suite.Stresstest(background, 10, 3, 100)
	r30 := suite.Stresstest(background, 30, 2, 100)
	r60 := suite.Stresstest(background, 60, 1, 100)
	r100 := suite.Stresstest(background, 100, 1, 100)
	r150 := suite.Stresstest(background, 150, 1, 100)
	r200 := suite.Stresstest(background, 200, 5, 100)
	
	testPrintStResult("Load   0", r0)
	testPrintStResult("Load  10", r10)
	testPrintStResult("Load  30", r30)
	testPrintStResult("Load  60", r60)
	testPrintStResult("Load 100", r100)
	testPrintStResult("Load 150", r150)
	testPrintStResult("Load 200", r200)
	if r0.Total <=0 || r0.N <= 0 {
		t.Error("No tests run without load")
		t.FailNow()
	}
	if r0.Fail > 0 || r0.Err > 0 {
		t.Error("Failures without load")
		t.FailNow()
	}	

	// There will be failures in the 200 run....
	if r200.Total <=0 || r200.N <= 0 {
		t.Error("No tests run with load 200")
		t.FailNow()
	}
	if r200.Fail == 0 || r200.Err == 0 {
		t.Error("Expected Failures at load of 200!")
	}	
	
}
