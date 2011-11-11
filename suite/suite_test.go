package suite

import (
	"fmt"
	"flag"
	"html"
	"http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"launchpad.net/gocheck"
)

var (
	skipStress bool = true
)

// keep server a life for n seconds after last testcase to allow manual testt the test server...
var testserverStayAlive = flag.Int64("keepalive", 0, "Keep server running for n seconds")

// Our test suite
func TestGoCheckInit(t *testing.T) { gocheck.TestingT(t) }

type S struct{}

var _ = gocheck.Suite(&S{})

func (s *S) SetUpSuite(c *gocheck.C) {
	go func() {
		http.Handle("/html.html", http.HandlerFunc(htmlHandler))
		http.Handle("/bin.bin", http.HandlerFunc(binHandler))
		http.Handle("/post", http.HandlerFunc(postHandler))
		http.Handle("/404.html", http.NotFoundHandler())
		http.Handle("/cookie.html", http.HandlerFunc(cookieHandler))
		http.Handle("/redirect/", http.HandlerFunc(redirectHandler))
		fmt.Printf("\nRunning test server on http://localhost:54123\n")
		if err := http.ListenAndServe(":54123", nil); err != nil {
			c.Fatalf("Cannot run test server on port: %s", err.String())
		}
	}()
	time.Sleep(2e8)
	c.Succeed()
}
func (s *S) TearDownSuite(c *gocheck.C) {
	fmt.Printf("Will stay alive for %d seconds.\n", testserverStayAlive)
	time.Sleep(1e9 * (*testserverStayAlive))
}

//
// ######################################################################
//
// Webserver
//

var htmlPat = `<!DOCTYPE html>
<html>
  <head>
    <title>Dummy HTML 1</title>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
  </head>
  <body>
	<h1>Dummy Document for html based testst</h1>
	<p class="a">Some fancy text Braunschweig Weiler</p>
	%s
	<div>
		<form action="/post" method="post" enctype="multipart/form-data">
			Go to: <input type="text" name="q" value="Search" />
			File <input type="file" name="file-upload">
			<input type="submit" />
		</form>
	</div>
	%s
	<p class="b">Stupid stuff here.</p>
	<span>UTF-8 Umlaute äöüÄÖÜ Euro €</span>
`

var xCounter = map[string]int{"foo": 0, "bar": 0, "baz": 0} // used for tries

func htmlHandler(w http.ResponseWriter, req *http.Request) {
	if log, err := os.OpenFile("log.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666); err == nil {
		txt := req.FormValue("tolog")
		log.WriteString("Stamp[html] Sehr Wichtig\nStamp[html] Hubba Buba\n")
		if len(txt) > 0 {
			log.WriteString(txt + "\n")
		}
		log.Sync()
		log.Close()
		trace("Wrote to log.log")
	} else {
		panic(err.String())
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Fancy-Header", "Important Value")
	w.WriteHeader(200)
	t := req.FormValue("text")
	s := req.FormValue("sleep")
	x := req.FormValue("xxx")
	t2 := ""
	if x == "foo" || x == "bar" || x == "baz" {
		xCounter[x] = xCounter[x] + 1
		if xCounter[x] < 4 { // Fifth run succeeds....
			t2 += "\n<h2>Still Running...</h2>"
		} else {
			t2 += "\n<h2 class=\"okay\">Finished.</h2>"
		}
	}
	if ms, err := strconv.Atoi(s); err == nil {
		time.Sleep(1000000 * int64(ms))
	}
	if len(req.Cookies()) > 0 {
		t2 += "\n<a href=\"/bin.bin\" title=\"TheCookieValue\">" + req.Cookies()[0].Name + " = " + req.Cookies()[0].Value + "</a>"
	}
	body := fmt.Sprintf(htmlPat, html.EscapeString(t), t2)
	if req.FormValue("badhtml") == "bad" {
		body += "</h3></html>"
	} else {
		body += "</body></html>"
	}
	w.Write([]byte(body))
}

func postHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if cv := req.FormValue("cookie"); cv != "" {
		trace("postHandler recieved param cookie %s.", cv)
		cp := strings.SplitN(cv, "=", 2)
		if cp[1] != "-DELETE-" {
			exp := time.SecondsToUTC(time.UTC().Seconds() + 7*24*3600).Format(http.TimeFormat) // Now + 7 days
			w.Header().Set("Set-Cookie", fmt.Sprintf("%s=%s; Path=/de/index; expires=%s; Domain=my.domain.org; Secure;", cp[0], cp[1], exp))
		} else {
			trace("post-handler: Deleting cookie %s\n", cp[0])
			w.Header().Set("Set-Cookie", fmt.Sprintf("%s=%s; Path=/de/index; MaxAge=-1; Domain=my.domain.org; Secure;", cp[0], "X"))
		}
	}
	t := req.FormValue("q")
	if req.Method != "POST" {
		fmt.Printf("====== called /post with GET! ======\n")
	}

	_, header, err := req.FormFile("datei")
	if err == nil {
		info("Recieved datei: %s. %v", header.Filename, header.Filename == "file äöü 1.txt")
	}

	if t != "" {
		// w.Header().Set("Location", "http://localhost:54123/"+t)
		// w.Header().Set("Location", "localhost:54123/"+t)
		w.Header().Set("Location", "/"+t)
		w.WriteHeader(302)
	} else {
		text := req.FormValue("text")
		w.WriteHeader(200)
		body := "<html><body><h1>Post Page</h1><p>t = " + html.EscapeString(text) + "</p></body></html>"
		w.Write([]byte(body))
	}
}

func lastPath(req *http.Request) string {
	p := req.URL.Path
	if n := strings.LastIndex(p, "/"); n > -1 {
		return p[n+1:]
	}

	return ""
}

// Multiredirect:
//  /redirect  -->  /redirect/first         cookie "rda" on /
//  /redirect/first --> /redirect/second    cookie "rdb" on /redirect
//  /redirect/second --> /redirect/third    cookie "rdc" on /otherpath
//  /redirect/third --> /redirect/fourth    cookie "clearme" maxage = 0
//  /redirect/fourth --> /redirect/last     200 iff rda, rdb present and rdc and clearme absent; 500 else
//

func redirectHandler(w http.ResponseWriter, req *http.Request) {

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch lastPath(req) {
	case "redirect", "":
		w.Header().Set("Location", "http://localhost:54123/redirect/first")
		w.Header().Add("Set-Cookie", "rda=rda; Path=/")
		w.Header().Add("Set-Cookie", "clearme=eraseme; Path=/")
		w.WriteHeader(302)
		return
	case "first":
		w.Header().Set("Location", "http://localhost:54123/redirect/second")
		w.Header().Set("Set-Cookie", "rdb=rdb; Path=/redirect")
		w.WriteHeader(302)
		return
	case "second":
		w.Header().Set("Location", "http://localhost:54123/redirect/third")
		w.Header().Set("Set-Cookie", "rdc=rdc; Path=/otherpath")
		w.WriteHeader(302)
		return
	case "third":
		w.Header().Set("Location", "http://localhost:54123/redirect/fourth")
		exp := time.SecondsToUTC(time.UTC().Seconds() - 10000).Format(http.TimeFormat)
		w.Header().Set("Set-Cookie", "clearme=; Path=/; Max-Age=0; Expires="+exp)
		w.WriteHeader(302)
		return
	case "fourth":
		w.Header().Set("Location", "http://localhost:54123/redirect/last")
		rdav, rdae := req.Cookie("rda")
		rdbv, rdbe := req.Cookie("rdb")
		_, rdce := req.Cookie("rdc")
		_, cme := req.Cookie("clearme")
		if rdae == nil && rdav.Value == "rda" && rdbe == nil && rdbv.Value == "rdb" && rdce != nil && cme != nil {
			w.WriteHeader(302)
		} else {
			w.WriteHeader(500)
			body := "<html><body><h1>Wrong cookies</h1><pre>"
			for _, c := range req.Cookies() {
				body += fmt.Sprintf("\n%#v\n", *c)
			}
			body += "</pre></body></html>"
			w.Write([]byte(body))
		}
		return
	case "last":
		w.WriteHeader(200)
		w.Write([]byte("<html><body><h1>No more redirects.</h1></body></html>"))
		return
	default:
		w.WriteHeader(404)
		w.Write([]byte("<html><body><h1>Oooops..." + lastPath(req) + "</h1></body></html>"))
		return
	}
}

func binHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/data")
	w.Header().Set("Fancy-Header", "Arbitary Value")
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
}

func code(s string) string {
	s = strings.Replace(s, "&", "&amp;", -1)
	s = strings.Replace(s, "<", "&lt;", -1)
	s = strings.Replace(s, ">", "&gt;", -1)
	s = strings.Replace(s, "\"", "&quot;", -1)
	return "<code>" + s + "</code>"
}

func cookieHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if cn := req.FormValue("set"); cn != "" {
		cv, cp := req.FormValue("val"), req.FormValue("pat")
		trace("cookieHandler recieved cookie %s=%s; path=%s.", cn, cv, cp)
		w.Header().Set("Set-Cookie", fmt.Sprintf("%s=%s; Path=/de/index; Domain=my.domain.org; Secure;", cn, cv))
	}

	if t := req.FormValue("goto"); t != "" {
		w.Header().Set("Location", "localhost:54123/"+t)
		w.WriteHeader(302)
	} else {
		w.WriteHeader(200)
		body := "<html><head><title>Cookies</title></head>\n<body><h1>All Submitted Cookies</h1>"
		for _, cookie := range req.Cookies() {
			body += "<div class=\"cookies\">\n"
			body += "  <ul>\n"
			body += "   <li>" + cookie.Name + " :: " + cookie.Value + "</li>\n"
			body += "  </ul>\n"
			body += "</div>\n"
		}
		body += "</body></html>"
		w.Write([]byte(body))
	}
}

//
// #######################################################################################
//
// Stresstesting
//
var backgroundSuite = `
----------------------
html
----------------------
GET http://localhost:54123/html.html
SETTING
	Repeat 5
	Sleep  1
	
----------------------
bin
----------------------
GET http://localhost:54123/bin.bin
SETTING
	Repeat 5
	Sleep  1
`

var stressSuite = `
----------------------
Basic
----------------------
GET http://localhost:54123/html.html
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
GET http://localhost:54123/bin.bin
RESPONSE
	Content-Type  ==  application/data
BODY
	Txt  ~=  Hallo Welt!
`

func testPrintStResult(txt string, result StressResult) {
	fmt.Printf("%s: Response Time %5d / %5d / %5d (min/avg/max). Status %2d / %2d / %2d (err/pass/fail). %2d / %2d (tests/checks).\n",
		txt, result.MinRT, result.AvgRT, result.MaxRT, result.Err, result.Pass, result.Fail, result.N, result.Total)
}

func (s *S) TestStresstest(c *gocheck.C) {
	if skipStress {
		c.Skip("Skipped stresstest")
	}
	LogLevel = 1
	p := NewParser(strings.NewReader(backgroundSuite), "background")
	var background *Suite
	var err os.Error
	background, err = p.ReadSuite()
	if err != nil {
		c.Fatalf("Cannot read suite: %s", err.String())
	}
	p = NewParser(strings.NewReader(stressSuite), "suite")
	var suite *Suite
	suite, err = p.ReadSuite()
	if err != nil {
		c.Fatalf("Cannot read suite: %s", err.String())
	}

	r0 := suite.Stresstest(background, 0, 3, 100)
	r10 := suite.Stresstest(background, 10, 3, 100)
	r30 := suite.Stresstest(background, 30, 2, 100)
	time.Sleep(100000000)
	r60 := suite.Stresstest(background, 60, 1, 100)
	time.Sleep(100000000)
	r100 := suite.Stresstest(background, 100, 1, 100)
	time.Sleep(200000000)
	r150 := suite.Stresstest(background, 150, 1, 100)
	time.Sleep(200000000)
	r200 := suite.Stresstest(background, 200, 5, 100)
	time.Sleep(200000000)

	testPrintStResult("Load    0", r0)
	testPrintStResult("Load   10", r10)
	testPrintStResult("Load   30", r30)
	testPrintStResult("Load   60", r60)
	testPrintStResult("Load  100", r100)
	testPrintStResult("Load  150", r150)
	testPrintStResult("Load  200", r200)
	if r0.Total <= 0 || r0.N <= 0 {
		c.Error("No tests run without load")
		c.FailNow()
	}
	if r0.Fail > 0 || r0.Err > 0 {
		c.Error("Failures without load")
		c.FailNow()
	}

}

//
// ######################################################################
//
// Individual Tests
//

// Helper functions to test one test
func printresults(test *Test) {
	fmt.Printf("Results of Test %s:\n--------------------------------------\n:", test.Title)
	for _, r := range test.Result {
		fmt.Println(r.AsText())
	}
}
func runsingletest(name, st string, no int, ep, ef, ee int, c *gocheck.C) {
	parser := NewParser(strings.NewReader(st), name)
	suite, err := parser.ReadSuite()
	if err != nil {
		c.Fatalf("Cannot read suite %s: %s", name, err.String())
		return
	}
	if len(suite.Test) <= no {
		titles := ""
		for i := range suite.Test {
			titles += ": " + suite.Test[i].Title + " : "
		}
		c.Fatalf("Suite %s has only %d tests [%s]. %d required to run.", name, len(suite.Test), titles, no)
		return
	}

	suite.RunTest(no)
	p, f, e := suite.Test[no].Stat()

	// Stopp test if error mismatch
	if e != ee {
		printresults(&suite.Test[no])
		c.Fatalf("Wrong no of errors: expected %d, obtained %d.", ee, e)
	}

	if p != ep || f != ef {
		printresults(&suite.Test[no])
		printresults(&suite.Test[no])
	}

	if p+f != ep+ef {
		fmt.Printf("\nResult: %#v\n\n", suite.Test[no].Result)
		c.Fatalf("Wrong no of executed tests: expected %d (p:%d, f:%d), obtained %d (p:%d, f:%d).",
			ep+ef, ep, ef, p+f, p, f)
	}
	c.Check(p, gocheck.Equals, ep, gocheck.Bug("Wrong number of passed tests"))
	c.Check(f, gocheck.Equals, ef, gocheck.Bug("Wrong number of failed tests"))
}

// A very basic test
func (s *S) TestBasic(c *gocheck.C) {
	st := `
----------------------
Basic Test
----------------------
GET http://localhost:54123/html.html
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
`
	runsingletest("Basic Test", st, 0, 20, 0, 0, c)
}

//
func (s *S) TestBinary(c *gocheck.C) {
	st := `
----------------------
Binary Test 1
----------------------
GET http://localhost:54123/bin.bin
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
GET http://localhost:54123/bin.bin
PARAM
	sc  :=  401
RESPONSE
	Status-Code    ==  401
	Content-Type  ==  application/data
BODY
	Txt  ~=  Hallo Welt!
`
	runsingletest("Binary Test 1", st, 0, 6, 0, 0, c)
	runsingletest("Binary Test 2", st, 1, 3, 0, 0, c)
}

//
func (s *S) TestSequence(c *gocheck.C) {
	st := `
----------------------
Sequence Test
----------------------
GET http://localhost:54123/html.html
SEQ
	name  :=  Anna Berta Claudia "Doris Dagmar" Emmely
PARAM
	text  :=  ${name}
BODY
	Txt  ~=  ${name}
SETTING
	Repeat	:=  3
`
	runsingletest("Sequence Test", st, 0, 3, 0, 0, c)
}

//
func (s *S) TestRandom(c *gocheck.C) {
	st := `
----------------------
Random Test
----------------------
GET http://localhost:54123/html.html
RAND
	name :=  Anna Berta Claudia "Doris Dagmar" Emmely
PARAM
	text :=  ${name}
BODY
	Txt  ~=  ${name}
SETTING
	Repeat	:=  10
`
	runsingletest("Random Test", st, 0, 10, 0, 0, c)
}

//
func (s *S) TestTooSlow(c *gocheck.C) {
	st := `
------------------------
Too slow
------------------------
GET http://localhost:54123/html.html
PARAM
	sleep  :=  110
SETTING
	Max-Time  :=  100
`
	runsingletest("Too slow", st, 0, 0, 1, 0, c)
}

//
func (s *S) TestPost(c *gocheck.C) {
	st := `
-------------------------
Plain Post (no Redirect)
-------------------------
POST http://localhost:54123/post
RESPONSE
	Final-Url == http://localhost:54123/post
BODY
	Txt  ~=  Post Page 

-------------------------
Post (with Redirect)
-------------------------
POST http://localhost:54123/post
PARAM
	q  := html.html
RESPONSE
	Final-Url == http://localhost:54123/html.html
TAG
	h1 == Dummy Document *
	p class=a == *Braunschweig Weiler
SETTING
	Dump := 1

-------------------------
Multipart Post
-------------------------
POST:mp http://localhost:54123/post
RESPONSE
	Final-Url == http://localhost:54123/post
PARAM
	text  :=  ABCDwxyz1234
BODY
	Txt  ~=  Post Page
	Txt  ~= ABCDwxyz1234
SETTING
	Dump := 1
`
	runsingletest("Plain Post", st, 0, 2, 0, 0, c)
	runsingletest("Post", st, 1, 3, 0, 0, c)
	runsingletest("Multipart Post", st, 2, 3, 0, 0, c)
}

//
func (s *S) TestEncoding(c *gocheck.C) {
	st := `
----------------------
Encoding 1
----------------------
GET http://localhost:54123/html.html
PARAM
	text  :=  "€ & <ÜÖÄ> = üöa" 
BODY
	Txt  ~=  UTF-8 Umlaute äöüÄÖÜ Euro €
	Txt  ~=  € &amp; &lt;ÜÖÄ&gt; = üöa

-------------------------
Encoding 2
-------------------------
POST http://localhost:54123/post
PARAM
	datei  :=  @file:condition.go
	text   :=  "€ & <ÜÖÄ> = üöa"
RESPONSE
	Final-Url  ==  http://localhost:54123/post
BODY
	Txt  ~=  Post Page 
	Txt  ~=  € &amp; &lt;ÜÖÄ&gt; = üöa
`
	runsingletest("Encoding 1", st, 0, 2, 0, 0, c)
	runsingletest("Encoding 2", st, 1, 3, 0, 0, c)
}

//
func (s *S) TestNowVariable(c *gocheck.C) {
	st := `
----------------------
Now
----------------------
GET http://localhost:54123/html.html
PARAM
	text :=  "Its now ${NOW + 3hours -15minutes | 02.01.2006 15:04:05} oclock"
BODY
	Txt  ~=  Its now ${NOW + 3hours -15minutes | 02.01.2006 15:04:05} oclock
SETTING
	Repeat  :=  5
	Sleep   :=  1500
`
	runsingletest("Now", st, 0, 5, 0, 0, c)
}

//
func (s *S) TestTries(c *gocheck.C) {
	st := `
----------------------
Passing Gate
----------------------
GET http://localhost:54123/html.html
PARAM
	xxx  :=  ${val} 
SEQ
	val  :=  foo bar
TAG
	h2 class=okay == Finished.
SETTING
	Dump   :=  1
	Repeat :=  1
	Tries  :=  5

----------------------
Failing Gate
----------------------
GET http://localhost:54123/html.html
PARAM
	xxx  :=  baz
TAG
	h2 class=okay == Finished.
SETTING
	Dump   := 1
	Tries  := 3
`
	runsingletest("Passing Gate", st, 0, 1, 0, 0, c)
	runsingletest("Failing Gate", st, 1, 0, 1, 0, c)
}

//
func (s *S) TestLogfiles(c *gocheck.C) {
	st := `
----------------------
Logfile (Pass)
----------------------
GET http://localhost:54123/html.html
PARAM
	tolog := Logfile (Pass)
BEFORE 
	bash -c "echo Stamp0 ABC > log.log; echo Stamp1 FALSCH >> log.log; echo Stamp2 Wichtig >> log.log"
SETTING
	Sleep  :=  150
LOG
	log.log ~= Wichtig
	log.log _= Stamp[html]
	log.log _= Stamp[html] Sehr
	! log.log ~= Komisch
	! log.log ~= Fail

----------------------
Logfile (Fail)
----------------------
GET http://localhost:54123/html.html
PARAM
	tolog := "Logfile (Fail)"
BEFORE 
	bash -c "echo Stamp0 ABC > log.log; echo Stamp1 FALSCH >> log.log; echo Stamp2 Wichtig >> log.log"
SETTING
	Sleep  :=  150
LOG
	! log.log ~= Wichtig
	log.log =_ Wichtig
	! log.log ~= Pass
	log.log _= Logfile (fail)
	log.log ~= Komisch
`
	runsingletest("Logfile pass", st, 0, 5, 0, 0, c)
	runsingletest("Logfile fail", st, 1, 2, 3, 0, c)
}

//
func (s *S) TestFilePost(c *gocheck.C) {
	st := `
-------------------------
Filepost
-------------------------
POST http://localhost:54123/post
PARAM
	datei  :=  @file:condition.go
RESPONSE
	Final-Url  ==  http://localhost:54123/post
BODY
	Txt  ~=  Post Page 
`
	runsingletest("Filepost", st, 0, 2, 0, 0, c)
}

//
func (s *S) TestGlobalSubst(c *gocheck.C) {
	st := `
----------------------
Global
----------------------
GET http://wont.use
RESPONSE
	Status-Code	== 200
CONST
	URL         :=  http://localhost:54123
	SomeGlobal  :=  GlobalValue

----------------------
Variable Subst
----------------------
GET http://localhost:54123/html.html
PARAM
	text  := ${SomeGlobal}
BODY
	Txt  ~=  GlobalValue
`
	runsingletest("Global Subst", st, 0, 2, 0, 0, c)
}

//
func (s *S) TestMultipleParameters(c *gocheck.C) {
	st := `
----------------------
Multiple Parameters
----------------------
GET http://localhost:54123/html.html
PARAM
	text  :=  Hund Katze "Anna Berta"
RESPONSE
	Status-Code == 200

`
	runsingletest("Multiple parameters", st, 0, 1, 0, 0, c)
}

//
func (s *S) TestHTMLValidation(c *gocheck.C) {
	st := `
----------------------
HTML Validation Pass
----------------------
GET http://localhost:54123/html.html
VALIDATE
	html
SETTING
	Dump := 1

----------------------
HTML Validation Fail
----------------------
GET http://localhost:54123/html.html
PARAM
	badhtml := bad
VALIDATE
	html
SETTING
	Dump := 1
`
	runsingletest("HTML Validation", st, 0, 1, 0, 0, c)
	runsingletest("HTML Validation", st, 1, 0, 1, 0, c)
}

//
// ######################################################################
// Complicated stuff
//

// Cookie suite
func (s *S) TestCookieSuite(c *gocheck.C) {
	c.Skip("CookieSuite currentlx broken")
	return

	st := `
----------------------
Global
----------------------
GET http://wont.use
CONST
	URL  := http://localhost:54123

----------------------
Display Cookies
----------------------
GET ${URL}/cookie.html
SEND-COOKIE
	MyFirst     := MyFirstCookieValue
	Sessionid   := abc123XYZ
	JSESSIONID  := 5AE613FC082DEB79484C774677651164
	
TAG
	li == Sessionid :: abc123XYZ
	li == MyFirst :: MyFirstCookieValue
	li == JSESSIONID :: 5AE613FC082DEB79484C774677651164
SETTING
	Dump := 1

	
----------------------
Login
----------------------
POST ${URL}/post
PARAM
	# cookie parameter is added to response header by /post handler
	cookie :=  TheSession=randomsessionid
RESPONSE
	Final-Url  ==  ${URL}/post
SET-COOKIE
	TheSession         ~=  randomsessionid
	TheSession:::Value   ==  randomsessionid
	TheSession::/de/    ==  randomsessionid
	TheSession:::Secure  ==  true
	TheSession:my.domain.org::  ==  my.domain.org
	TheSession:::Expires  <  ${NOW + 1 year}
	TheSession:::Expires  >  ${NOW + 1 day}
	
BODY
	Txt  ~=  Post Page 
SETTING
	# Store recieved Cookies in Global
	Keep-Cookies :=  1
	Dump         :=  1
	
	
---------------------
Access
---------------------
GET ${URL}/cookie.html
TAG
	li == TheSession :: randomsessionid

SETTING
	Dump := 1


----------------------
Logout
----------------------
POST ${URL}/post
PARAM
	# cookie parameter is added to response header by /post handler
	cookie  := TheSession=-DELETE-
RESPONSE
	Final-Url  ==  ${URL}/post
SET-COOKIE
	TheSession:::MaxAge   <   0
	!TheSession         ~=  randomsessionid
	TheSession:::Delete ==  true

BODY
	Txt  ~=  Post Page 
SETTING
	# Store recieved/deleted cookies in Global
	Keep-Cookies  := 1
	Dump          := 1
	
---------------------
Failing Access
---------------------
GET ${URL}/cookie.html
TAG
	! li == *TheSession*

`
	runsingletest("Display Cookies", st, 0, 3, 0, 0, c)
	runsingletest("Login", st, 1, 9, 0, 0, c)
	runsingletest("Access", st, 2, 1, 0, 0, c)
	runsingletest("Logout", st, 3, 5, 0, 0, c)
	runsingletest("Failing Access", st, 4, 1, 0, 0, c)
}

// Redirect Chain with cookies
func (s *S) TestRedirectChain(c *gocheck.C) {
	st := `
----------------------
Global
----------------------
GET http://wont.use
CONST
	URL := http://localhost:54123

----------------------
Redirect Chain
----------------------
GET ${URL}/redirect/
SEND-COOKIE
	clearme  :=  somevalue
	theOther :=  othervalue

RESPONSE
	Final-Url    ==  ${URL}/redirect/last
	Status-Code  ==  200
SET-COOKIE
	rda   ==  rda
	rdb::/redirect   ==  rdb
	rdc::/otherpath  ==  rdc
	clearme:::Delete == true
`
	runsingletest("Redirect Chain", st, 0, 6, 0, 0, c)
}

/*
//
func (s *S) Test(c *gocheck.C) {
	st := `
	runsingletest("", st, 0, 20, 0, 0, c)
`
}
*/
