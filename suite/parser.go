package suite

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/vdobler/webtest/tag"
)

var DefaultSettings = map[string]int{"Repeat": 1,
	"Tries":        1,
	"Max-Time":     -1,
	"Sleep":        0,
	"Keep-Cookies": 0,
	"Abort":        0,
	"Dump":         0,
}

type ParserError struct {
	cause string
}

func (pe ParserError) Error() string {
	return pe.cause
}

type Parser struct {
	reader *bufio.Reader
	line   []string
	test   *Test
	suite  []Test
	i      int
	name   string
	errors []string
}

// Set up a new Parser which reads a suite from r (named name).
func NewParser(r io.Reader, name string) *Parser {
	parser := new(Parser)
	parser.reader = bufio.NewReader(r)
	parser.line = []string{}
	parser.suite = []Test{}
	parser.errors = []string{}
	parser.name = name
	return parser
}

// Log an error in the input file
func (p *Parser) error(f string, m ...interface{}) {
	e := fmt.Sprintf("%s:%d: %s", p.name, p.i+1, fmt.Sprintf(f, m...))
	p.errors = append(p.errors, e)
}

// check if all okay
func (p *Parser) okay() bool {
	return len(p.errors) == 0
}

// Read a line from the Reader
func (p *Parser) nextLine() (line string, err error) {
	var isprefix bool
	var by []byte
	var str string
	by, isprefix, err = p.reader.ReadLine()
	if err != nil {
		return
	}
	for isprefix {
		str += string(by)
		by, isprefix, err = p.reader.ReadLine()
		if err != nil {
			return
		}
	}
	str += string(by)
	p.i++
	return str, nil
}

func isComment(line string) bool {
	line = trim(line)
	return hp(line, "#")
}

// Fill list of lines.
func (p *Parser) readLines() {
	p.i = 0
	for {
		line, err := p.nextLine()
		if err != nil {
			return
		}
		p.line = append(p.line, line)
	}
}

// Abrevations for strings.HasPrefix
func hp(s, p string) bool {
	return strings.HasPrefix(s, p)
}

// Abrevation for strings.HasSuffix
func hs(s, p string) bool {
	return strings.HasSuffix(s, p)
}

func trim(s string) string {
	return strings.Trim(s, " \t")
}

func deescape(str string) string {
	str = fmt.Sprintf("%#v", str)
	return str
}

func dequote(str string) (string, error) {
	if len(str) < 2 {
		return str, nil
	}
	if hp(str, "\"") && hs(str, "\"") {
		return strconv.Unquote(str)
	}
	return str, nil
}

// Return index of first space/tab in s or -1 if none found.
func firstSpace(s string) int {
	return strings.IndexAny(s, " \t")
}

// Read a string->string map. Stopp if unindented line is found
func (p *Parser) readMap(m *map[string]string) {
	for p.i < len(p.line)-1 {
		done, _, key, _, val := p.nextStuff([]string{":="})
		if done {
			return
		}
		if s, err := dequote(val); err != nil {
			p.error("Malformed string '%s'.", val)
			continue
		} else {
			val = s
		}
		(*m)[key] = val
		tracef("Added to map (line %d): %s: %s", p.i, key, val)
	}
}

// parse smth like  "name:domain:path:Secure ~= value", line must be trimmed
func parseCookie(key, host string) (name, domain, path, field string, err error) {
	cls := strings.Split(key, ":")
	name = cls[0]
	switch len(cls) {
	case 4:
		domain, path, field = cls[1], cls[2], strings.ToLower(cls[3])
	case 3:
		domain, path = cls[1], cls[2]
	case 2:
		if cls[1] != "" && cls[1][0] == '/' {
			path = cls[1]
		} else {
			domain = cls[1]
		}
	case 1:
	default:
		err = errors.New("Too many ':' in cookie definition.")
		return
	}

	// Add defaults of path and domain
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		err = fmt.Errorf("Illegal path '%s' in cookie.", path)
		return
	}
	if domain == "" {
		domain = host
	}

	return
}

//
func (p *Parser) readCookieCond(host string) (cc []Condition) {
	cc = make([]Condition, 0, 10)
	for p.i < len(p.line)-1 {
		done, neg, key, op, value := p.nextStuff([]string{"==", "~=", "_=", "=_", "/=", ">", ">=", "<", "<="})
		if done {
			return
		}

		cond := Condition{}
		cond.Neg = neg

		name, domain, path, field, err := parseCookie(key, host)
		if err != nil {
			p.error("%s", err.Error())
			continue
		}
		switch field {
		case "":
			field = "value"
		case "secure", "httponly", "maxage", "expires", "delete", "deleted", "value":
		default:
			p.error("Unknown cookie field '%s'.", field)
			continue
		}

		cond.Key = fmt.Sprintf("%s:%s:%s:%s", name, domain, path, field)
		cond.Op = op
		dval, err := dequote(value)
		if err != nil {
			p.error("Cannot parse string '%s': %s", value, err.Error())
		} else {
			value = dval
		}
		cond.Val = value
		cond.Id = fmt.Sprintf("%s:%d", p.name, p.i)
		cc = append(cc, cond)
	}
	return
}

//
func (p *Parser) readSendCookies(jar *CookieJar, host string) {
	for p.i < len(p.line)-1 {
		done, _, key, _, value := p.nextStuff([]string{":="})
		if done {
			return
		}
		name, domain, path, field, err := parseCookie(key, host)
		if err != nil {
			p.error("%s.", err.Error())
			continue
		}
		if field != "" && field != "secure" {
			p.error("Wrong field '%s' (only secure allowed).", field)
			continue
		}
		dval, err := dequote(value)
		if err != nil {
			p.error("Cannot parse string '%s': %s", value, err.Error())
		} else {
			value = dval
		}

		cookie := http.Cookie{Name: name, Domain: domain, Path: path, Value: value, Secure: field == "secure"}
		jar.Update(cookie, "")
	}
}

// Read a string->int map for settings. Stopp if unindented line is found
func (p *Parser) readSettingMap(m *map[string]int) {
	for p.i < len(p.line)-1 {
		done, _, key, _, val := p.nextStuff([]string{":="})
		if done {
			return
		}

		if _, ok := DefaultSettings[key]; !ok {
			p.error("Unknown settign '%s'.", key)
			continue
		}

		val = strings.ToLower(val)
		var n int
		// Some numbers may be given as cleartext. This allows stuff like
		// SETTING
		//     Dump         append
		//     Keep-Cookies keep
		//     Abort        false
		switch val {
		case "false", "no", "nein", "non":
			n = 0
		case "true", "yes", "ja", "qui", "create", "new", "keep", "abort":
			n = 1
		case "append":
			n = 2
		case "both", "body":
			n = 3
		default:
			var err error
			n, err = strconv.Atoi(val)
			if err != nil {
				p.error("Cannot convert %s to integer.", val)
			}
		}

		// Safeuard against stupid or wrong settings.
		switch key {
		case "Repeat":
			if n > 100 {
				warnf("More then 100 repetitions on line %d.", p.i)
			}
		case "Tries":
			if n <= 0 {
				warnf("Setting Tries to value <= 0 is unsensical line %d.", p.i)
			}
		case "Keep-Cookies", "Abort":
			if n != 0 && n != 1 {
				warnf("Keep-Cookies and Abort accept only 0 and 1 as value on line %d.", p.i)
			}
		case "Dump":
			if n < 0 || n > 3 {
				warnf("Dump accepts only 0, 1 and 2 as value (was %s=%d) on line %d.", val, n, p.i)
			}
		}
		(*m)[key] = n
		tracef("Added to settings-map (line %d): %s: %s", p.i, key, val)
	}
}

// Split line at spaces into fields. Quotes can be used to hold together a
// filed containing spaces. E.g.
// 		cat dog "foo bar" fish "mouse" shark
// would yield
//		cat
//		dog
//		foo bar
//		fisch
//		mouse
//		shark
func StringList(line string) (list []string, err error) {

	for len(line) > 0 {
		var quoted bool
		var j int
		if line[0] == '"' {
			j = endQuoteIndex(line)
			quoted = true
		} else {
			j = nextSpaceIndex(line) - 1
			quoted = false
		}

		var p string
		if j < len(line) {
			p = line[0 : j+1]
			line = line[j+1:]
		} else {
			p = line[0:j]
			if quoted {
				p += "\"" // gracefuly add missing " at end
			}
			line = ""
		}
		if quoted {
			if p, err = dequote(p); err != nil {
				return
			}
		}
		list = append(list, p)
		line = strings.TrimLeft(line, " ")
	}
	return
}

func nextSpaceIndex(line string) int {
	n := len(line)
	for i := 1; i < n; i++ {
		if line[i] == ' ' {
			return i
		}
	}
	return n
}

func endQuoteIndex(line string) int {
	// fmt.Printf("endQuoteIndex: %s\n", line)
	n := len(line)
	var lwb bool // LastWasBackslash
	for i := 1; i < n; i++ {
		//fmt.Printf("  %d: ", i)
		if line[i] == '\\' {
			lwb = !lwb
			// fmt.Printf(" toggled lwb to %t\n", lwb)
			continue
		}
		if line[i] == '"' && !lwb {
			// fmt.Printf(" found quote\n")

			if (i+1 < n && line[i+1] == ' ') || (i+1 == n) {
				return i
			}
		} else {
			// fmt.Printf("other %c\n", line[i])
		}
		lwb = false
	}
	return n
}

// Like readMap, but treat value as list of strings
func (p *Parser) readMultiMap(m *map[string][]string) {
	for p.i < len(p.line)-1 {
		done, _, key, _, val := p.nextStuff([]string{":="})
		if done {
			return
		}
		var list []string
		if val == "" {
			list = []string{}
		} else {
			var err error
			list, err = StringList(val)
			if err != nil {
				p.error("Cannot decode '%s': %s.", val, err.Error())
				continue
			}
		}
		(*m)[key] = list
		tracef("Added to mulit map (line %d): key: %v", p.i, key, list)
	}
}

// Parse strings like "[:10]" or "[50:-2]" into a Range.
func parseRange(s string) (r Range, err error) {
	if s == "" {
		return
	}

	if !hp(s, "[") || !hs(s, "]") {
		err = errors.New("Missing [ or ].")
		return
	}
	s = s[1 : len(s)-1]
	var ss []string
	if s == ":" {
		ss = []string{"", ""}
	} else if hp(s, ":") {
		ss = []string{"", s[1:]}
	} else if hs(s, ":") {
		ss = []string{s[:len(s)-1], ""}
	} else {
		ss = strings.Split(s, ":")
	}
	// fmt.Printf("s='%s'  ss = %#v\n", s, ss)
	if len(ss) != 2 {

		err = errors.New("Missing or multiple :")
		return
	}

	if ss[0] != "" {
		n, e := strconv.Atoi(ss[0])
		if e != nil {
			err = e
			return
		}
		r.Low, r.N = true, n
	}
	if ss[1] != "" {
		m, e := strconv.Atoi(ss[1])
		if e != nil {
			err = e
			return
		}
		r.High, r.M = true, m
	}
	return
}

// Read a list of shell conditions
func (p *Parser) readShellCond() [][]string {
	var list [][]string = make([][]string, 0, 3)

	for p.i < len(p.line)-1 {
		p.i++
		line := p.line[p.i]
		if isComment(line) {
			continue
		}
		if !hp(line, "\t") {
			p.i--
			break
		}

		line = trim(line)
		args, err := StringList(line)
		if err != nil {
			p.error("Unable to parse command '%s': %s", line, err.Error())
			continue
		}
		list = append(list, args)
	}
	return list
}

// find, split and return next line
func (p *Parser) nextStuff(validOps []string) (done, neg bool, key, op, val string) {
	for p.i < len(p.line)-1 {
		p.i++
		line := p.line[p.i]
		if isComment(line) || len(trim(line)) == 0 {
			continue
		}

		if !hp(line, "\t") {
			p.i--
			done = true
			return
		}

		line = trim(line)
		if validOps[0] != ":=" {
			if line[0] == '!' {
				line = trim(line[1:])
				neg = true
			}
		}

		j := firstSpace(line)
		if j == -1 {
			key = line
			return
		}

		key, line = line[:j], trim(line[j:])
		j = firstSpace(line)
		if j == -1 {
			p.error("Missing operator (%v).", validOps)
			continue
		}

		op, val = line[:j], trim(line[j:])
		found := false
		for _, o := range validOps {
			if op == o {
				found = true
				break
			}
		}
		if !found {
			p.error("Illegal operator '%s'.", op)
			continue
		}

		return
	}
	done = true
	return
}

// Read a Header or Body Condition
func (p *Parser) readCond(mode int) []Condition {
	var list []Condition = make([]Condition, 0, 3)

	for p.i < len(p.line)-1 {
		done, neg, key, op, val := p.nextStuff(
			[]string{"==", "~=", "_=", "=_", "/=", ">", ">=", "<", "<="})
		if done {
			return list
		}
		var rng Range

		if mode == mode_body {
			if val == "" {
				p.error("Missing value for body condition.")
				continue
			}
			if !(hp(key, "Txt") || hp(key, "Bin")) {
				p.error("No such condition type '%s' for body.", key)
				continue
			}
			rs := ""
			key, rs = key[:3], key[3:]

			// optional range
			if rs != "" {
				if rg, err := parseRange(rs); err != nil {
					p.error("Unable to parse range '%s': %s", rs, err.Error())
					continue
				} else {
					rng = rg
				}
			}

			if key == "Bin" {
				v := strings.ToLower(strings.Replace(val, " ", "", -1))
				if len(v)%2 == 1 {
					warnf("Odd number of nibbles in binary value on line %d. Will discard last nibble.", p.i)
					v = v[:len(v)-2]
				}
				n := len(v) / 2
				var c byte
				for i := 0; i < n; i++ {
					r, err := fmt.Sscanf(v[2*i:2*i+2], "%x", &c)
					if err != nil || r != 1 {
						p.error("Cannot parse '%s' in hex string '%s'.",
							v[2*i:2*i+2], v, p.i)
						break
					}
				}
				val = v
			}
		}

		var dval string
		var err error
		if dval, err = dequote(val); err != nil {
			p.error("Cannot parse string '%s': %s", val, err)
			continue
		}
		//fmt.Printf("\nvvvvvvvvvvvvvvvvvvvvvvvvvvv\nOrig=%s\nValu=%s\nDeqt=%s\n^^^^^^^^^^^^^^^^^^^^^^\n",
		//	p.line[p.i], val, dval)

		id := fmt.Sprintf("%s:%d", p.name, p.i)
		cond := Condition{Key: key, Op: op, Val: dval, Neg: neg, Id: id, Range: rng}
		list = append(list, cond)
		tracef("Added to condition (line %d): %s", p.i, cond.String())
	}
	return list
}

// List of valid tag-name --> attrib-name combinations on html where the
// attrib points to some external URL.
var knownLinkAttr = map[string]string{
	"link":   "href",
	"a":      "href",
	"frame":  "src",
	"iframe": "src",
	"img":    "src",
	"script": "src",
}

// Read a Validation conditions
func (p *Parser) readValidation() []string {
	var list []string = make([]string, 0, 3)

	for p.i < len(p.line)-1 {
		p.i++
		line := p.line[p.i]
		if isComment(line) || len(trim(line)) == 0 {
			continue
		}

		if !hp(line, "\t") {
			p.i--
			return list
		}
		line = trim(line)
		// TODO: lowercase

		// id := fmt.Sprintf("%s:%d", p.name, p.i)
		list = append(list, line)
		tracef("Added to validation (line %d): %s", p.i, line)
	}
	return list
}

// Read a Header or Body Condition
func (p *Parser) readLogCond() []LogCondition {
	var list []LogCondition = make([]LogCondition, 0, 3)

	for p.i < len(p.line)-1 {
		done, neg, key, op, val := p.nextStuff(
			[]string{"~=", "_=", "=_", "/=", ">", "<"})
		if done {
			return list
		}

		var err error
		if val, err = dequote(val); err != nil {
			errorf("Malformed string '%s': %s", val, err.Error())
			continue
		}
		id := fmt.Sprintf("%s:%d", p.name, p.i)
		cond := LogCondition{Path: key, Op: op, Val: val, Neg: neg, Id: id}
		list = append(list, cond)
		tracef("Added to condition (line %d): %s", p.i, cond.String())
	}
	return list
}

// Helper to extract count an spec from strings like ">= 5  a href=/index.html"
// off is the number of charactes to strip before trying to read an int.
func numStr(line string, off int) (n int, spec string, err error) {
	tracef("line = %s, off = %d", line, off)
	beg := line[:off]
	line = trim(line[off:])
	i := firstSpace(line)
	if i < 0 {
		i = strings.Index(line, "[")
		if i < 0 {
			err = errors.New(fmt.Sprintf("Missing space after %s", beg))
			return
		}
	}
	n, err = strconv.Atoi(line[:i])
	if err != nil {
		return
	}
	spec = trim(line[i:])
	tracef("n=%d, spec=%s", n, spec)
	return
}

// Reads the following tag conditions (like readMap)
func (p *Parser) readTagCond() []TagCondition {
	var list []TagCondition = make([]TagCondition, 0, 3)

	for p.i < len(p.line)-1 {
		p.i++
		line := p.line[p.i]
		if isComment(line) || len(trim(line)) == 0 {
			continue
		}

		if !hp(line, "\t") {
			p.i--
			return list
		}
		line = trim(line)
		for hp(line, "! ") { //  transform "!  =3 a href == xyz" to "!=3 a href == xyz"
			line = "!" + line[2:]
		}

		cond := TagCondition{}
		cond.Id = fmt.Sprintf("%s:%d", p.name, p.i)
		var spec string
		var err error

		if false {
		} else if hp(line, "!=") {
			cond.Cond = CountNotEqual
			cond.Count, spec, err = numStr(line, 2)
		} else if hp(line, "!>=") {
			cond.Cond = CountLess
			cond.Count, spec, err = numStr(line, 3)
		} else if hp(line, "!>") {
			cond.Cond = CountLessEqual
			cond.Count, spec, err = numStr(line, 2)
		} else if hp(line, "!<=") {
			cond.Cond = CountGreater
			cond.Count, spec, err = numStr(line, 3)
		} else if hp(line, "!<") {
			cond.Cond = CountGreaterEqual
			cond.Count, spec, err = numStr(line, 2)
		} else if hp(line, "!") {
			cond.Cond = TagForbidden
			spec = line[1:]
		} else if hp(line, "==") {
			cond.Cond = CountEqual
			cond.Count, spec, err = numStr(line, 2)
		} else if hp(line, "=") {
			cond.Cond = CountEqual
			cond.Count, spec, err = numStr(line, 1)
		} else if hp(line, ">=") {
			cond.Cond = CountGreaterEqual
			cond.Count, spec, err = numStr(line, 2)
		} else if hp(line, ">") {
			cond.Cond = CountGreater
			cond.Count, spec, err = numStr(line, 1)
		} else if hp(line, "<=") {
			cond.Cond = CountLessEqual
			cond.Count, spec, err = numStr(line, 2)
		} else if hp(line, "<") {
			cond.Cond = CountLess
			cond.Count, spec, err = numStr(line, 1)
		} else {
			cond.Cond = TagExpected
			spec = line
		}
		if err != nil {
			p.error("Unable to determin count: %s", err.Error())
			continue
		}

		spec = trim(spec)
		if hp(spec, "[") { // multiline tag spec
			tracef("Multiline tag spec")
			spec = ""
			for p.i < len(p.line)-1 {
				p.i++
				line := p.line[p.i]
				tracef("Next line: %s", line)
				if !hp(line, "\t") {
					p.error("Nonindented line in multiline tag spec.")
					break
				}
				if hs(trim(line), "]") {
					tracef("End of multiline tag spec found in line %d.", p.i)
					break
				}
				if spec == "" {
					line = trim(line)
				} else {
					spec += "\n"
				}
				spec += line
				supertracef("Spec now: '%#v'", spec)
			}
			// fmt.Printf("\n-------------------\n%s\n----------------------\n", spec)
		}

		if ts, err := tag.ParseTagSpec(spec); err == nil {
			cond.Spec = *ts
			list = append(list, cond)
			tracef("Added to tag condition (line %d): %s", p.i, cond.String())
		} else {
			p.error("Problems parsing tagspec %#v: %s", spec, err.Error())
		}
	}
	return list
}

const (
	mode_response = iota
	mode_body
)

func (p *Parser) readGetPost(line string) (method, u string) {
	if hp(line, "GET") {
		method, u = "GET", trim(line[3:])
	} else if hp(line, "POST:mp ") {
		method, u = "POST:mp", trim(line[7:])
	} else if hp(line, "POST ") {
		method, u = "POST", trim(line[4:])
	}

	if i := strings.Index(u, "#"); i != -1 {
		warnf("URL may not contain fragment (#-part) in line %d.", p.i)
		u = u[:i]
	}
	if _, ue := url.Parse(u); ue != nil {
		p.error("Malformed url '%s': %s", u, ue.Error())
	}
	return
}

// Check if file uploads are present with GET request.
func noGetWithFile(test *Test, p *Parser) {
	if test.Method != "GET" {
		return
	}

	// Check if files-uploads are present
	for k, list := range test.Param {
		for _, val := range list {
			if strings.HasPrefix(val, "@file:") {
				p.error("Cannot upload files with GET method in test %s, parameter %s.",
					test.Title, k)
			}
		}
	}
}

// Parse the suite.
func (p *Parser) ReadSuite() (suite *Suite, err error) {
	p.readLines()

	var test *Test
	suite = NewSuite()
	var first bool = true

	for p.i = 0; p.i < len(p.line); p.i++ {
		line := p.line[p.i]
		if isComment(line) || len(trim(line)) == 0 {
			continue
		}

		// start of test
		if hp(line, "---------") {
			if test != nil {
				// store last test read
				if first && test.Title == "Global" {
					suite.Global = test
				} else {
					noGetWithFile(test, p)
					suite.Test = append(suite.Test, *test)
					tracef("Append test to suite: \n%s", test.String())
					test = nil
				}
				first = false
			}
			if p.i+3 >= len(p.line) {
				p.error("Not enough lines left for valid test.")
				break
			}
			p.i++
			line = trim(p.line[p.i])
			if len(line) == 0 {
				p.error("No Title found")
				continue
			}
			test = NewTest(line)
			p.i++
			line = p.line[p.i]
			if !hp(line, "---------") {
				errorf("Title lower border missing")
				continue
			}
			continue
		}

		if hp(line, "\t") || hp(line, " ") {
			p.error("Misplaced indented stuff '%s'.", line)
			continue
		}

		line = trim(line)

		if hp(line, "GET") || hp(line, "POST") {
			test.Method, test.Url = p.readGetPost(line)
			continue
		}

		switch line {
		case "HEADER":
			p.readMap(&test.Header)
		case "SEND-COOKIE", "SEND-COOKIES", "COOKIE", "COOKIES":
			p.readSendCookies(test.Jar, "{CURRENT}")
		case "RESPONSE":
			test.RespCond = p.readCond(mode_response)
		case "SET-COOKIE", "RECIEVED-COOKIE":
			test.CookieCond = p.readCookieCond("{CURRENT}")
		case "BODY":
			test.BodyCond = p.readCond(mode_body)
		case "PARAM", "PARAMS", "PARAMETER", "PARAMETERS":
			p.readMultiMap(&test.Param)
		case "SETTING", "SETTINGS":
			p.readSettingMap(&test.Setting)
		case "CONST":
			p.readMap(&test.Const)
		case "RAND", "RANDOM":
			p.readMultiMap(&test.Rand)
		case "SEQ", "SEQUENCE":
			p.readMultiMap(&test.Seq)
		case "TAG", "TAGS":
			test.Tag = p.readTagCond()
		case "LOG", "LOGS":
			test.Log = p.readLogCond()
		case "BEFORE":
			test.Before = p.readShellCond()
		case "AFTER":
			test.After = p.readShellCond()
		case "VALIDATION", "VALIDATE":
			test.Validation = p.readValidation()
		default:
			if hp(line, "-") {
				p.error("Unknown stuff '%s'. Maybe to short test-title border?", line)
			} else {
				if test == nil {
					p.error("No test declared jet on '%s'", line)
				} else {
					p.error("Unknown section '%s'.", line)
				}
			}
			continue
		}

	}

	if test != nil {
		noGetWithFile(test, p)
		suite.Test = append(suite.Test, *test)
		tracef("Append test to suite: \n%s", test.String())

	}

	if !p.okay() {
		err = ParserError{strings.Join(p.errors, "\n")}
	}
	return
}
