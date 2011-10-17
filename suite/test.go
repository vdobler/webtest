package suite

import (
	"encoding/base64"
	"exec"
	"fmt"
	"http"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"strconv"
	"time"

	"github.com/vdobler/webtest/tag"
	"url"
)

var (
	BenchTolerance float32 = 1.3 // 30% of test may fail during benchmarking without aborting the benchmark.
)

type Error string

func (e Error) String() string {
	return string(e)
}

// Test collects all information about one test to perform, that is one URL fetched
// and conditions tested.
type Test struct {
	Title      string              // The title of the test
	Method     string              // Method: GET or POST (in future also POST:mp for multipart posts)
	Url        string              // full URL
	Header     map[string]string   // key/value pairs for request header
	Jar        *CookieJar          // cookies to send
	RespCond   []Condition         // list of conditions the response header must fullfill
	CookieCond []Condition         // conditions for recieved cookies
	BodyCond   []Condition         // conditions for the body (text or binary)
	Tag        []TagCondition      // list of tags to look for in the body
	Log        []LogCondition      // list of conditions to test on "log" files
	Pre        []string            // currently unused: list of test which are prerequisites to this test
	Param      map[string][]string // request parameter
	Setting    map[string]int      // setting like repetition, sleep time, etc. for this test
	Const      map[string]string   // const variables
	Rand       map[string][]string // random varibales
	Seq        map[string][]string // sequence variables
	SeqCnt     map[string]int      // internal stuff for sequnece variables 
	Vars       map[string]string   // internal stuff for variables
	Result     []Result            // list of pass/fails reports
	Body       []byte              // body of last non-failing response
	Dump       io.Writer           // a writer to dump requests and responses to
	Before     [][]string          // list of commands to execute before test
	After      [][]string          // list of commands to execute afterwards
}

type TestStatus int

const (
	TestPassed  TestStatus = iota
	TestFailed  TestStatus = iota
	TestErrored TestStatus = iota
)

func (status TestStatus) String() string {
	switch status {
	case TestPassed:
		return "Passed"
	case TestFailed:
		return "Failed"
	case TestErrored:
		return "Error"
	}
	panic(fmt.Sprintf("No such TestStatus %d", int(status)))
}

// Result encapsulates information about a performed check in a test.
type Result struct {
	Id     string // id of test/check e.g. "Line 64: Tag a href=" or "Line 12: Txt _= Hello" 
	Status TestStatus

	// Short reason
	// For failures: "missing", "forbidden", "wrong value", "wrong count" 
	// For errors: "bad test", "cannot connect", "cannot parse"
	Cause string

	// Long message with details
	Message string // Full error/failure message
}

func (result Result) String() string {
	return result.Id + ": " + result.Status.String() + " " + result.Cause + " " + result.Message
}

// Make a deep copy of src. dest will not share "any" data structures with src.
// Except Dump,Before and After
func (src *Test) Copy() (dest *Test) {
	dest = new(Test)
	dest.Title = src.Title
	dest.Method = src.Method
	dest.Url = src.Url
	dest.Header = copyMap(src.Header)
	dest.Jar = src.Jar.Copy()
	dest.RespCond = make([]Condition, len(src.RespCond))
	copy(dest.RespCond, src.RespCond)
	dest.CookieCond = make([]Condition, len(src.CookieCond))
	copy(dest.CookieCond, src.CookieCond)
	dest.BodyCond = make([]Condition, len(src.BodyCond))
	copy(dest.BodyCond, src.BodyCond)
	dest.Tag = make([]TagCondition, len(src.Tag))
	copy(dest.Tag, src.Tag)
	for i, tc := range dest.Tag {
		dest.Tag[i].Spec = *((&tc.Spec).DeepCopy())
	}
	dest.Pre = make([]string, len(src.Pre))
	copy(dest.Pre, src.Pre)
	dest.Param = copyMultiMap(src.Param)
	dest.Setting = make(map[string]int, len(src.Setting))
	for k, v := range src.Setting {
		dest.Setting[k] = v
	}
	dest.Const = copyMap(src.Const)
	dest.Rand = copyMultiMap(src.Rand)
	dest.Seq = copyMultiMap(src.Seq)
	dest.SeqCnt = make(map[string]int, len(src.SeqCnt))
	for k, v := range src.SeqCnt {
		dest.SeqCnt[k] = v
	}
	dest.Vars = copyMap(src.Vars)
	dest.Result = make([]Result, len(src.Result))
	copy(dest.Result, src.Result)

	dest.Dump = src.Dump
	dest.Before = src.Before
	dest.After = src.After
	dest.Log = src.Log

	return
}

func copyMultiMap(src map[string][]string) (dest map[string][]string) {
	dest = make(map[string][]string, len(src))
	for k, vl := range src {
		nl := make([]string, len(vl))
		copy(nl, vl)
		dest[k] = nl
	}
	return
}

func copyMap(src map[string]string) (dest map[string]string) {
	dest = make(map[string]string, len(src))
	for k, v := range src {
		dest[k] = v
	}
	return
}

func (t *Test) Failed(id, cause, text string) {
	t.Result = append(t.Result, Result{Id: id, Status: TestFailed, Cause: cause, Message: text})
}
func (t *Test) Passed(text string) {
	t.Result = append(t.Result, Result{Id: "", Status: TestPassed, Cause: "", Message: text})
}
func (t *Test) Error(id, cause, text string) {
	t.Result = append(t.Result, Result{Id: id, Status: TestErrored, Cause: cause, Message: text})
}
/****
func (t *Test) Info(text string)   { 
	t.Result = append(t.Result, Result{Id: "", Status: TestFailed, Cause: "", Message: text}) 
	t.Result = append(t.Result, "       "+text) }
*****/

// Return number of executed (total), passed and failed tests. 
func (t *Test) Stat() (passed, failed, errors int) {
	for _, result := range t.Result {
		switch result.Status {
		case TestPassed:
			passed++
		case TestFailed:
			failed++
		case TestErrored:
			errors++
		default:
			panic(fmt.Sprintf("No such TestStatus %d", int(result.Status)))
		}
	}
	return
}

// Texttual representation of t.Stat()
func (t *Test) Status() (status string) {
	p, f, e := t.Stat()
	if p+f+e == 0 {
		status = "not jet run"
	} else {
		if e > 0 {
			status = "ERROR "
		} else if f > 0 {
			status = "FAILED"
		} else {
			status = "PASSED"
		}
		status += fmt.Sprintf(" (passed: %-2d,  failed: %-2d,  error: %-2d)", p, f, e)
	}
	return
}

// NewTest sets up a new test, empty test.
func NewTest(title string) *Test {
	t := Test{Title: title}

	t.Header = make(map[string]string)
	t.Jar = NewCookieJar()
	t.Param = make(map[string][]string)
	t.Setting = make(map[string]int, len(DefaultSettings))
	t.Const = make(map[string]string)
	t.Rand = make(map[string][]string)
	t.Seq = make(map[string][]string)
	t.SeqCnt = make(map[string]int)
	t.Vars = make(map[string]string)

	for k, v := range DefaultSettings {
		t.Setting[k] = v
	}

	return &t
}

// Helper to read a setting of a test.
func (t *Test) getSetting(name string) int {
	if t.Setting != nil {
		if n, ok := t.Setting[name]; ok {
			return n
		}
	}
	return DefaultSettings[name]
}

// Let compiler find misspellings...
func (t *Test) Repeat() int      { return t.getSetting("Repeat") }
func (t *Test) Sleep() int       { return t.getSetting("Sleep") }
func (t *Test) Tries() int       { return t.getSetting("Tries") }
func (t *Test) KeepCookies() int { return t.getSetting("Keep-Cookies") }
func (t *Test) Abort() int       { return t.getSetting("Abort") }
func (t *Test) Validate() int    { return t.getSetting("Validate") }
func (t *Test) DoDump() int      { return t.getSetting("Dump") }
func (t *Test) MaxTime() int     { return t.getSetting("Max-Time") }

// Look for name in cookies. Return index if found and -1 otherwise.
// Looup happens from behind as last setting wins in browser.
func cookieIndex(cookies []*http.Cookie, name, domain, path string) int {
	trace("Looking for recieved cookie %s:%s:%s", name, domain, path)
	for i := len(cookies) - 1; i >= 0; i-- {
		c := cookies[i]
		trace("  compare with %s:%s:%s", c.Name, c.Domain, c.Path)
		if c.Name == name && strings.HasSuffix(domain, c.Domain) && strings.HasPrefix(path, c.Path) {
			trace("    --> found")
			return i
		}
	}
	return -1
}

// Test if a cookie deletion is reliable:
// Reliable deleting a cookie requires all of
//   Max-Age: 0
//   Expires is set and before NOW()
//   Value is empty
func testCookieDeletion(orig *Test, c *http.Cookie, neg bool, ci string) {
	trace("Test for deletion of cookie '%s' (neg=%t)", c.Name, neg)
	if neg {
		orig.Error(ci, "bad test", "You cannot test on 'not deletion' of cookie.")
		return
	}

	// Reliable deleted == Max-Age: 0 AND Expired in the past
	if c.MaxAge < 0 && c.Expires.Year != 0 && c.Expires.Seconds() < time.UTC().Seconds() && c.Value == "" {
		trace("  Properly deleted")
		orig.Passed(ci)
	} else {
		cause := ""
		if c.MaxAge >= 0 {
			cause += "Missing 'Max-Age: 0'."
		}
		if c.Value != "" {
			cause += "Value '" + c.Value + "' given."
		}
		if c.Expires.Year == 0 {
			cause += " Expires not set."
		} else if c.Expires.Seconds() >= time.UTC().Seconds() {
			cause += fmt.Sprintf(" Wrong Expires '%s'.",
				c.Expires.Format(http.TimeFormat))
		}
		trace("  Not properly deleted %s", cause)
		orig.Failed(ci, "not deleted", cause)
	}
}

// Test response header and (set-)cookies.
func testHeader(resp *http.Response, cookies []*http.Cookie, t, orig *Test) {
	if len(t.RespCond) > 0 {
		debug("Testing Header")
		for _, c := range t.RespCond {
			cs := c.Info("resp")
			v := resp.Header.Get(c.Key)
			if !c.Fullfilled(v) {
				orig.Failed(c.Id, "", fmt.Sprintf("%s: Got '%s'", cs, v))
			} else {
				orig.Passed(cs)
			}
		}
	}

	if len(t.CookieCond) > 0 {
		debug("Testing Cookies")
		domain := stripPort(resp.Request.URL.Host)
		for _, cc := range t.CookieCond {
			cc.Key = strings.Replace(cc.Key, "{CURRENT}", domain, 1)
			testSingleCookie(orig, cc, cookies)
		}
	}
}

func testSingleCookie(orig *Test, cc Condition, cookies []*http.Cookie) {
	ci := cc.Info("cookie")
	a := strings.Split(cc.Key, ":")
	name, domain, path, field := a[0], a[1], a[2], a[3]
	idx := cookieIndex(cookies, name, domain, path)
	if cc.Op == "." {
		panic("'Mere cookie existence' not implemented jet....")
	} else {
		if idx == -1 {
			orig.Failed(cc.Id, "missing", fmt.Sprintf("%s: Cookie was not set at all.", ci))
			// TODO: Report all cookies
			return
		}
		rc := cookies[idx]
		var v string
		switch field {
		case "value":
			v = rc.Value
		case "path":
			v = rc.Path
		case "domain":
			v = rc.Domain
		case "expires":
			v = rc.Expires.Format(http.TimeFormat)
		case "secure":
			v = fmt.Sprintf("%t", rc.Secure)
		case "httponly":
			v = fmt.Sprintf("%t", rc.HttpOnly)
		case "maxage":
			v = fmt.Sprintf("%d", rc.MaxAge)
		case "delete":
			testCookieDeletion(orig, rc, cc.Neg, ci)
			return
		default:
			orig.Error(cc.Id, "bad test", ": Oooops: Unknown cookie field "+field)
			return
		}
		if !cc.Fullfilled(v) {
			orig.Failed(cc.Id, "wrong", fmt.Sprintf("%s: Got '%s'", ci, v))
		} else {
			orig.Passed(ci)
		}
	}
}

// Test response body.
func testBody(body []byte, t, orig *Test) {
	if len(t.BodyCond) > 0 {
		debug("Testing Body")
	} else {
		return
	}

	for _, c := range t.BodyCond {
		cs := c.Info("body")
		switch c.Key {
		case "Txt":
			trace("Text Matching '%s'", c.String())
			if !c.Fullfilled(string(body)) {
				orig.Failed(c.Id, "Txt Failed", cs)
			} else {
				orig.Passed(cs)
			}
		case "Bin":
			if !c.BinFullfilled(body) {
				orig.Failed(c.Id, "Bin Failed", cs)
			} else {
				orig.Passed(cs)
			}
		default:
			error("Unkown type of test '%s' (%s). Ignored.", c.Key, c.Id)
		}
	}
	return
}

// Perform tag test on response body.
func testTags(t, orig *Test, doc *tag.Node) {
	if len(t.Tag) > 0 {
		debug("Testing Tags")
	} else {
		return
	}

	if doc == nil {
		orig.Error("Tag", "Bad test", "No body to parse. doc == nil")
		return
	}

	for _, tc := range t.Tag {
		cs := tc.Info("tag")
		switch tc.Cond {
		case TagExpected, TagForbidden:
			n := tag.FindTag(&tc.Spec, doc)
			if tc.Cond == TagExpected {
				if n != nil {
					orig.Passed(cs)
				} else {
					orig.Failed(tc.Id, "Missing", fmt.Sprintf("%s: Missing", cs))
				}
			} else {
				if n == nil {
					orig.Passed(cs)
				} else {
					orig.Failed(tc.Id, "Forbidden", fmt.Sprintf("%s: Forbidden", cs))
				}
			}
		case CountEqual, CountNotEqual, CountLess, CountLessEqual, CountGreater, CountGreaterEqual:
			got, exp := tag.CountTag(&tc.Spec, doc), tc.Count
			switch tc.Cond {
			case CountEqual:
				if got != exp {
					orig.Failed(tc.Id, "Wrong count",
						fmt.Sprintf("%s: Found %d expected %d", cs, got, exp))
					continue
				}
			case CountNotEqual:
				if got == exp {
					orig.Failed(tc.Id, "Wrong count",
						fmt.Sprintf("%s: Found %d expected != %d", cs, got, exp))
					continue
				}
			case CountLess:
				if got >= exp {
					orig.Failed(tc.Id, "Wrong count",
						fmt.Sprintf("%s: Found %d expected < %d", cs, got, exp))
					continue
				}
			case CountLessEqual:
				if got > exp {
					orig.Failed(tc.Id, "Wrong count",
						fmt.Sprintf("%s: Found %d expected <= %d", cs, got, exp))
					continue
				}
			case CountGreater:
				if got <= exp {
					orig.Failed(tc.Id, "Wrong count",
						fmt.Sprintf("%s: Found %d expected > %d", cs, got, exp))
					continue
				}
			case CountGreaterEqual:
				if got < exp {
					orig.Failed(tc.Id, "Wrong count",
						fmt.Sprintf("%s: Found %d expected >= %d", cs, got, exp))
					continue
				}
			}
			orig.Passed(cs)
		default:
			error("Unkown type of test %d (%s). Ignored.", tc.Cond, tc.Id)
		}
	}
}

// List of allready checked URLs in this run
var ValidUrls = map[string]bool{}

// If url is considered checkable (and is parsable) an http.URL is returned; else nil.
func shallCheckUrl(url_ string, base *url.URL) *url.URL {
	if strings.HasPrefix(url_, "#") || strings.HasPrefix(strings.ToLower(url_), "mailto:") {
		trace("Will not check plain page anchors or mailto links in %s", url_)
		return nil
	}
	if j := strings.Index(url_, "#"); j != -1 {
		url_ = url_[:j] // Strip #fragment like browsers do
	}
	pu, err := url.Parse(url_)
	if err != nil {
		error("Cannot parse url " + url_)
		return nil
	}
	if !pu.IsAbs() {
		u, e := base.Parse(url_)
		if e != nil {
			error("Cannot parse %s relative to %s.", url_, base.String())
			return nil
		}
		return u
	}
	if pu.Host == base.Host {
		return pu
	}
	return nil
}

// Check if html in body is valid. If doc not nil check links too.
func testLinkValidation(t, orig, global *Test, doc *tag.Node, resp *http.Response, base string) {
	if doc == nil {
		warn("Cannot check links on nil document.")
		return
	}
	trace("Validating links")

	baseUrl, _ := url.Parse(base)     // Should not fail!
	urls := make(map[string]bool, 50) // keys are urls to prevent doubles

	for _, pat := range []string{"a href", "link href", "img src"} {
		ts := tag.MustParseTagSpec(pat)
		for _, tg := range tag.FindAllTags(ts, doc) {
			for _, a := range tg.Attr {
				if (a.Key == "href" || a.Key == "src") && a.Val != "" {
					if url_ := shallCheckUrl(a.Val, baseUrl); url_ != nil {
						urls[url_.String()] = true
					}
				}
			}
		}
	}
	tmpl := t.Copy()
	tmpl.Method = "GET"
	tmpl.Tag = nil
	tmpl.BodyCond = nil
	tmpl.CookieCond = nil
	tmpl.Param = nil
	// tmpl.Dump = nil
	tmpl.Setting = DefaultSettings
	tmpl.RespCond = []Condition{Condition{Key: "Status-Code", Op: "==", Val: "200"}}

	pass := true
	failures := "Bad Links:"
	for url_, _ := range urls {
		if _, ok := ValidUrls[url_]; ok {
			warn("Will not retest " + url_)
		}
		test := tmpl.Copy()
		test.Url = url_
		_, _, err := test.RunSingle(global, false) // TODO: No global! Or?
		if err != nil {
			pass = false
			failures += "\n" + fmt.Sprintf("Cannot access `%s': %s", test.Url, err.String())
			continue
		}
		if _, failed, _ := test.Stat(); failed > 0 {
			s := "Failures for " + test.Url + ": "
			for _, r := range test.Result {
				if r.Status != TestPassed {
					s += "\n    " + r.Cause + ". " + r.Message
				}
			}
			failures += "\n" + s
			pass = false
		} else {
			orig.Passed("Link " + url_)
			ValidUrls[url_] = true
		}
	}

	if !pass {
		orig.Failed("Link Validation", "Invalid Links.", failures)
	} else {
		orig.Passed("Link Validation")
	}
}

// Check if html is valid html
func testHtmlValidation(t, orig, global *Test, body string) {
	checkId := "HTML Validation"
	trace("Validating HTML")
	f, err := ioutil.TempFile("", "htmlvalid")
	if err != nil {
		orig.Error(checkId, "Cannot open temp file", err.String())
		return
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	name += ".html"
	f, err = os.Create(name)
	f.Write([]byte(body))
	f.Close()
	defer func() { os.Remove(name) }()

	test := NewTest("W3C validator")
	test.Method = "POST"
	test.Url = "http://validator.w3.org/check"
	test.Tag = nil
	test.BodyCond = nil
	test.CookieCond = nil
	test.Param = map[string][]string{"charset": []string{"(detect automatically)"},
		"doctype":       []string{"Inline"},
		"group":         []string{"0"},
		"uploaded_file": []string{"@file:" + name},
	}
	test.Dump = t.Dump
	test.Setting = DefaultSettings
	test.RespCond = []Condition{Condition{Key: "X-W3C-Validator-Status", Op: "==",
		Val: "Valid", Id: "html-validation"}}
	_, valbody, verr := test.RunSingle(global, false) // TODO: global?
	if verr != nil {
		orig.Error(checkId, "Cannot access W3C validator", verr.String())
		return
	}
	if _, failed, _ := test.Stat(); failed > 0 {
		failures := "Invalid HTML:"
		orig.Failed(checkId, "Invalid HTML", "")
		doc, err := tag.ParseHtml(string(valbody))
		if err != nil {
			warn("Cannot parse response from W3C validator: " + err.String())
			// orig.Error(checkId, "Cannot parse response from W3C validator", err.String())
		} else {
			fts := tag.MustParseTagSpec("li class=msg_err\n  em\n  span class=msg")
			for _, en := range tag.FindAllTags(fts, doc) {
				failures += "\n" + en.Full
			}
		}
		orig.Failed(checkId, "Invalid HTML", failures)
	} else {
		orig.Passed("html is valid")
	}
}

// Add header conditions from global to test.
func addMissingHeader(test, global *map[string]string) {
	for k, v := range *global {
		if _, ok := (*test)[k]; !ok {
			(*test)[k] = v
			trace("Adding missing header %s: %s", k, v)
		}
	}
}

// Add cookie conditions from global to test.
func addMissingCookies(test, global *CookieJar, u *url.URL) {
	for _, cookie := range global.Select(u) {
		if test.Contains(cookie.Domain, cookie.Path, cookie.Name) == nil {
			test.Update(*cookie, cookie.Domain)
		}
	}
}

// Add missing response conditions from global.
func addMissingCond(test, global []Condition) []Condition {
	a := len(test)
	for _, cond := range global {
		found := false
		for i := 0; i < a; i++ {
			if cond.Key == test[i].Key {
				found = true
				break
			}
		}
		if found {
			continue
		} // do not overwrite
		test = append(test, cond)
		trace("Adding response condition '%s'", cond.String())
	}
	return test
}

// Add all body conditions from global to test.
func addAllCond(test, global []Condition) []Condition {
	for _, cond := range global {
		trace("Adding body condition '%s'", cond.String())
		test = append(test, cond)
	}
	return test
}

// Prepare the test: Add new stuff from global
func prepareTest(t, global *Test) *Test {
	debug("Preparing test '%s' (global %t).", t.Title, (global != nil))

	// Clear map of variable values: new run, new values (overkill for consts)
	for k, _ := range t.Vars {
		t.Vars[k] = "", false
	}

	// deep copy, add stuff from global
	test := t.Copy()
	if global != nil {
		addMissingHeader(&test.Header, &global.Header)
		u, _ := url.Parse(test.Url) // TODO: make sure during pasring that url is parsabel :)
		addMissingCookies(test.Jar, global.Jar, u)
		test.RespCond = addMissingCond(test.RespCond, global.RespCond)
		test.BodyCond = addAllCond(test.BodyCond, global.BodyCond)
	}

	substituteVariables(test, global, t)
	if uc, ok := test.Header["Basic-Authorization"]; ok {
		// replace Basic-Authorization: user:pass with Authorization: Basic=encoded
		enc := base64.URLEncoding
		encoded := make([]byte, enc.EncodedLen(len(uc)))
		enc.Encode(encoded, []byte(uc))
		test.Header["Authorization"] = "Basic " + string(encoded)
		test.Header["Basic-Authorization"] = "", false
	}

	// Domain in cookie defaults to possible changable host of request...
	if u, eu := url.Parse(test.Url); eu == nil {
		host := stripPort(u.Host)
		for i := range test.Jar.cookies {
			test.Jar.cookies[i].Domain = strings.Replace(test.Jar.cookies[i].Domain, "{CURRENT}",
				host, 1)
		}
	}

	test.Dump = t.Dump
	supertrace("Test to execute = \n%s", test.String())
	return test
}

// Pattern (with shell/path globbing) of content types considered parsable by tag package.
var ParsableContentTypes []string = []string{"text/html", "text/html;*",
	"application/xml", "application/xhtml+xml", "application/xml; *", "application/xhtml+xml;*",
	"text/xml", "text/xml;*", "application/*+xml*", "application/xml-*"}

// Return true if body is considered parsabel (=checkable with tag)
func parsableBody(resp *http.Response) bool {
	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	for _, pat := range ParsableContentTypes {
		if m, _ := path.Match(pat, ct); m {
			return true
		}
	}
	info("Response body is not considered parsable")
	return false
}

// Set up bookkeeping stuff for variable substitutions.
func (test *Test) init() {
	// Initialize sequenze count
	if test.SeqCnt == nil {
		test.SeqCnt = make(map[string]int, len(test.Seq))
		for k, _ := range test.Seq {
			test.SeqCnt[k] = 0
		}
	}

	// Initialize storage for current value of vars
	if test.Vars == nil {
		cnt := len(test.Seq) + len(test.Rand) + len(test.Const)
		test.Vars = make(map[string]string, cnt)
	}
}

// Run a test. Number of repetitions (or no run at all) is taken from "Repeat"
// field in Param. If global is non nil it will be used as "template" for the
// test. The test.Result field is updated.
func (test *Test) Run(global *Test) {

	if test.Repeat() == 0 {
		info("Test '%s' is disabled.", test.Title)
		return
	}

	test.init()

	// Debuging dump
	if dd := test.DoDump(); dd == 1 || dd == 2 {
		fname := titleToFilename(test.Title) + ".dump"
		var mode int = os.O_TRUNC
		if dd == 2 {
			mode = os.O_APPEND
		}
		file, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|mode, 0666)
		if err != nil {
			error("Cannot dump to file '%s': %s.", fname, err.String())
		} else {
			defer file.Close()
			test.Dump = file
		}
	}

	reps := test.Repeat()
	for i := 1; i <= reps; i++ {
		info("Test '%s': Round %d of %d.", test.Title, i, reps)
		test.RunSingle(global, false)
	}

	info("Test '%s': %s", test.Title, test.Status())
	return
}

// Execute test, but do not test conditions. Usefull as background task in loadtesting.
func (test *Test) RunWithoutTest(global *Test) {
	if test.Repeat() == 0 {
		info("Test no '%s' is disabled.", test.Title)
		return
	}

	test.init()
	test.Setting["Tries"] = 0 // no test -> no need to try to succeed
	for i := 1; i <= test.Repeat(); i++ {
		test.RunSingle(global, true)
	}
	return
}

// Benchmark test.
func (test *Test) Bench(global *Test, count int) (durations []int, failures int, err os.Error) {
	test.init()
	test.Dump = nil              // prevent dumping during benchmarking
	test.Setting["Validate"] = 0 // no use in benchmarking

	if count < 5 {
		warn("Cannot benchmark with less than 5 rounds. Will use 5.")
		count = 5
	}

	durations = make([]int, count)
	total, okay := 0, 0

	for okay < count {
		if float32(total) > BenchTolerance*float32(count) {
			info("Too many errors for %d: %f > %f", count, float32(total), BenchTolerance*float32(count))
			err = Error("Too many failures/errors during benching")
			return
		}
		info("Bench '%s':", test.Title)
		dur, _, e := test.RunSingle(global, false)
		total++
		if e != nil {
			warn("Failure during bench")
		} else {
			durations[okay] = dur
			okay++
		}
	}

	failures = total - okay

	return
}

// Retrive file extension from content type or url.
func determineExt(url_, ct string) string {
	ct = strings.ToLower(ct)
	if strings.Contains(ct, "text/html") || strings.Contains(ct, "text/xhtml") || strings.Contains(ct, "application/xhtml+xml") {
		return "html"
	}
	if strings.Contains(ct, "xml") {
		return "xml"
	}
	if strings.Contains(ct, "application/pdf") {
		return "pdf"
	}
	u, err := url.Parse(url_)
	if err == nil {
		p := u.Path
		if i := strings.LastIndex(p, "."); i != -1 {
			return p[i+1:]
		}
	}
	return "bin"
}

// Execute shell command
func executeShellCmd(cmdline []string) (e int, s string) {
	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	if err := cmd.Start(); err != nil {
		e = -9999
		s = fmt.Sprintf("Cannot start %s: %s", cmdline[0], err.String())
		return
	}
	err := cmd.Wait()
	if err == nil {
		e, s = 0, ""
		return
	}
	if wm, ok := err.(*os.Waitmsg); ok {
		e = wm.ExitStatus()
		s = wm.String()
		return
	}

	e = -9998
	s = err.String()
	return
}

// Perform the checks in log on file.
func checkLog(test *Test, file *os.File, log LogCondition, origsize int64) {
	switch log.Op {
	case "~=", "/=", "_=", "=_":
		os, err := file.Seek(origsize, 0)
		if err != nil || os != origsize {
			test.Error(log.Id, "Cannot seek in "+log.Path, err.String())
			return
		}
		buf, err := ioutil.ReadAll(file)
		if err != nil {
			test.Error(log.Id, "Cannot read from "+log.Path, err.String())
			return
		}
		checkLogContent(test, buf, log)
	case "<", "<=", ">", ">=":
		fi, err := file.Stat()
		if err != nil {
			test.Error(log.Id, "Cannot get Fileinfo", err.String())
			return
		}
		delta := fi.Size - origsize
		if delta < 0 {
			test.Error(log.Id, "Logfile shrinked", "Maybe log rotated?")
			return
		}
		expected, err := strconv.Atoi64(log.Val)
		if err != nil {
			test.Error(log.Id, "Bad test.",
				fmt.Sprintf("Cannot convert '%s' to int: %s", log.Val, err.String()))
			return
		}
		checkLogSize(test, log, delta, expected)
	default:
		panic("No such LogCondition op: " + log.Op)
	}
}

// check size constraints on log file.
func checkLogSize(test *Test, log LogCondition, delta, expected int64) {
	if log.Op == "<=" {
		delta--
	}
	if log.Op == ">=" {
		delta++
	}
	switch log.Op[0] {
	case '<':
		if delta >= expected {
			test.Failed(log.Id, "Too much logfile growth",
				fmt.Sprintf("Logfile grew by %d bytes.", delta))
			return
		}
	case '>':
		if delta <= expected {
			test.Failed(log.Id, "Not enough logfile growth",
				fmt.Sprintf("Logfile grew by %d bytes.", delta))
			return
		}
	}

}

// check content added to log file.
func checkLogContent(test *Test, buf []byte, log LogCondition) {
	txt := string(buf)
	trace("Checking %s on: %s", log.String(), txt)
	found := false
	switch log.Op {
	case "~=":
		found = strings.Index(txt, log.Val) != -1
	case "/=":
		re, err := regexp.Compile(log.Val)
		if err != nil {
			test.Error(log.Id, "Bad Test", "Unparsable regexp: "+err.String())
			return
		}
		found = re.Find(buf) != nil
	case "_=", "=_":
		txt = strings.Replace(txt, "\r\n", "\n", -1)
		var cf func(string) bool
		if log.Op == "_=" {
			cf = func(s string) bool { return hp(s, log.Val) }
		} else {
			cf = func(s string) bool { return hs(s, log.Val) }
		}
		for _, s := range strings.Split(txt, "\n") {
			if cf(s) {
				found = true
				break
			}
		}
	default:
		panic("No such operator '" + log.Op + "' for logfiles")
	}

	if !found && !log.Neg {
		trace("Not found")
		test.Failed(log.Id, "Missing", fmt.Sprintf("Missing %s in\n%s", log.Val, txt))
		return
	} else if found && log.Neg {
		trace("Found")
		test.Failed(log.Id, "Forbidden", fmt.Sprintf("Forbidden %s in\n%s", log.Val, txt))
		return
	} else {
		test.Passed("Log okay: " + log.String())
	}
}

// Perform a single run of the test.  Return duration for server response in ms,
// recieved body or error.  If request itself failed, then err is non nil and contains the reason.
// Logs the results of the tests in Result field.
func (test *Test) RunSingle(global *Test, skipTests bool) (duration int, body []byte, err os.Error) {
	ti := prepareTest(test, global)

	// Before Commands and log file initialisation
	var logfilesize map[string]int64 // sizes of log files in byte; same order as test.Log
	if !skipTests {
		for _, cmd := range ti.Before {
			if rv, msg := executeShellCmd(cmd); rv != 0 {
				test.Error(fmt.Sprintf("Before cmd %s", cmd),
					fmt.Sprintf("Failure %d", rv),
					msg)
				duration = 0
				err = os.NewError("Failed BEFORE command")
				return
			}
		}
		trace("Number of logfile tests: %d", len(ti.Log))
		logfilesize = determinLogfileSize(ti.Log, test)
	}

	tries := ti.Tries()
	var tryCnt int
	for {
		starttime := time.Nanoseconds()
		var (
			response *http.Response
			url_     string
			cookies  []*http.Cookie
			reqerr   os.Error
		)

		if ti.Method == "GET" {
			response, url_, cookies, reqerr = Get(ti)
		} else if ti.Method == "POST" || ti.Method == "POST:mp" {
			response, url_, cookies, reqerr = Post(ti)
		}
		endtime := time.Nanoseconds()
		duration = int((endtime - starttime) / 1000000) // in milliseconds (ms)

		if reqerr != nil {
			test.Error("Request", "Failed Request", reqerr.String())
			err = Error("Error: " + reqerr.String())
		} else {
			body = performChecks(test, ti, global, response, cookies, url_, duration, skipTests)
		}

		if test.Sleep() > 0 {
			trace("Sleeping for %d seconds.", test.Sleep())
			time.Sleep(1000000 * int64(test.Sleep()))
		}

		tryCnt++
		_, failed, _ := test.Stat()
		// fmt.Printf(">>> %s tryCnt: %d,  tries: %d, failed: %d  [%s]\n", test.Title, tryCnt, tries, failed, test.Status()) 
		if tryCnt >= tries || failed == 0 {
			break
		}
		// clear Result and start over
		test.Result = test.Result[0:0]
		// fmt.Printf("\n-----\n%s: %s\n=========\n", test.Title, test.Status()) 

	}

	// After Commands and Logfile
	if !skipTests {
		for _, cmd := range ti.After {
			if rv, msg := executeShellCmd(cmd); rv != 0 {
				test.Error(fmt.Sprintf("After cmd %s", cmd),
					fmt.Sprintf("Failure %d", rv),
					msg)
				return
			}
		}

		for _, log := range ti.Log {
			if logfilesize[log.Path] == -1 {
				continue
			}
			file, err := os.Open(log.Path)
			if err != nil {
				test.Error(log.Id, "Cannot open "+log.Path, err.String())
				continue
			}
			checkLog(test, file, log, logfilesize[log.Path])
			file.Close()
		}

	}

	return
}

// Main function to perform test on the response: cookies, header, body, tags, and timing. 
func performChecks(test, ti, global *Test, response *http.Response, cookies []*http.Cookie,
url_ string, duration int, skipTests bool) (body []byte) {

	body = readBody(response.Body)

	trace("Recieved cookies: %v", cookies)
	if len(cookies) > 0 && test.KeepCookies() == 1 && global != nil {
		if global.Jar == nil {
			global.Jar = NewCookieJar()
		}

		u, _ := url.Parse(url_)
		for _, c := range cookies {
			trace("Updating cookie %s in global jar.", c.Name)
			global.Jar.Update(*c, u.Host)
		}
	}

	if skipTests {
		return
	}

	// Response: Add special fields to header befor testing
	response.Header.Set("Status-Code", fmt.Sprintf("%d", response.StatusCode))
	response.Header.Set("Final-Url", url_)
	testHeader(response, cookies, ti, test)

	// Body:
	if ti.DoDump() == 3 {
		dumpBody(body, ti.Title, url_, response.Header.Get("Content-Type"))
	}
	testBody(body, ti, test)

	// Tag:
	if len(ti.Tag) > 0 || ti.Validate()&1 != 0 {
		var doc *tag.Node
		if parsableBody(response) {
			var e os.Error
			doc, e = tag.ParseHtml(string(body))
			if e != nil {
				test.Error("Tag/Validation",
					"HTML unparsable", e.String())
				error("Problems parsing html: " + e.String())
			}
		} else {
			error("Unparsable body ")
			test.Error("Tag/Validation",
				"Body considered unparsable.", "")
		}

		testTags(ti, test, doc)
		if ti.Validate()&1 != 0 {
			testLinkValidation(ti, test, global, doc, response, url_)
		}
		if ti.Validate()&2 != 0 {
			testHtmlValidation(ti, test, global, string(body))
		}
	}

	// Timing:
	if max := ti.MaxTime(); max > 0 {
		if duration > max {
			test.Failed("Response Time", "Exeeded limit",
				fmt.Sprintf("Response exeeded Max-Time of %d (was %d).",
					max, duration))
		} else {
			test.Passed(fmt.Sprintf("Response took %d ms (allowed %d).",
				duration, max))
		}
	}
	return
}
