package suite

import (
	"fmt"
	"http"
	"strings"
	"os"
	"dobler/webtest/tag"
	"encoding/hex"
	"encoding/base64"
	"time"
	"strconv"
	"io"
	"path"
)

var (
	BenchTolerance float32 = 1.3
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
	Setting    map[string]string   // setting like repetition, sleep time, etc. for this test
	Const      map[string]string   // const variables
	Rand       map[string][]string // random varibales
	Seq        map[string][]string // sequence variables
	SeqCnt     map[string]int      // internal stuff for sequnece variables TODO: do not export
	Vars       map[string]string   // internal stuff for variables  TODO: do not export
	Result     []string            // list of pass/fails reports
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
	dest.Setting = copyMap(src.Setting)
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
func (t *Test) Report(pass bool, f string, m ...interface{}) {
	s := fmt.Sprintf(f, m...)
	if pass {
		s = "Passed " + s
		debug(s)
	} else {
		s = "FAILED " + s
		info(s)
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

func NewTest(title string) *Test {
	t := Test{Title: title}

	t.Header = make(map[string]string)
	t.Cookie = make(map[string]string)
	t.Param = make(map[string][]string)
	t.Setting = make(map[string]string)
	t.Const = make(map[string]string)
	t.Rand = make(map[string][]string)
	t.Seq = make(map[string][]string)
	t.SeqCnt = make(map[string]int)
	t.Vars = make(map[string]string)

	t.Setting["Repeat"] = "1"
	t.Setting["Max-Time"] = "-1"
	t.Setting["Sleep"] = "0"
	t.Setting["Keep-Cookies"] = "0"

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


func formatMap(s string, m *map[string]string) (f string) {
	if len(*m) > 0 {
		f = s + "\n"
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

func formatMultiMap(s string, m *map[string][]string) (f string) {
	if len(*m) > 0 {
		f = s + "\n"
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

func formatCond(s string, m *[]Condition) (f string) {
	if len(*m) > 0 {
		f = s + "\n"
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
				f += c.Key + "\n" // fmt.Sprintf("%-*s  %2s  %s\n", longest, c.Key, c.Op, quote(c.Val, false))
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
	s += formatMap("SETTING", &t.Setting)
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

func (t *Test) Repeat() int {
	if t.Setting == nil {
		return 1
	}
	r, ok := t.Setting["Repeat"]
	if !ok {
		warn("Test '%s' does not have Repeat parameter! Will use 1", t.Title)
		r = "1"
	}
	return atoi(r, "", 1)
}

func (t *Test) Sleep() (i int) {
	if t.Setting == nil {
		return 1
	}
	r, ok := t.Setting["Sleep"]
	if !ok {
		r = "0"
	}
	return atoi(r, "", 0)
}

func (t *Test) KeepCookies() bool {
	if t.Setting == nil {
		return false
	}
	kc, ok := t.Setting["Keep-Cookies"]
	if ok && (kc == "1" || kc == "true" || kc == "keep") {
		return true
	}
	return false
}

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

func testHeader(resp *http.Response, t, orig *Test) {
	if len(t.RespCond) > 0 {
		debug("Testing Header")
		for _, c := range t.RespCond {
			cs := c.Info("resp")
			v := resp.Header.Get(c.Key)
			if !c.Fullfilled(v) {
				orig.Report(false, "%s: Got '%s'", cs, v)
			} else {
				orig.Report(true, "%s.", cs)
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
				orig.Report(false, "%s: Cookie _was_ set.", ci)
			} else if (idx != -1 && !cc.Neg) || (idx == -1 && cc.Neg) {
				orig.Report(true, "%s", ci)
			} else if idx == -1 && !cc.Neg {
				orig.Report(false, "%s: Cookie was not set.", ci)
			} else {
				error("Oooops: This cannot happen....")
			}
		} else {
			if idx == -1 {
				orig.Report(false, "%s: Cookie was not set at all.", ci)
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
				v = rc.Expires.Format(time.RFC1123)
			case "Secure":
				v = fmt.Sprintf("%t", rc.Secure)
			case "HttpOnly":
				v = fmt.Sprintf("%t", rc.HttpOnly)
			case "MaxAge":
				v = fmt.Sprintf("%d", rc.MaxAge)
			default:
				error("Oooops: Unknown cookie field %s.", field)
			}
			if !cc.Fullfilled(v) {
				orig.Report(false, "%s: Got '%s'", ci, v)
			} else {
				orig.Report(true, "%s.", ci)
			}
		}
	}
	return
}

func testBody(body string, t, orig *Test) {
	if len(t.BodyCond) > 0 {
		debug("Testing Body")
	} else {
		return
	}

	var binbody *string
	for _, c := range t.BodyCond {
		cs := c.Info("body")
		switch c.Key {
		case "Txt":
			trace("Text Matching '%s'", c.String())
			if !c.Fullfilled(body) {
				orig.Report(false, cs)
			} else {
				orig.Report(true, cs)
			}
		case "Bin":
			if binbody == nil {
				bin := hex.EncodeToString([]byte(body))
				// fmt.Printf("binbody == >>%s<<\n", bin)
				binbody = &bin
			}
			if !c.Fullfilled(*binbody) {
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

func testTags(t, orig *Test, doc *tag.Node) {
	if len(t.Tag) > 0 {
		debug("Testing Tags")
	} else {
		return
	}

	for _, tc := range t.Tag {
		cs := tc.Info("tag")
		switch tc.Cond {
		case TagExpected, TagForbidden:
			n := tag.FindTag(&tc.Spec, doc)
			if tc.Cond == TagExpected {
				if n != nil {
					orig.Report(true, "%s", cs)
				} else {
					orig.Report(false, "%s: Missing", cs)
				}
			} else {
				if n == nil {
					orig.Report(true, "%s", cs)
				} else {
					orig.Report(false, "%s: Forbidden", cs)
				}
			}
		case CountEqual, CountNotEqual, CountLess, CountLessEqual, CountGreater, CountGreaterEqual:
			got, exp := tag.CountTag(&tc.Spec, doc), tc.Count
			switch tc.Cond {
			case CountEqual:
				if got != exp {
					orig.Report(false, "%s: Found %d expected %d", cs, got, exp)
					continue
				}
			case CountNotEqual:
				if got == exp {
					orig.Report(false, "%s: Found %d expected != %d", cs, got, exp)
					continue
				}
			case CountLess:
				if got >= exp {
					orig.Report(false, "%s: Found %d expected < %d", cs, got, exp)
					continue
				}
			case CountLessEqual:
				if got > exp {
					orig.Report(false, "%s: Found %d expected <= %d", cs, got, exp)
					continue
				}
			case CountGreater:
				if got <= exp {
					orig.Report(false, "%s: Found %d expected > %d", cs, got, exp)
					continue
				}
			case CountGreaterEqual:
				if got < exp {
					orig.Report(false, "%s: Found %d expected >= %d", cs, got, exp)
					continue
				}
			}
			orig.Report(true, "%s", cs)
		default:
			error("Unkown type of test %d (%s). Ignored.", tc.Cond, tc.Id)
		}
	}
}


func addMissingHeader(test, global *map[string]string) {
	for k, v := range *global {
		if _, ok := (*test)[k]; !ok {
			(*test)[k] = v
			trace("Adding missing header %s: %s", k, v)
		}
	}
}

func addMissingCookies(test, global *map[string]string) {
	for k, v := range *global {
		if _, ok := (*test)[k]; !ok {
			(*test)[k] = v
			trace("Adding missing cookie %s =%s", k, v)
		}
	}
}

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
	if dd, ok := test.Setting["Dump"]; ok && (dd == "1" || dd == "2") {
		fname := titleToFilename(test.Title)
		var mode int = os.O_TRUNC
		if dd == "2" {
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
	for i := 1; i <= test.Repeat(); i++ {
		test.RunSingle(global, true)
	}
	return
}


func (test *Test) Bench(global *Test, count int) (durations []int, failures int, err os.Error) {
	test.init()
	test.Dump = nil // prevent dumping during benchmarking

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
func (test *Test) RunSingle(global *Test, skipTests bool) (duration int, err os.Error) {

	ti := prepareTest(test, global)

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
	duration = int((endtime - starttime) / 1000000) // in milli seconds (ms)

	if reqerr != nil {
		test.Report(false, reqerr.String())
		err = Error("Error: " + reqerr.String())
		return
	}

	trace("Recieved cookies: %v", cookies)
	if len(cookies) > 0 && test.KeepCookies() && global != nil {
		if global.Cookie == nil {
			global.Cookie = make(map[string]string)
		}

		for _, c := range cookies {
			// TODO: Test for overwrite/modify
			global.Cookie[c.Name] = c.Value
			trace("kept cookie %s = %s (global = %p)", c.Name, c.Value, global)
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
		if len(ti.Tag) > 0 {
			var doc *tag.Node
			if parsableBody(response) {
				var e os.Error
				doc, e = tag.ParseHtml(body)
				if e != nil {
					error("Problems parsing html: " + e.String())
				}
			} else {
				error("Unparsable body ")
				test.Report(false, "Unparsabel Body. Skipped testing Tags.")
			}
			if doc != nil {
				testTags(ti, test, doc)
			} else {
				test.Report(false, "Problems parsing Body. Skipped testing Tags.")
			}
		}

		// Timing:
		if v, ok := ti.Setting["Max-Time"]; ok {
			max, err := strconv.Atoi(v)
			if err != nil {
				error("This should not happen: Max-Time is not an int.")
			} else if max > 0 && duration > max {
				test.Report(false, "Response exeeded Max-Time of %d (was %d).", max, duration)
			}
		}

		_, _, failed := test.Stat()
		if failed != 0 {
			err = Error("Failure: " + test.Status())
		}
	}

	time.Sleep(1000000 * int64(test.Sleep()))
	return
}
