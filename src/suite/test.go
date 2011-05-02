package suite

import (
	"fmt"
	"http"
	"strings"
	"os"
	"./../tag/tag"
	"encoding/hex"
	"time"
)

var (
	BenchTolerance float32 = 1.3
)

type Error string
func (e Error) String() string {
	return string(e)
}

type Test struct {
	Title    string
	Method   string
	Url      string
	Header   map[string]string
	RespCond []Condition
	BodyCond []Condition
	Pre      []string
	// MaxTime  int // -1: unset, 0=no limit, >0: limit in ms
	// Sleep    int // -1: unset, >=0: sleep after in ms
	// Repeat   int // -1: unset, 0=disabled, >0: count
	Param   map[string][]string
	Setting map[string]string
	Const   map[string]string
	Rand    map[string][]string
	Seq     map[string][]string
	SeqCnt  map[string]int
	Vars    map[string]string
	Result  []string
}

func (t *Test) Report(pass bool, f string, m ...interface{}) {
	s := fmt.Sprintf(f, m...)
	if pass {
		s = "Passed " + s
		info(s)
	} else {
		s = "FAILED " + s
		error(s)
	}
	t.Result = append(t.Result, s)
}

func (t *Test) Stat() (total, passed, failed int) {
	for _, r := range t.Result {
		total++
		trace("Result: %s", r)
		if strings.HasPrefix(r, "Passed") {
			passed++
		} else {
			failed++
		}
	}
	return
}

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

	t.Header = make(map[string]string, 3)
	t.Param = make(map[string][]string, 3)
	t.Setting = make(map[string]string, 3)
	t.Const = make(map[string]string, 3)
	t.Rand = make(map[string][]string, 3)
	t.Seq = make(map[string][]string, 3)
	t.SeqCnt = make(map[string]int, 3)
	t.Vars = make(map[string]string, 3)

	t.Setting["Repeat"] = "1"
	t.Setting["MaxTime"] = "unlimited"
	t.Setting["Sleep"] = "0"

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
			f += fmt.Sprintf("%-*s  %2s  %s\n", longest, c.Key, c.Op, quote(c.Val, false))
		}
	}
	return
}


func (t *Test) String() (s string) {
	s = "-------------------------------\n" + t.Title + "\n-------------------------------\n"
	s += t.Method + " " + t.Url + "\n"
	s += formatMap("HEADER", &t.Header)
	s += formatCond("RESPONSE", &t.RespCond)
	s += formatCond("BODY", &t.BodyCond)
	s += formatMultiMap("PARAM", &t.Param)
	s += formatMap("CONST", &t.Const)
	s += formatMultiMap("SEQ", &t.Seq)
	s += formatMultiMap("RAND", &t.Rand)

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
	return atoi(r, 0, 1)
}

func (t *Test) Sleep() (i int) {
	if t.Setting == nil {
		return 1
	}
	r, ok := t.Setting["Sleep"]
	if !ok {
		r = "0"
	}
	return atoi(r, 0, 0)
}


func testHeader(resp *http.Response, t, orig *Test) {
	debug("Testing Header")
	for _, c := range t.RespCond {
		cs := c.Info("resp", false)
		v := resp.Header.Get(c.Key)
		if !c.Fullfilled(v) {
			orig.Report(false, "%s: Got '%s'", cs, v)
		} else {
			orig.Report(true, "%s.", cs)
		}
	}
	return
}

func testBody(body string, t, orig *Test, doc *tag.Node) {
	debug("Testing Body")
	var binbody *string
	for _, c := range t.BodyCond {
		cs := c.Info("body", true)
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
				binbody = &bin
			}
			if !c.Fullfilled(*binbody) {
				orig.Report(false, cs)
			} else {
				orig.Report(true, cs)
			}
		case "Tag":
			if doc == nil {
				error("FAILED %s: Document unparsable.", cs)
				continue
			}
			ts := tag.ParseTagSpec(c.Val)
			if c.Op == "" { // no counting
				n := tag.FindTag(ts, doc)
				if n == nil && !c.Neg {
					orig.Report(false, "%s: Missing", cs)
				} else if n != nil && c.Neg {
					orig.Report(false, "%s: Forbidden", cs)
				} else {
					orig.Report(true, "%s", cs)
				}
			} else {
				warn("Tag counting not implemented jet (line %d).", c.Line)
			}
		default:
			error("Unkown type of test '%s' (line %d). Ignored.", c.Key, c.Line)
		}
	}
	return
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
	debug("Preparing test '%s'.", t.Title)

	// Clear map of variable values: new run, new values (overkill for consts)
	for k, _ := range t.Vars {
		t.Vars[k] = "", false
	}

	test := *t // create copy
	if global != nil {
		test.RespCond = addMissingCond(test.RespCond, global.RespCond)
		test.BodyCond = addAllCond(test.BodyCond, global.BodyCond)
	}
	substituteVariables(&test, global, t)
	debug("Test to execute = \n%s", test.String())
	return &test
}

func parsableBody(resp *http.Response) bool {
	if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return true
	} else if strings.Contains(resp.Header.Get("Content-Type"), "text/xml") {
		return true
	}
	info("Response body is not considered parsable")
	return false
}

func (test *Test) Init() {
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
	test.Init()

	if test.Repeat() == 0 {
		info("Test no '%s' is disabled.", test.Title)
	} else {
		for i := 1; i <= test.Repeat(); i++ {
			info("Test '%s': Round %d of %d.", test.Title, i, test.Repeat())
			test.RunSingle(global)
		}
	}

	info("Test '%s': %s", test.Title, test.Status())
	return
}

func (test *Test) Bench(global *Test, count int) (durations []int, failures int, err os.Error) {
	test.Init()

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
		dur, e := test.RunSingle(global)
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
func (test *Test) RunSingle(global *Test) (duration int, err os.Error) {

	ti := prepareTest(test, global)
	info("Running test '%s'", ti.Title)

	if ti.Method != "GET" {
		error("Post not jet implemented")
		duration = -1
		return
	}

	starttime := time.Nanoseconds()
	response, url, geterr := Get(ti)
	endtime := time.Nanoseconds()
	duration = int(endtime - starttime)/1000000
	
	if geterr != nil {
		test.Report(false, geterr.String())
		err = Error("Error: " + geterr.String())
		return
	}

	// Add special fields to header
	response.Header.Set("StatusCode", fmt.Sprintf("%d", response.StatusCode))
	response.Header.Set("Url", url)
	testHeader(response, ti, test)

	body := readBody(response.Body)
	var doc *tag.Node
	if parsableBody(response) {
		doc = tag.ParseHtml(body)
	}
	testBody(body, ti, test, doc)

	_, _, failed := test.Stat()
	if failed != 0 {
		err = Error("Failure: " + geterr.String())
	}
	
	time.Sleep(1000 * int64(test.Sleep()))
	return
}
