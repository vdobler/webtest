package suite

import (
	"fmt"
	"http"
	"strings"
	"os"
	"dobler/webtest/tag"
	//	"encoding/hex"
	"encoding/base64"
	"time"
	"io"
	"io/ioutil"
	"path"
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
	Pre        []string            // currently unused: list of test which are prerequisites to this test
	Param      map[string][]string // request parameter
	Setting    map[string]int      // setting like repetition, sleep time, etc. for this test
	Const      map[string]string   // const variables
	Rand       map[string][]string // random varibales
	Seq        map[string][]string // sequence variables
	SeqCnt     map[string]int      // internal stuff for sequnece variables TODO: do not export
	Vars       map[string]string   // internal stuff for variables  TODO: do not export
	Result     []string            // list of pass/fails reports
	Body       []byte              // body of last non-failing response
	Dump       io.Writer           // a writer to dump requests and responses to
}

// Make a deep copy of src. dest will not share any data structures with src.
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


// Store pass or fail in t.
func (t *Test) Report(pass bool, text string) {
	var s string
	if pass {
		s = "Passed " + text
		info(s)
	} else {
		s = "FAILED " + text
		warn(s)
	}
	t.Result = append(t.Result, s)
}

// Return number of executed (total), passed and failed tests. 
func (t *Test) Stat() (total, passed, failed int) {
	for _, r := range t.Result {
		total++
		// trace("Result: %s", r)
		if strings.HasPrefix(r, "Passed") {
			passed++
		} else {
			failed++
		}
	}
	return
}

// Texttual representation of t.Stat()
func (t *Test) Status() (status string) {
	n, p, f := t.Stat()
	if n == 0 {
		status = "not jet run"
	} else {
		if f > 0 {
			status = "FAILED"
		} else {
			status = "PASSED"
		}
		status += fmt.Sprintf(" (total: %-2d,  passed: %-2d,  failed: %-2d)", n, p, f)
	}
	return
}

var DefaultSettings = map[string]int{"Repeat": 1,
	"Tries":        1,
	"Max-Time":     -1,
	"Sleep":        0,
	"Keep-Cookies": 0,
	"Abort":        0,
	"Dump":         0,
	"Validate":     0,
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

func needQuotes(s string, containedSpacesNeedQuotes bool) bool {
	if containedSpacesNeedQuotes && strings.Contains(s, " ") {
		return true
	}
	return strings.Contains(s, "\"") || strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") || strings.Contains(s, "\n") || strings.Contains(s, "\t")
}

func quote(s string, containedSpacesNeedQuotes bool) string {
	if !needQuotes(s, containedSpacesNeedQuotes) {
		return s
	}
	s = strings.Replace(s, "\"", "\\\"", -1)
	s = strings.Replace(s, "\n", "\\n", -1)
	s = strings.Replace(s, "\t", "\\t", -1)

	return "\"" + s + "\""
}


// Prety print a map m with title. 
func formatMap(title string, m *map[string]string) (f string) {
	if len(*m) > 0 {
		f = title + "\n"
		longest := 0
		for k, _ := range *m {
			if len(k) > longest {
				longest = len(k)
			}
		}
		for k, v := range *m {
			f += fmt.Sprintf("\t%-*s  %s\n", longest, k, quote(v, false))
		}
	}
	return
}

// Prety print a map m with title. 
func formatIntMap(title string, m *map[string]int) (f string) {
	if len(*m) > 0 {
		f = title + "\n"
		longest := 0
		for k, _ := range *m {
			if len(k) > longest {
				longest = len(k)
			}
		}
		for k, v := range *m {
			f += fmt.Sprintf("\t%-*s  %d\n", longest, k, v)
		}
	}
	return
}

// Pretty print a multi-map m.
func formatMultiMap(title string, m *map[string][]string) (f string) {
	if len(*m) > 0 {
		f = title + "\n"
		longest := 0
		for k, _ := range *m {
			if len(k) > longest {
				longest = len(k)
			}
		}
		for k, l := range *m {
			f += fmt.Sprintf("\t%-*s ", longest, k)
			for _, v := range l {
				f += " " + quote(v, true)
			}
			f += "\n"
		}
	}
	return
}


// Pretty print a list of Conditions m.
func formatCond(title string, m *[]Condition) (f string) {
	if len(*m) > 0 {
		f = title + "\n"
		longest := 0
		for _, c := range *m {
			if len(c.Key) > longest {
				longest = len(c.Key)
			}
		}
		for _, c := range *m {
			if c.Neg {
				f += "\t!"
			} else {
				f += "\t "
			}
			if c.Op != "." {
				f += fmt.Sprintf("%-*s  %2s  %s\n", longest, c.Key, c.Op, quote(c.Val, false))
			} else {
				f += c.Key + "\n"
			}
		}
	}
	return
}

// String representation as as used by the parser.
func (t *Test) String() (s string) {
	s = "-------------------------------\n" + t.Title + "\n-------------------------------\n"
	s += t.Method + " " + t.Url + "\n"
	s += formatMap("HEADER", &t.Header)
	s += formatMap("SEND-COOKIE", &t.Cookie)
	s += formatCond("RESPONSE", &t.RespCond)
	s += formatCond("SET-COOKIE", &t.CookieCond)
	s += formatCond("BODY", &t.BodyCond)
	s += formatMultiMap("PARAM", &t.Param)
	s += formatMap("CONST", &t.Const)
	s += formatMultiMap("SEQ", &t.Seq)
	s += formatMultiMap("RAND", &t.Rand)
	specSet := make(map[string]int) // map with non-standard settings
	for k, v := range t.Setting {
		if dflt, ok := DefaultSettings[k]; ok && v != dflt {
			specSet[k] = v
		}
	}
	s += formatIntMap("SETTING", &specSet)
	if len(t.Tag) > 0 {
		s += "TAG\n"
		for i, tagCond := range t.Tag {
			fts := tagCond.String()
			if i > 0 && strings.Contains(fts, "\n") {
				s += "\t\n"
			}
			s += "\t" + fts + "\n"
		}
	}

	return
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
func (t *Test) Repeat() int {
	return t.getSetting("Repeat")
}
func (t *Test) Sleep() int {
	return t.getSetting("Sleep")
}
func (t *Test) Tries() int {
	return t.getSetting("Tries")
}
func (t *Test) KeepCookies() int {
	return t.getSetting("Keep-Cookies")
}
func (t *Test) Abort() int {
	return t.getSetting("Abort")
}
func (t *Test) Validate() int {
	return t.getSetting("Validate")
}
func (t *Test) DoDump() int {
	return t.getSetting("Dump")
}
func (t *Test) MaxTime() int {
	return t.getSetting("Max-Time")
}


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
func testHeader(resp *http.Response, t, orig *Test) {
	if len(t.RespCond) > 0 {
		debug("Testing Header")
		for _, c := range t.RespCond {
			cs := c.Info("resp")
			v := resp.Header.Get(c.Key)
			if !c.Fullfilled(v) {
				orig.Report(false, fmt.Sprintf("%s: Got '%s'", cs, v))
			} else {
				orig.Report(true, cs)
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
		idx := cookieIndex(resp.SetCookie, name)

		if cc.Op == "." {
			// just test existence
			if field != "" {
				error("Ooooops: field but op==.")
			}
			if idx != -1 && cc.Neg {
				orig.Report(false, fmt.Sprintf("%s: Cookie _was_ set.", ci))
			} else if (idx != -1 && !cc.Neg) || (idx == -1 && cc.Neg) {
				orig.Report(true, ci)
			} else if idx == -1 && !cc.Neg {
				orig.Report(false, fmt.Sprintf("%s: Cookie was not set.", ci))
			} else {
				error("Oooops: This cannot happen....")
			}
		} else {
			if idx == -1 {
				orig.Report(false, fmt.Sprintf("%s: Cookie was not set at all.", ci))
				continue
			}
			rc := resp.SetCookie[idx]
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
				orig.Report(false, fmt.Sprintf("%s: Got '%s'", ci, v))
			} else {
				orig.Report(true, ci)
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
				orig.Report(false, cs)
			} else {
				orig.Report(true, cs)
			}
		case "Bin":
			if !c.BinFullfilled(body) {
				orig.Report(false, cs)
			} else {
				orig.Report(true, cs)
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
		orig.Report(false, "No body to parse.")
		return
	}

	for _, tc := range t.Tag {
		cs := tc.Info("tag")
		switch tc.Cond {
		case TagExpected, TagForbidden:
			n := tag.FindTag(&tc.Spec, doc)
			if tc.Cond == TagExpected {
				if n != nil {
					orig.Report(true, cs)
				} else {
					orig.Report(false, fmt.Sprintf("%s: Missing", cs))
				}
			} else {
				if n == nil {
					orig.Report(true, cs)
				} else {
					orig.Report(false, fmt.Sprintf("%s: Forbidden", cs))
				}
			}
		case CountEqual, CountNotEqual, CountLess, CountLessEqual, CountGreater, CountGreaterEqual:
			got, exp := tag.CountTag(&tc.Spec, doc), tc.Count
			switch tc.Cond {
			case CountEqual:
				if got != exp {
					orig.Report(false, fmt.Sprintf("%s: Found %d expected %d", cs, got, exp))
					continue
				}
			case CountNotEqual:
				if got == exp {
					orig.Report(false, fmt.Sprintf("%s: Found %d expected != %d", cs, got, exp))
					continue
				}
			case CountLess:
				if got >= exp {
					orig.Report(false, fmt.Sprintf("%s: Found %d expected < %d", cs, got, exp))
					continue
				}
			case CountLessEqual:
				if got > exp {
					orig.Report(false, fmt.Sprintf("%s: Found %d expected <= %d", cs, got, exp))
					continue
				}
			case CountGreater:
				if got <= exp {
					orig.Report(false, fmt.Sprintf("%s: Found %d expected > %d", cs, got, exp))
					continue
				}
			case CountGreaterEqual:
				if got < exp {
					orig.Report(false, fmt.Sprintf("%s: Found %d expected >= %d", cs, got, exp))
					continue
				}
			}
			orig.Report(true, cs)
		default:
			error("Unkown type of test %d (%s). Ignored.", tc.Cond, tc.Id)
		}
	}
}

// List of allready checked URLs in this run
var ValidUrls = map[string]bool{}

func shallCheckUrl(url string, base *http.URL) *http.URL {
	if strings.HasPrefix(url, "#") || strings.HasPrefix(strings.ToLower(url), "mailto:") {
		trace("Will not check plain page anchors or mailto links in %s", url)
		return nil
	}
	if j := strings.Index(url, "#"); j != -1 {
		url = url[:j] // Strip #fragment like browsers do
	}
	pu, err := http.ParseURL(url)
	if err != nil {
		error("Cannot parse url " + url)
		return nil
	}
	if !pu.IsAbs() {
		u, e := base.ParseURL(url)
		if e != nil {
			error("Cannot parse %s relative to %s.", url, base.String())
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

	baseUrl, _ := http.ParseURL(base) // Should not fail!
	urls := make(map[string]bool, 50) // keys are urls to prevent doubles

	for _, pat := range []string{"a href", "link href", "img src"} {
		ts, _ := tag.ParseTagSpec(pat) // Wont err
		for _, tg := range tag.FindAllTags(ts, doc) {
			for _, a := range tg.Attr {
				if (a.Key == "href" || a.Key == "src") && a.Val != "" {
					if url := shallCheckUrl(a.Val, baseUrl); url != nil {
						urls[url.String()] = true
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
	for url, _ := range urls {
		if _, ok := ValidUrls[url]; ok {
			warn("Will not retest " + url)
		}
		test := tmpl.Copy()
		test.Url = url
		_, err := test.RunSingle(global, false)
		if err != nil {
			orig.Report(false, fmt.Sprintf("Cannot access `%s': %s", test.Url, err.String()))
			continue
		}
		if _, _, failed := test.Stat(); failed > 0 {
			s := "Failures for " + test.Url + ": "
			for _, r := range test.Result {
				if !strings.HasPrefix(r, "Passed") {
					s += r + "; "
				}
			}
			orig.Report(false, s)
		} else {
			orig.Report(true, "Link "+url)
			ValidUrls[url] = true
		}
	}
}


// Check if html is valid html
func testHtmlValidation(t, orig, global *Test, body string) {
	trace("Validating HTML")
	f, err := ioutil.TempFile("", "htmlvalid")
	if err != nil {
		error("Cannot open temp file: " + err.String())
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
	_, err = test.RunSingle(global, false)
	if err != nil {
		warn("Cannot access W3C validator: %s", err.String())
		return
	}
	if _, _, failed := test.Stat(); failed > 0 {
		orig.Report(false, "html is invalid.")
	} else {
		orig.Report(true, "html is valid")
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


// Add cookie conditions from global to test.  TODO: used/usefull/???
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


func titleToFilename(t string) (f string) {
	// TODO use unicode codepoints
	for i := 0; i < len(t); i++ {
		if t[i] == ' ' {
			f += "_"
		} else if (t[i] >= 'a' && t[i] <= 'z') || (t[i] >= 'A' && t[i] <= 'Z') || (t[i] >= '0' && t[i] <= '0') {
			f += string(t[i])
		} else if t[i] == '-' || t[i] == '+' || t[i] == '.' {
			f += string(t[i])
		}
	}
	f += ".dump"
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
		fname := titleToFilename(test.Title)
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
		dur, e := test.RunSingle(global, false)
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

// Perform a single run of the test.  Return duration for server response in ms.
// If request itself failed, then err is non nil and contains the reason.
// Logs the results of the tests in Result field.
func (test *Test) RunSingle(global *Test, skipTests bool) (duration int, err os.Error) {
	ti := prepareTest(test, global)
	tries := ti.getSetting("Tries")

	var tryCnt int

	for {
		starttime := time.Nanoseconds()
		var (
			response *http.Response
			url      string
			cookies  []*http.Cookie
			reqerr   os.Error
		)

		if ti.Method == "GET" {
			response, url, cookies, reqerr = Get(ti)
		} else if ti.Method == "POST" || ti.Method == "POST:mp" {
			response, url, cookies, reqerr = Post(ti)
		}
		endtime := time.Nanoseconds()
		duration = int((endtime - starttime) / 1000000) // in milliseconds (ms)

		if reqerr != nil {
			test.Report(false, reqerr.String())
			err = Error("Error: " + reqerr.String())
		} else {

			trace("Recieved cookies: %v", cookies)
			if len(cookies) > 0 && test.KeepCookies() == 1 && global != nil {
				if global.Cookie == nil {
					global.Cookie = make(map[string]string)
				}

				for _, c := range cookies {
					if c.MaxAge == -999 { // Delete
						trace("Deleting cookie %s from global (delete req from server).", c.Name)
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
				response.Header.Set("Final-Url", url)
				testHeader(response, ti, test)

				// Body:
				body := readBody(response.Body)
				testBody(body, ti, test)

				// Tag:
				if len(ti.Tag) > 0 || ti.Validate()&1 != 0 {
					var doc *tag.Node
					if parsableBody(response) {
						var e os.Error
						doc, e = tag.ParseHtml(string(body))
						if e != nil {
							test.Report(false, "Problems parsing html: "+e.String())
							error("Problems parsing html: " + e.String())
						}
					} else {
						error("Unparsable body ")
						test.Report(false, "Body considered unparsable.")
					}

					testTags(ti, test, doc)
					if ti.Validate()&1 != 0 {
						testLinkValidation(ti, test, global, doc, response, url)
					}
					if ti.Validate()&2 != 0 {
						testHtmlValidation(ti, test, global, string(body))
					}
				}

				// Timing:
				if max := ti.MaxTime(); max > 0 {
					if duration > max {
						test.Report(false, fmt.Sprintf("Response exeeded Max-Time of %d (was %d).", max, duration))
					} else {
						test.Report(true, fmt.Sprintf("Response took %d ms (allowed %d).", duration, max))
					}
				}

			}
		}

		if test.Sleep() > 0 {
			time.Sleep(1000000 * int64(test.Sleep()))
		}

		tryCnt++
		_, _, failed := test.Stat()
		// fmt.Printf(">>> tryCnt: %d,  tries: %d, failed: %d\n", tryCnt, tries, failed)
		// fmt.Printf("%s\n", test.Status()) 
		if tryCnt >= tries || failed == 0 {
			break
		}
		// clear Result and start over
		test.Result = make([]string, 0)
		// fmt.Printf("\n-----\n%s\n=========\n", test.Status()) 

	}

	return
}
