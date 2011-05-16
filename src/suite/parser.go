package suite

import (
	"strings"
	"os"
	"io"
	"fmt"
	linereader "encoding/line"
	"strconv"
	"dobler/webtest/tag"
)

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
	reader *linereader.Reader
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
	parser.reader = linereader.NewReader(r, 4000)
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
	str = strings.Replace(str, "\\\"", "\"", -1)
	str = strings.Replace(str, "\\n", "\n", -1)
	str = strings.Replace(str, "\\t", "\t", -1)

	return str
}

func dequote(str string) string {
	if hp(str, "\"") && hs(str, "\"") {
		str = str[1 : len(str)-1]
		return deescape(str)
	}
	return str
}

// Return index of first space/tab in s or -1 if none found.
func firstSpace(s string) int {
	si := strings.Index(s, " ")
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
		p.i++
		line, no := p.line[p.i].line, p.line[p.i].no
		if !hp(line, "\t") {
			p.i--
			return
		}
		line = trim(line)
		j := firstSpace(line)
		var k, v string
		if j == -1 {
			k = line
		} else {
			k = trim(line[:j])
			v = trim(line[j:])
		}
		(*m)[k] = v
		trace("Added to map (line %d): %s: %s", no, k, v)
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
//		"mouse"
//		shark
func StringList(line string) (list []string) {
	all := strings.Fields(line)

	for i := 0; i < len(all); i++ {
		if hp(all[i], "\"") {
			s := all[i]
			i++
			for ; i < len(all) && !(hs(all[i], "\"") && !hs(all[i], "\\\"")); i++ {
				s += " " + all[i]
			}
			if i < len(all) {
				s += " " + all[i]
			}
			list = append(list, dequote(s))
		} else {
			list = append(list, all[i])
		}
	}
	return
}

// Like readMap, but treat value as list of strings
func (p *Parser) readMultiMap(m *map[string][]string) {
	for p.i < len(p.line)-1 {
		p.i++
		line, no := p.line[p.i].line, p.line[p.i].no
		if !hp(line, "\t") {
			p.i--
			return
		}
		line = trim(line)
		j := firstSpace(line)
		var k string
		var list []string
		if j == -1 {
			k = line
			list = []string{}
		} else {
			k = trim(line[:j])
			line = trim(line[j:])
			list = StringList(line)
		}
		(*m)[k] = list
		trace("Added to mulit map (line %d): >>>%s<<<: %v", no, k, list)
	}
}

// Read a Header or Body Condition
func (p *Parser) readCond(body bool) []Condition {
	var list []Condition = make([]Condition, 0, 3)

	for p.i < len(p.line)-1 {
		p.i++
		line, no := p.line[p.i].line, p.line[p.i].no
		if !hp(line, "\t") {
			p.i--
			return list
		}
		line = trim(line)
		j := firstSpace(line)
		if j == -1 {
			error("No op or value on line %d", no)
			p.okay = false
			continue
		}

		var neg bool
		k := trim(line[:j])
		if hp(k, "!") {
			neg = true
			k = k[1:]
		}
		line = trim(line[j:])
		if body { // only some are allowed
			switch k {
			case "Txt", "Bin":
			default:
				error("No such condition type '%s' for body on line %d.", k, no)
				p.okay = false
				continue
			}
		}
		j = firstSpace(line)
		if j == -1 {
			error("No value on line %d (in %s) or missing operator", no, trim(p.line[p.i].line))
			p.okay = false
			continue
		}
		op := trim(line[:j])
		v := trim(line[j:])
		if k == "Bin" {
			v = strings.ToLower(v) // our internal bin-values are lowercase
		}
		cond := Condition{Key: k, Op: op, Val: v, Neg: neg, Id: fmt.Sprintf("%s:%d", p.name, no)}
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
		error("Missing space after %s in line %d", beg, no)
		err = ParserError{"Missing space"}
		return
	}
	n, err = strconv.Atoi(line[:i])
	if err != nil {
		return
	}
	spec = line[i+1:]
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
				trace("Spec now: '%s'", strings.Replace(strings.Replace(spec, "\n", "\\n", -1), "\t", "  ", -1))
			}
			//fmt.Printf("\n-------------------\n%s\n----------------------\n", spec)
		}

		ts := tag.ParseTagSpec(spec)
		if ts != nil {
			cond.Spec = *ts
			list = append(list, cond)
			trace("Added to tag condition (line %d): %s", no, cond.String())
		} else {
			error("Problems parsing '%s'.", spec)
			p.okay = false
		}
	}
	return list
}


func (p *Parser) checkSettings(settings *map[string]string, lineid string) {
	for k, v := range *settings {
		switch k {
		case "Repeat":
			i, e := strconv.Atoi(v)
			if e != nil {
				error("No in value given as Repeat count (was '%s') on line %s.", v, lineid)
				p.okay = false
			} else if i > 100 {
				warn("More than 100 repetitions of one test on line %s.", lineid)
			}
		case "Max-Time":
			_, e := strconv.Atoi(v)
			if e != nil {
				error("No in value given as Max-Time miliseconds (was '%s') on line %s.", v, lineid)
				p.okay = false
			}
		case "Sleep":
			i, e := strconv.Atoi(v)
			if e != nil {
				error("No in value given as Sleep miliseconds (was '%s') on line %s.", v, lineid)
				p.okay = false
			} else if i < 0 {
				error("Sleep is < 0 on line %s.", lineid)
				p.okay = false
			}
		case "Keep-Cookies":
			switch v {
			case "true", "1", "True", "TRUE", "keep", "Keep":
				(*settings)[k] = "1"
			case "false", "0", "False", "FALSE", "drop", "DROP":
				(*settings)[k] = "0"
			default:
				error("Unknown value for Keep-Cookies: must be 0 or 1 (was '%s') on line %s.", v, lineid)
				p.okay = false
			}
		default:
			warn("Unknown Setting '%s' (check spelling and capitalization).", k)
		}
	}
}


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
			if hp(line, "GET ") {
				test.Method, test.Url = "GET", trim(line[3:])
				continue
			} else if hp(line, "POST ") {
				test.Method, test.Url = "POST", trim(line[4:])
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
			p.readMap(&test.Cookie)
		case "RESPONSE":
			test.RespCond = p.readCond(false)
		case "SET-COOKIE", "RECIEVED-COOKIE":
			test.CookieCond = p.readCond(false)
		case "BODY":
			test.BodyCond = p.readCond(true)
		case "PARAM", "PARAMETERS":
			p.readMultiMap(&test.Param)
		case "SETTING", "SETTINGS":
			p.readMap(&test.Setting)
			p.checkSettings(&test.Setting, fmt.Sprintf("%d", no))
		case "CONST":
			p.readMap(&test.Const)
		case "RAND":
			p.readMultiMap(&test.Rand)
		case "SEQ":
			p.readMultiMap(&test.Seq)
		case "TAG", "TAGS":
			test.Tag = p.readTagCond()
		default:
			error("Unknow section '%s' in line %d. Skipped.", line, no)
			err = ParserError{"Unknown Section"}
			return
		}

	}

	if test != nil {
		suite.Test = append(suite.Test, *test)
		trace("Append test to suite: \n%s", test.String())
		trace("len(suite.Test) == %d", len(suite.Test))
	}

	if !p.okay {
		err = ParserError{"General problems."}
	}
	return
}
