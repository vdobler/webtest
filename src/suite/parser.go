package suite

import (
	"strings"
	"os"
	"io"
	linereader "encoding/line"
	// "bytes"
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
}

func NewParser(r io.Reader) *Parser {
	parser := new(Parser)
	parser.reader = linereader.NewReader(r, 4000)
	parser.line = []Line{}
	parser.suite = []Test{}
	return parser
}


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


func hp(s, p string) bool {
	return strings.HasPrefix(s, p)
}

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
			case "Txt", "Bin", "Tag":
			default:
				error("No such condition type '%s' for body.", k)
				continue
			}

			// Handle Tag
			if k == "Tag" {
				cond := Condition{Key: k, Val: line, Neg: neg, Line: no}
				list = append(list, cond)
				trace("Added to condition (line %d): %s", no, cond.String())
				continue
			}
		}
		j = firstSpace(line)
		if j == -1 {
			error("No value on line %d", no)
			continue
		}
		op := trim(line[:j])
		v := trim(line[j:])
		if k == "Bin" {
			v = strings.ToLower(v) // our internal bin-values are lowercase
		}
		cond := Condition{Key: k, Op: op, Val: v, Neg: neg, Line: no}
		list = append(list, cond)
		trace("Added to condition (line %d): %s", no, cond.String())
	}
	return list
}

func (p *Parser) ReadSuite() (suite *Suite, err os.Error) {
	p.readLines()

	var test *Test
	suite = NewSuite()

	for p.i = 0; p.i < len(p.line); p.i++ {
		line, no := p.line[p.i].line, p.line[p.i].no

		// sart of test
		if hp(line, "---------") {
			if test != nil {
				suite.Test = append(suite.Test, *test)
				trace("Append test to suite: \n%s", test.String())
				test = nil
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
				test.Method, test.Url = "POST", trim(line[3:])
				continue
			} else {
				error("Method and Url missing or wrong in line %d", no)
				err = ParserError{"Method and Url missing or wrong"}
				return
			}
		}

		if hp(line, "\t") || hp(line, " ") {
			error("Misplaced indented stuff in line %d", no)
			err = ParserError{"Mispalced indented stuff"}
			return
		}

		line = trim(line)
		switch line {
		case "HEADER":
			p.readMap(&test.Header)
		case "RESPONSE":
			test.RespCond = p.readCond(false)
		case "BODY":
			test.BodyCond = p.readCond(true)
		case "PARAM", "PARAMETERS":
			p.readMultiMap(&test.Param)
		case "SETTING", "SETTINGS":
			p.readMap(&test.Setting)
		case "CONST":
			p.readMap(&test.Const)
		case "RAND":
			p.readMultiMap(&test.Rand)
		case "SEQ":
			p.readMultiMap(&test.Seq)
		default:
			error("Unknow element '%s' in line %d. Skipped.", line, no)
		}

	}

	if test != nil {
		suite.Test = append(suite.Test, *test)
		trace("Append test to suite: \n%s", test.String())
		trace("len(suite.Test) == %d", len(suite.Test))
	}
	return
}
