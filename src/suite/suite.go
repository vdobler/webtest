package suite

import (
	"fmt"
	// "bufio"
	"os"
	"http"
	"log"
	"strings"
	// "strconv"
	"./../tag/tag"
)

var logLevel int = 3 // 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace

const MaxConditionLen = 40

func error(f string, m ...interface{}) {
	if logLevel >= 1 {
		log.Print("*ERROR* " + fmt.Sprintf(f, m...))
	}
}
func warn(f string, m ...interface{}) {
	if logLevel >= 2 {
		log.Print("*WARN * " + fmt.Sprintf(f, m...))
	}
}
func info(f string, m ...interface{}) {
	if logLevel >= 3 {
		log.Print("*INFO * " + fmt.Sprintf(f, m...))
	}
}
func debug(f string, m ...interface{}) {
	if logLevel >= 4 {
		log.Print("*DEBUG* " + fmt.Sprintf(f, m...))
	}
}
func trace(f string, m ...interface{}) {
	if logLevel >= 5 {
		log.Print("*TRACE* " + fmt.Sprintf(f, m...))
	}
}


type Test struct {
	Title    string
	Method   string
	Url      string
	Header   map[string]string
	RespCond []Condition
	BodyCond []Condition
	Pre      []string
	MaxTime  int // -1: unset, 0=no limit, >0: limit in ms
	Sleep    int // -1: unset, >=0: sleep after in ms
	Repeat   int // -1: unset, 0=disabled, >0: count
	Param    map[string]string
	Const    map[string]string
	Rand     map[string][]string
	Seq      map[string][]string
	SeqCnt   map[string]int
	Vars     map[string]string
	Run      bool
	Passed   bool
}

func (t *Test) String() (s string) {
	s = "-------------------------------\n" + t.Title + "\n-------------------------------\n"
	s += t.Method + " " + t.Url + "\n"
	if len(t.Header) > 0 {
		s += "HEADER:\n"
		for k, v := range t.Header {
			s += "\t" + k + ": " + v + "\n"
		}
	}
	if len(t.RespCond) > 0 {
		s += "RESPONSE:\n"
		for _, cond := range t.RespCond {
			s += "\t" + cond.String() + "\n"
		}
	}
	if len(t.BodyCond) > 0 {
		s += "Body:\n"
		for _, cond := range t.BodyCond {
			s += "\t" + cond.String() + "\n"
		}
	}
	return
}

type TestError struct {
	os.ErrorString
}

var (
	ErrTimeout = &TestError{"Connection timed out."}
	ErrSystem  = &TestError{"Underlying system failed."}
	ErrTest    = &TestError{"Failed Test."}
)


// TODO: Results?
type Suite struct {
	Test   []Test
	Result map[string]int // 0: not run jet, 1: pass, 2: fail, 3: err
}

func (c *Condition) Info(txt string) string {
	// TODO: remove/mask/... newlines
	vs := c.String()
	if len(vs) > MaxConditionLen {
		vs = vs[:MaxConditionLen] + "..."
	}
	cs := fmt.Sprintf("%s (line %d) '%s'", txt, c.Line, vs)
	return cs
}

func report(f string, m ...interface{}) {
	s := fmt.Sprintf(f, m...)
	if strings.HasPrefix(s, "FAILED") {
		error(s)
	} else {
		info(s)
	}
}

func testHeader(resp *http.Response, t *Test) (err os.Error) {
	debug("Testing Header")
	for _, c := range t.RespCond {
		cs := c.Info("resp")
		v := resp.Header.Get(c.Key)
		if !c.Fullfilled(v) {
			report("FAILED %s: Got '%s'", cs, v)
			err = ErrTest
		} else {
			report("Passed %s.", cs)
		}
	}
	return
}

func testBody(body string, t *Test, doc *tag.Node) (err os.Error) {
	debug("Testing Body")
	for _, c := range t.BodyCond {
		cs := c.Info("body")
		switch c.Key {
		case "Text":
			trace("Text Matching '%s'", c.String())
			if !c.Fullfilled(body) {
				report("FAILED %s", cs)
				err = ErrTest
			} else {
				report("Passed %s", cs)
			}
		case "Bin":
			error("Unimplemented: Binary matching")
			err = ErrSystem
		case "Tag":
			if doc == nil {
				error("Faild body tag condition (line %d): Document unparsable.", c.Line)
				err = ErrSystem
				continue
			}
			ts := tag.ParseTagSpec(c.Val)
			if c.Op == "" { // no counting
				n := tag.FindTag(ts, doc)
				if n == nil && !c.Neg {
					report("FAILED %s: Missing", cs)
					err = ErrTest
				} else if n != nil && c.Neg {
					report("FAILED %s: Forbidden", cs)
					err = ErrTest
				} else {
					report("Passed %s", cs)
				}
			} else {
				warn("Tag counting not implemented jet (line %d).", c.Line)
			}
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
	trace("Len(test) == %d.", len(test))
	return test
}

func addAllCond(test, global []Condition) []Condition {
	for _, cond := range global {
		trace("Adding body condition '%s'", cond.String())
		test = append(test, cond)
	}
	trace("Len(test) == %d.", len(test))
	return test
}


// Prepare the test: Add new stuff from global
func prepareTest(s *Suite, n int) *Test {
	debug("Preparing test no %d.", n)

	// Clear map of variable values: new run, new values (overkill for consts)
	for k, _ := range s.Test[n].Vars {
		s.Test[n].Vars[k] = "", false
	}

	test := s.Test[n]
	global := s.Test[0]
	test.RespCond = addMissingCond(test.RespCond, global.RespCond)
	test.BodyCond = addAllCond(test.BodyCond, global.BodyCond)
	// info("#Headers: %d, #RespCond: %d, #BodyCond: %d", len(test.Header), len(test.RespCond), len(test.BodyCond))


	substituteVariables(&test, &global, &s.Test[n])
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

func (s *Suite) RunTest(n int) (err os.Error) {

	tt := &s.Test[n]
	// Initialize sequenze count
	if tt.SeqCnt == nil {
		tt.SeqCnt = make(map[string]int, len(tt.Seq))
		for k, _ := range tt.Seq {
			tt.SeqCnt[k] = 0
		}
	}

	// Initialize storage for current value of vars
	if s.Test[n].Vars == nil {
		cnt := len(tt.Seq) + len(tt.Rand) + len(tt.Const)
		tt.Vars = make(map[string]string, cnt)
	}

	if tt.Repeat == 0 {
		info("Test no %d '%s' is disabled.", n, tt.Title)
	} else {
		for i := 1; i <= tt.Repeat; i++ {
			info("Test %d '%s': Round %d of %d.", n, tt.Title, i, tt.Repeat)
			s.RunSingleRound(n)
		}
	}
	return
}

func (s *Suite) RunSingleRound(n int) (err os.Error) {

	t := prepareTest(s, n)
	info("Running test %d: '%s'", n, t.Title)

	if t.Method != "GET" {
		error("Post not jet implemented")
		return ErrSystem
	}

	response, url, err := Get(t)

	if err != nil {
		error(err.String())
		return ErrSystem
	}

	// Add special fields to header
	response.Header.Set("StatusCode", fmt.Sprintf("%d", response.StatusCode))
	response.Header.Set("Url", url)
	herr := testHeader(response, t)

	body := readBody(response.Body)
	var doc *tag.Node
	if parsableBody(response) {
		doc = tag.ParseHtml(body)
	}

	berr := testBody(body, t, doc)

	if herr != nil || berr != nil {
		err = ErrTest
	}

	return
}
