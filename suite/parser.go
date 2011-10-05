package suite

import (
	"bufio"
	"fmt"
	"http"
	"io"
	"os"
	"strconv"
	"strings"
	"url"

	"github.com/vdobler/webtest/tag"
)

var DefaultSettings = map[string]int{"Repeat": 1,
	"Tries":        1,
	"Max-Time":     -1,
	"Sleep":        0,
	"Keep-Cookies": 0,
	"Abort":        0,
	"Dump":         0,
	"Validate":     0,
}

type ParserError struct {
	cause string
}

func (pe ParserError) String() string {
	return pe.cause
}

type Line struct {
	line string
	no   int
}

type Parser struct {
	reader *bufio.Reader
	line   []Line
	test   *Test
	suite  []Test
	i      int
	name   string
	okay   bool
}

// Set up a new Parser which reads a suite from r (named name).
func NewParser(r io.Reader, name string) *Parser {
	parser := new(Parser)
	parser.reader = bufio.NewReader(r)
	parser.line = []Line{}
	parser.suite = []Test{}
	parser.name = name
	parser.okay = true
	return parser
}

// Read a line from the Reader
func (p *Parser) nextLine() (line Line, err os.Error) {
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
	// trace("NextLine %d: %s", p.i, str)
	return Line{str, p.i}, nil
}

// Return the next non-blank, non-comment line.
func (p *Parser) nextRealLine() (line Line, err os.Error) {
	for {
		line, err = p.nextLine()
		if err != nil {
			return
		}
		if len(trim(line.line)) > 0 && !hp(trim(line.line), "#") {
			break
		}
	}
	// trace("NextRealLine %d: %s", line.no, line.line)
	return
}

// Fill list of lines.
func (p *Parser) readLines() {
	p.i = 0
	for {
		line, err := p.nextRealLine()
		if err != nil {
			return
		}
		p.line = append(p.line, line)
		trace("%-3d: %s", line.no, line.line)
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

func dequote(str string) (string, os.Error) {
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
	si := strings.Index(s, " ") // TODO: use IndexAny?
	ti := strings.Index(s, "\t")
	if si == -1 && ti == -1 {
		return -1
	} else if si == -1 {
		return ti
	} else if ti == -1 {
		return si
	} else if si < ti {
		return si
	}
	return ti
}

// Read a string->string map. Stopp if unindented line is found
func (p *Parser) readMap(m *map[string]string) {
	for p.i < len(p.line)-1 {
		done, _, key, _, val := p.nextStuff([]string{":="})
		if done {
			return
		}
		if s, err := dequote(val); err != nil {
			error("Malformed string '%s' in line %d.", val, p.i)
			p.okay = false
			continue
		} else {
			val = s
		}
		(*m)[key] = val
		trace("Added to map (line %d): %s: %s", p.i, key, val)
	}
}

// parse smth like  "name:domain:path:Secure ~= value", line must be trimmed
func parseCookie(key, host string) (name, domain, path, field string, err os.Error) {
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
		err = os.NewError("Too many ':' in cookie definition.")
		return
	}

	// Add defaults of path and domain
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		err = os.NewError(fmt.Sprintf("Illegal path '%s' in cookie.", path))
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
			error("%s on line %d.", err.String(), p.i)
			p.okay = false
			continue
		}
		switch field {
		case "":
			field = "value"
		case "secure", "httponly", "maxage", "expires", "delete", "deleted", "value":
		default:
			error("Unknown cookie field '%s' on line %d.", field, p.i)
			p.okay = false
			continue
		}

		cond.Key = fmt.Sprintf("%s:%s:%s:%s", name, domain, path, field)
		cond.Op = op
		cond.Val = value
		cond.Id = fmt.Sprintf("%s:d", p.name, p.i)
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
			error("%s on line %d.", err.String(), p.i)
			p.okay = false
			continue
		}
		if field != "" && field != "secure" {
			error("Wrong field '%s' (only secure allowed) on line %d.", field, p.i)
			p.okay = false
			continue
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
			error("Unknown settign '%s' in line %d.", key, p.i)
			p.okay = false
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
		case "true", "yes", "ja", "qui", "create", "new", "keep", "abort", "link", "links":
			n = 1
		case "append", "html", "xhtml":
			n = 2
		case "both", "links+html", "html+links", "body":
			n = 3
		default:
			var err os.Error
			n, err = strconv.Atoi(val)
			if err != nil {
				error("Cannot convert %s to integer on line %d.", val, p.i)
				p.okay = false
			}
		}

		// Safeuard against stupid or wrong settings.
		switch key {
		case "Repeat":
			if n > 100 {
				warn("More then 100 repetitions on line %d.", p.i)
			}
		case "Tries":
			if n <= 0 {
				warn("Setting Tries to value <= 0 is unsensical line %d.", p.i)
			}
		case "Keep-Cookies", "Abort":
			if n != 0 && n != 1 {
				warn("Keep-Cookies and Abort accept only 0 and 1 as value on line %d.", p.i)
			}
		case "Dump":
			if n < 0 || n > 3 {
				warn("Dump accepts only 0, 1 and 2 as value (was %s=%d) on line %d.", val, n, p.i)
			}
		case "Validate":
			if n < 0 || n > 3 {
				warn("Validates accept only 0, 1, 2 and 3 as value (was %s=%d) on line %d.", val, n, p.i)
			}
		}
		(*m)[key] = n
		trace("Added to settings-map (line %d): %s: %s", p.i, key, val)
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
func StringList(line string) (list []string, err os.Error) {

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
		for len(line) > 0 && line[0] == ' ' {
			line = line[1:]
		} // TODO: inneficient
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
			var err os.Error
			list, err = StringList(val)
			if err != nil {
				error("Cannot decode '%s' on line %d: %s.", val, p.i, err.String())
				p.okay = false
			}
		}
		(*m)[key] = list
		trace("Added to mulit map (line %d): key: %v", p.i, key, list)
	}
}

// Parse strings like "[:10]" or "[50:-2]" into a Range.
func parseRange(s string) (r Range, err os.Error) {
	if s == "" {
		return
	}

	if !hp(s, "[") || !hs(s, "]") {
		err = os.NewError("Missing [ or ].")
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

		err = os.NewError("Missing or multiple :")
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
		line := p.line[p.i].line
		if !hp(line, "\t") {
			p.i--
			break
		}

		line = trim(line)
		args, err := StringList(line)
		if err != nil {
			error("Unable to parse command '%s' on line %d: %s", line, p.line[p.i].no, err.String())
			p.okay = false
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
		line, no := p.line[p.i].line, p.line[p.i].no
		fmt.Printf("line1=%s\n", line)
		if !hp(line, "\t") {
			p.i--
			done = true
			return
		}

		line = trim(line)
		if len(line) == 0 {
			continue
		}

		fmt.Printf("line2=%s\n", line)
		if validOps[0] != ":=" {
			if line[0] == '!' {
				line = trim(line[1:])
				neg = true
			}
		}

		fmt.Printf("line3=%s\n", line)
		j := firstSpace(line)
		if j == -1 {
			key = line
			return
		}

		key, line = line[:j], trim(line[j:])
		j = firstSpace(line)
		if j == -1 {
			error("Missing operator (%v) in line %d.", validOps, no)
			p.okay = false
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
			error("Illegal operator '%s' in line %d.", op, no)
			p.okay = false
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
		done, neg, key, op, val := p.nextStuff([]string{"==", "~=", "_=", "=_", "/=", ">", ">=", "<", "<="})
		fmt.Printf("key=%s   op=%s  val=%s\n", key, op, val)
		if done {
			return list
		}
		var rng Range

		if mode == mode_body {
			if val == "" {
				error("Missing value for body condition on line %d.", p.i)
				p.okay = false
				continue
			}
			if !(hp(key, "Txt") || hp(key, "Bin")) {
				error("No such condition type '%s' for body on line %d.", key, p.i)
				p.okay = false
				continue
			}
			rs := ""
			key, rs = key[:3], key[3:]

			// optional range
			if rs != "" {
				if rg, err := parseRange(rs); err != nil {
					error("Unable to parse range '%s' on line %d. %s", rs, p.i, err.String())
					p.okay = false
					continue
				} else {
					rng = rg
				}
			}

			if key == "Bin" {
				v := strings.ToLower(strings.Replace(val, " ", "", -1))
				if len(v)%2 == 1 {
					warn("Odd number of nibbles in binary value on line %d. Will discard last nibble.", p.i)
					v = v[:len(v)-2]
				}
				n := len(v) / 2
				var c byte
				for i := 0; i < n; i++ {
					r, err := fmt.Sscanf(v[2*i:2*i+2], "%x", &c)
					if err != nil || r != 1 {
						error("Cannot parse '%s' in hex string '%s' on line %d",
							v[2*i:2*i+2], v, p.i)
						p.okay = false
						break
					}
				}
			}
		}

		if val, err := dequote(val); err != nil {
			error("Cannot parse string '%s' on line %d: %s", val, p.i, err)
			p.okay = false
			continue
		}

		id := fmt.Sprintf("%s:%d", p.name, p.i)
		cond := Condition{Key: key, Op: op, Val: val, Neg: neg, Id: id, Range: rng}
		fmt.Println(cond)
		list = append(list, cond)
		trace("Added to condition (line %d): %s", p.i, cond.String())
	}
	return list
}

// Read a Header or Body Condition
func (p *Parser) readLogCond() []LogCondition {
	var list []LogCondition = make([]LogCondition, 0, 3)

	for p.i < len(p.line)-1 {
		p.i++
		line, no := p.line[p.i].line, p.line[p.i].no
		if !hp(line, "\t") {
			p.i--
			return list
		}

		// Normal format is "[!] <field> <op> <value>", reduced format is just "[!] <field>"
		line = trim(line)
		var k, op, v string
		var neg bool
		if hp(line, "!") {
			neg = true
			line = trim(line[1:])
		}
		j := firstSpace(line)

		if j == -1 {
			error("Cannot read log condition on line %d.", no)
			p.okay = false
			continue
		}
		k = trim(line[:j])
		line = trim(line[j:])
		j = firstSpace(line)
		if j == -1 {
			error("Cannot read log condition on line %d.", no)
			p.okay = false
			continue
		}
		op = trim(line[:j])
		switch op {
		case "_=", "=_", "~=", "/=":
		default:
			error("Unknown operator '%s' in %s:%d.", op, p.name, no)
			p.okay = false
			continue
		}
		v = trim(line[j:])
		var err os.Error
		if v, err = dequote(v); err != nil {
			error("Cannot read string on line %d: %s", no, err.String())
			p.okay = false
			continue
		}
		cond := LogCondition{Path: k, Op: op, Val: v, Neg: neg, Id: fmt.Sprintf("%s:%d", p.name, no)}
		list = append(list, cond)
		trace("Added to condition (line %d): %s", no, cond.String())
	}
	return list
}

// Helper to extract count an spec from strings like ">= 5  a href=/index.html"
// off is the number of charactes to strip before trying to read an int.
func numStr(line string, off, no int) (n int, spec string, err os.Error) {
	trace("line = %s, off = %d", line, off)
	beg := line[:off]
	line = trim(line[off:])
	i := firstSpace(line)
	if i < 0 {
		i = strings.Index(line, "[")
		if i < 0 {
			cause := fmt.Sprintf("Missing space after %s in line %d", beg, no)
			error(cause)
			err = ParserError{cause}
			return
		}
	}
	n, err = strconv.Atoi(line[:i])
	if err != nil {
		return
	}
	spec = trim(line[i:])
	trace("n=%d, spec=%s", n, spec)
	return
}

// Reads the following tag conditions (like readMap)
func (p *Parser) readTagCond() []TagCondition {
	var list []TagCondition = make([]TagCondition, 0, 3)

	for p.i < len(p.line)-1 {
		p.i++
		line, no := p.line[p.i].line, p.line[p.i].no
		if !hp(line, "\t") {
			p.i--
			return list
		}
		line = trim(line)
		if len(line) == 0 {
			continue
		}
		for hp(line, "! ") { //  transform "!  =3 a href == xyz" to "!=3 a href == xyz"
			line = "!" + line[2:]
		}

		cond := TagCondition{}
		cond.Id = fmt.Sprintf("%s:%d", p.name, no)
		var spec string
		var err os.Error

		if false {
		} else if hp(line, "!=") {
			cond.Cond = CountNotEqual
			cond.Count, spec, err = numStr(line, 2, no)
		} else if hp(line, "!>=") {
			cond.Cond = CountLess
			cond.Count, spec, err = numStr(line, 3, no)
		} else if hp(line, "!>") {
			cond.Cond = CountLessEqual
			cond.Count, spec, err = numStr(line, 2, no)
		} else if hp(line, "!<=") {
			cond.Cond = CountGreater
			cond.Count, spec, err = numStr(line, 3, no)
		} else if hp(line, "!<") {
			cond.Cond = CountGreaterEqual
			cond.Count, spec, err = numStr(line, 2, no)
		} else if hp(line, "!") {
			cond.Cond = TagForbidden
			spec = line[1:]
		} else if hp(line, "==") {
			cond.Cond = CountEqual
			cond.Count, spec, err = numStr(line, 2, no)
		} else if hp(line, "=") {
			cond.Cond = CountEqual
			cond.Count, spec, err = numStr(line, 1, no)
		} else if hp(line, ">=") {
			cond.Cond = CountGreaterEqual
			cond.Count, spec, err = numStr(line, 2, no)
		} else if hp(line, ">") {
			cond.Cond = CountGreater
			cond.Count, spec, err = numStr(line, 1, no)
		} else if hp(line, "<=") {
			cond.Cond = CountLessEqual
			cond.Count, spec, err = numStr(line, 2, no)
		} else if hp(line, "<") {
			cond.Cond = CountLess
			cond.Count, spec, err = numStr(line, 1, no)
		} else {
			cond.Cond = TagExpected
			spec = line
		}
		if err != nil {
			p.okay = false
			return list
		}

		spec = trim(spec)
		if hp(spec, "[") { // multiline tag spec
			trace("Multiline tag spec")
			spec = ""
			for p.i < len(p.line)-1 {
				p.i++
				line, no := p.line[p.i].line, p.line[p.i].no
				trace("Next line: %s", line)
				if !hp(line, "\t") {
					error("Nonindented line in multiline tag spec on line %d", no)
					p.okay = false
					break
				}
				if hs(trim(line), "]") {
					trace("End of multiline tag spec found in line %d.", no)
					break
				}
				if spec == "" {
					line = trim(line)
				} else {
					spec += "\n"
				}
				spec += line
				supertrace("Spec now: '%#v'", spec)
			}
			// fmt.Printf("\n-------------------\n%s\n----------------------\n", spec)
		}

		if ts, err := tag.ParseTagSpec(spec); err == nil {
			cond.Spec = *ts
			list = append(list, cond)
			trace("Added to tag condition (line %d): %s", no, cond.String())
		} else {
			error("Problems parsing tagspec %#v on line %d: %s", spec, no, err.String())
			p.okay = false
		}
	}
	return list
}

const (
	mode_response = iota
	mode_body
)

// Parse the suite.
func (p *Parser) ReadSuite() (suite *Suite, err os.Error) {
	p.readLines()

	var test *Test
	suite = NewSuite()
	var first bool = true

	for p.i = 0; p.i < len(p.line); p.i++ {
		line, no := p.line[p.i].line, p.line[p.i].no

		// sart of test
		if hp(line, "---------") {
			if test != nil {
				if first && test.Title == "Global" {
					suite.Global = test
				} else {
					suite.Test = append(suite.Test, *test)
					trace("Append test to suite: \n%s", test.String())
					test = nil
				}
				first = false
			}
			p.i++
			line, no = trim(p.line[p.i].line), p.line[p.i].no
			if len(line) == 0 {
				error("No Title found in line %d", no)
				err = ParserError{"No tite found."}
				return
			}
			test = NewTest(line)
			p.i++
			line, no = p.line[p.i].line, p.line[p.i].no
			if !hp(line, "---------") {
				error("Title lower border missing in line %d", no)
				err = ParserError{"Title lower border missing."}
				return
			}
			p.i++
			line, no = trim(p.line[p.i].line), p.line[p.i].no
			// TODO: check giveb url on parsability
			if hp(line, "GET ") {
				urll := trim(line[3:])
				if i := strings.Index(urll, "#"); i != -1 {
					warn("URL may not contain fragment (#-part) in line %d.", no)
					urll = urll[:i]
				}
				test.Method, test.Url = "GET", urll
				if _, ue := url.Parse(urll); ue != nil {
					error("Malformed url '" + urll + "': " + ue.String())
					err = ParserError{"Malformed url '" + urll + "': " + ue.String()}
					return
				}
				continue
			} else if hp(line, "POST ") {
				test.Method, test.Url = "POST", trim(line[4:])
				continue
			} else if hp(line, "POST:mp ") {
				test.Method, test.Url = "POST:mp", trim(line[7:])
				continue
			} else {
				error("Method and Url missing or wrong in line %d", no)
				err = ParserError{"Method and Url missing or wrong"}
				return
			}
		}

		if hp(line, "\t") || hp(line, " ") {
			error("Misplaced indented stuff in line %d", no)
			err = ParserError{"Misplaced indented stuff"}
			return
		}

		line = trim(line)
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
		case "PARAM", "PARAMETERS":
			p.readMultiMap(&test.Param)
		case "SETTING", "SETTINGS":
			p.readSettingMap(&test.Setting)
		case "CONST":
			p.readMap(&test.Const)
		case "RAND":
			p.readMultiMap(&test.Rand)
		case "SEQ":
			p.readMultiMap(&test.Seq)
		case "TAG", "TAGS":
			test.Tag = p.readTagCond()
		case "LOG", "LOGS":
			test.Log = p.readLogCond()
		case "BEFORE":
			test.Before = p.readShellCond()
		case "AFTER":
			test.After = p.readShellCond()
		default:
			error("Unknow section '%s' in line %d. Skipped.", line, no)
			err = ParserError{"Unknown Section " + line}
			return
		}

	}

	if test != nil {
		if test.Method == "GET" {
			// Check if files-uploads are present
			for k, list := range test.Param {
				for _, val := range list {
					if strings.HasPrefix(val, "@file:") {
						error("Cannot upload files with GET method in test %s, parameter %s.", test.Title, k)
						p.okay = false
					}
				}
			}
		}
		suite.Test = append(suite.Test, *test)
		trace("Append test to suite: \n%s", test.String())
	}

	if !p.okay {
		err = ParserError{"General problems."}
	}
	return
}
