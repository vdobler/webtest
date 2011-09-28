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
	"strings"
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

type Test struct {
	Title      string              // The title of the test
	Method     string              // Method: GET or POST (in future also POST:mp for multipart posts)
	Url        string              // full URL
	Header     map[string]string   // key/value pairs for request header
	Cookie     map[string]string   // cookie-name/value pairs for cookies to send in request
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
	Result     []string            // list of pass/fails reports
	Body       []byte              // body of last non-failing response
	Dump       io.Writer           // a writer to dump requests and responses to
	Before     [][]string          // list of commands to execute before test
	After      [][]string          // list of commands to execute afterwards
}

// Make a deep copy of src. dest will not share "any" data structures with src.
// Except Dump,Before and After
func (src *Test) Copy() (dest *Test) {
	dest = new(Test)
	dest.Title = src.Title
	dest.Method = src.Method
	dest.Url = src.Url
	dest.Header = copyMap(src.Header)
	dest.Cookie = copyMap(src.Cookie)
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
	dest.Result = make([]string, len(src.Result))
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

func (t *Test) Failed(text string) { t.Result = append(t.Result, "Failed "+text) }
func (t *Test) Passed(text string) { t.Result = append(t.Result, "Passed "+text) }
func (t *Test) Error(text string)  { t.Result = append(t.Result, "Error  "+text) }
func (t *Test) Info(text string)   { t.Result = append(t.Result, "       "+text) }

// Return number of executed (total), passed and failed tests. 
func (t *Test) Stat() (passed, failed, errors, info int) {
	for _, r := range t.Result {
		if strings.HasPrefix(r, "Passed ") {
			passed++
		} else if strings.HasPrefix(r, "Failed ") {
			failed++
		} else if strings.HasPrefix(r, "Error ") {
			errors++
		} else if strings.HasPrefix(r, "      ") {
			info++
		} else {
			error("Oooops: Unknown status in %s.", r)
		}
	}
	return
}

// Texttual representation of t.Stat()
func (t *Test) Status() (status string) {
	p, f, e, i := t.Stat()
	if p+f+e+i == 0 {
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

func NewTest(title string) *Test {
	t := Test{Title: title}

	t.Header = make(map[string]string)
	t.Cookie = make(map[string]string)
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
func cookieIndex(cookies []*http.Cookie, name string) int {
	idx := -1
	for i, c := range cookies {
		if c.Name == name {
			idx = i
			break
		}
	}
	return idx
}

// Test response header and (set-)cookies.
func testHeader(resp *http.Response, cookies []*http.Cookie, t, orig *Test) {
	if len(t.RespCond) > 0 {
		debug("Testing Header")
		for _, c := range t.RespCond {
			cs := c.Info("resp")
			v := resp.Header.Get(c.Key)
			if !c.Fullfilled(v) {
				orig.Failed(fmt.Sprintf("%s: Got '%s'", cs, v))
			} else {
				orig.Passed(cs)
			}
		}
	}

	if len(t.CookieCond) > 0 {
		debug("Testing Cookies")
	} else {
		return
	}

	for _, cc := range t.CookieCond {
		ci := cc.Info("cookie")
		var name, field string
		if fi := strings.Index(cc.Key, ":"); fi != -1 {
			name = cc.Key[:fi]
			field = cc.Key[fi+1:]
		} else {
			name = cc.Key
		}
		idx := cookieIndex(cookies, name)

		if cc.Op == "." {
			// just test existence
			if field != "" {
				error("Ooooops: field but op==.")
			}
			if idx != -1 && cc.Neg {
				orig.Failed(fmt.Sprintf("%s: Cookie _was_ set.", ci))
			} else if (idx != -1 && !cc.Neg) || (idx == -1 && cc.Neg) {
				orig.Passed(ci)
			} else if idx == -1 && !cc.Neg {
				orig.Failed(fmt.Sprintf("%s: Cookie was not set.", ci))
			} else {
				error("Oooops: This cannot happen....")
			}
		} else {
			if idx == -1 {
				orig.Failed(fmt.Sprintf("%s: Cookie was not set at all.", ci))
				continue
			}
			rc := cookies[idx]
			var v string
			switch field {
			case "Value":
				v = rc.Value
			case "Path":
				v = rc.Path
			case "Domain":
				v = rc.Domain
			case "Expires":
				v = rc.Expires.Format(http.TimeFormat)
			case "Secure":
				v = fmt.Sprintf("%t", rc.Secure)
			case "HttpOnly":
				v = fmt.Sprintf("%t", rc.HttpOnly)
			case "MaxAge":
				v = fmt.Sprintf("%d", rc.MaxAge)
			default:
				error("Oooops: Unknown cookie field " + field)
			}
			if !cc.Fullfilled(v) {
				orig.Failed(fmt.Sprintf("%s: Got '%s'", ci, v))
			} else {
				orig.Passed(ci)
			}
		}
	}
	return
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
				orig.Failed(cs)
			} else {
				orig.Passed(cs)
			}
		case "Bin":
			if !c.BinFullfilled(body) {
				orig.Failed(cs)
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
		orig.Error("No body to parse.")
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
					orig.Failed(fmt.Sprintf("%s: Missing", cs))
				}
			} else {
				if n == nil {
					orig.Passed(cs)
				} else {
					orig.Failed(fmt.Sprintf("%s: Forbidden", cs))
				}
			}
		case CountEqual, CountNotEqual, CountLess, CountLessEqual, CountGreater, CountGreaterEqual:
			got, exp := tag.CountTag(&tc.Spec, doc), tc.Count
			switch tc.Cond {
			case CountEqual:
				if got != exp {
					orig.Failed(fmt.Sprintf("%s: Found %d expected %d", cs, got, exp))
					continue
				}
			case CountNotEqual:
				if got == exp {
					orig.Failed(fmt.Sprintf("%s: Found %d expected != %d", cs, got, exp))
					continue
				}
			case CountLess:
				if got >= exp {
					orig.Failed(fmt.Sprintf("%s: Found %d expected < %d", cs, got, exp))
					continue
				}
			case CountLessEqual:
				if got > exp {
					orig.Failed(fmt.Sprintf("%s: Found %d expected <= %d", cs, got, exp))
					continue
				}
			case CountGreater:
				if got <= exp {
					orig.Failed(fmt.Sprintf("%s: Found %d expected > %d", cs, got, exp))
					continue
				}
			case CountGreaterEqual:
				if got < exp {
					orig.Failed(fmt.Sprintf("%s: Found %d expected >= %d", cs, got, exp))
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
		ts, _ := tag.ParseTagSpec(pat) // Wont err
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
	for url_, _ := range urls {
		if _, ok := ValidUrls[url_]; ok {
			warn("Will not retest " + url_)
		}
		test := tmpl.Copy()
		test.Url = url_
		_, _, err := test.RunSingle(global, false)
		if err != nil {
			orig.Failed(fmt.Sprintf("Cannot access `%s': %s", test.Url, err.String()))
			continue
		}
		if _, failed, _, _ := test.Stat(); failed > 0 {
			s := "Failures for " + test.Url + ": "
			for _, r := range test.Result {
				if !strings.HasPrefix(r, "Passed") {
					s += r + "; "
				}
			}
			orig.Failed(s)
		} else {
			orig.Passed("Link " + url_)
			ValidUrls[url_] = true
		}
	}
}

// Check if html is valid html
func testHtmlValidation(t, orig, global *Test, body string) {
	trace("Validating HTML")
	f, err := ioutil.TempFile("", "htmlvalid")
	if err != nil {
		orig.Error("Cannot open temp file: " + err.String())
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
	test.RespCond = []Condition{Condition{Key: "X-W3C-Validator-Status", Op: "==", Val: "Valid", Id: "html-validation"}}
	_, valbody, verr := test.RunSingle(global, false)
	if verr != nil {
		warn("Cannot access W3C validator: %s", verr.String())
		return
	}
	if _, failed, _, _ := test.Stat(); failed > 0 {
		orig.Failed("html is INVALID.")
		doc, err := tag.ParseHtml(string(valbody))
		if err != nil {
			warn("Cannot parse response from W3C validator: " + err.String())
			return
		}
		for _, en := range tag.FindAllTags(tag.MustParseTagSpec("li class=msg_err\n  em\n  span class=msg"), doc) {
			orig.Info(en.Full)
		}

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
func addMissingCookies(test, global *map[string]string) {
	for k, v := range *global {
		if _, ok := (*test)[k]; !ok {
			(*test)[k] = v
			trace("Adding missing cookie %s =%s", k, v)
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
		addMissingCookies(&test.Cookie, &global.Cookie)
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

// Sanitize t (by replacing anything uncomfortable in a filename) by _.
// The default output path is prepended automatically.
func titleToFilename(t string) (f string) {
	f = OutputPath
	if !strings.HasSuffix(f, "/") {
		f += "/"
	}

	for _, cp := range t {
		switch true {
		case cp >= 'a' && cp <= 'z', cp >= 'A' && cp <= 'Z', cp >= '0' && cp <= '9',
			cp == '-', cp == '+', cp == '.', cp == ',', cp == '_':
			f += string(cp)
		case cp <= 32, cp >= 127:
			// eat
		default:
			f += "_"
		}
	}
	for strings.Contains(f, "__") {
		f = strings.Replace(f, "__", "_", -1)
	}
	f = strings.Replace(f, "--", "-", -1)
	return
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

// Write body to a new file (name pattern is <TestTitle>.<N>.<FileExtension>).
// N is increased up to 999 to find a "new" file.
func dumpBody(body []byte, title, url_, ct string) {
	name := titleToFilename(title)
	ext := determineExt(url_, ct)
	var fname string
	for i := 0; i < 1000; i++ {
		fname = fmt.Sprintf("%s.%03d.%s", name, i, ext)
		_, err := os.Stat(fname)
		if e, ok := err.(*os.PathError); ok && e.Error == os.ENOENT {
			break
		}
	}

	file, err := os.Create(fname)
	if err != nil {
		error("Cannot dump body to file '%s': %s.", fname, err.String())
	} else {
		defer file.Close()
		file.Write(body)
	}
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

// Return filesize of file path (or -1 on error).
func filesize(path string) int64 {
	file, err := os.Open(path)
	if err != nil {
		if err == os.ENOENT {
			return 0
		}
		return -1
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		return -1
	}
	return fi.Size
}

func checkLog(test *Test, buf []byte, log LogCondition) {
	txt := string(buf)
	trace("Checking %s on: %s", log.String(), txt)
	switch log.Op {
	case "~=":
		if strings.Index(txt, log.Val) == -1 {
			trace("Not found")
			if !log.Neg {
				test.Failed(fmt.Sprintf("Missing %s in log %s (%s)", log.Val, log.Path, log.Id))
				return
			}
		} else if log.Neg {
			trace("FOund")
			test.Failed(fmt.Sprintf("Forbidden %s in log %s (%s)", log.Val, log.Path, log.Id))
			return
		}
	case "/=":
		panic("Operator /= unimplemented for logfiles")
		return
	case "_=", "=_":
	default:
		panic("No such operator '" + log.Op + "' for logfiles")
	}
	test.Passed("Log okay: " + log.String())
}

// Perform a single run of the test.  Return duration for server response in ms.
// If request itself failed, then err is non nil and contains the reason.
// Logs the results of the tests in Result field.
func (test *Test) RunSingle(global *Test, skipTests bool) (duration int, body []byte, err os.Error) {
	ti := prepareTest(test, global)

	// Before Commands and log
	var logfilesize map[string]int64 // sizes of log files in byte; same order as test.Log
	if !skipTests {
		for _, cmd := range ti.Before {
			if rv, msg := executeShellCmd(cmd); rv != 0 {
				test.Error(fmt.Sprintf("Before cmd %d: %s: %s", rv, cmd, msg))
				duration = 0
				err = os.NewError("Failed before condition")
				return
			}
		}
		trace("Number of logfile tests: %d", len(ti.Log))
		if len(ti.Log) > 0 {
			logfilesize = make(map[string]int64)
			for _, log := range ti.Log {
				if _, ok := logfilesize[log.Path]; ok {
					continue
				}
				logfilesize[log.Path] = filesize(log.Path)
				trace("Filesize of %s = %d", log.Path, logfilesize[log.Path])
				if logfilesize[log.Path] == -1 {
					test.Error(fmt.Sprintf("Cannot check logfile %s", log.Path))
				}
			}
		}
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
			test.Error(reqerr.String())
			err = Error("Error: " + reqerr.String())
		} else {
			body = readBody(response.Body)

			trace("Recieved cookies: %v", cookies)
			if len(cookies) > 0 && test.KeepCookies() == 1 && global != nil {
				if global.Cookie == nil {
					global.Cookie = make(map[string]string)
				}

				// TODO: proper cookie handling of expiration/deletion
				for _, c := range cookies {
					if c.MaxAge == -999 { // Delete
						trace("Deleting cookie %s from global (delete req from server).",
							c.Name)
						global.Cookie[c.Name] = "", false
					} else {
						trace("Storing cookie %s in global.", c.Name)
						global.Cookie[c.Name] = c.Value
					}
				}
			}

			if !skipTests {
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
							test.Error("Problems parsing html: " + e.String())
							error("Problems parsing html: " + e.String())
						}
					} else {
						error("Unparsable body ")
						test.Error("Body considered unparsable.")
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
						test.Failed(fmt.Sprintf("Response exeeded Max-Time of %d (was %d).",
							max, duration))
					} else {
						test.Passed(fmt.Sprintf("Response took %d ms (allowed %d).",
							duration, max))
					}
				}

			}
		}

		if test.Sleep() > 0 {
			trace("Sleeping for %d seconds.", test.Sleep())
			time.Sleep(1000000 * int64(test.Sleep()))
		}

		tryCnt++
		_, failed, _, _ := test.Stat()
		// fmt.Printf(">>> tryCnt: %d,  tries: %d, failed: %d\n", tryCnt, tries, failed)
		// fmt.Printf("%s\n", test.Status()) 
		if tryCnt >= tries || failed == 0 {
			break
		}
		// clear Result and start over
		test.Result = make([]string, 0)
		// fmt.Printf("\n-----\n%s\n=========\n", test.Status()) 

	}

	// After Commands and Logfile
	if !skipTests {
		for _, cmd := range ti.After {
			if rv, msg := executeShellCmd(cmd); rv != 0 {
				test.Error(fmt.Sprintf("After cmd %d: %s: %s", rv, cmd, msg))
				return
			}
		}

		for _, log := range ti.Log {
			if logfilesize[log.Path] == -1 {
				continue
			}
			file, err := os.Open(log.Path)
			if err != nil {
				test.Error(fmt.Sprintf("Cannot open logfile %s", log.Path))
				continue
			}
			defer file.Close()
			os, err := file.Seek(logfilesize[log.Path], 0)
			if err != nil || os != logfilesize[log.Path] {
				test.Error(fmt.Sprintf("Cannot seek in logfile %s", log.Path))
				continue
			}
			buf, err := ioutil.ReadAll(file)
			if err != nil {
				test.Error(fmt.Sprintf("Cannot read logfile %s", log.Path))
				continue
			}
			checkLog(test, buf, log)
			file.Close()
		}

	}

	return
}
