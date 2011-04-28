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
	no int
}

type Parser struct {
	reader *linereader.Reader
	line []Line
	test *Test
	suite []Test
	i int
}

func NewParser(r io.Reader) *Parser {
	parser := new(Parser)
	parser.reader = linereader.NewReader(r, 4000)
	parser.line = make([]Line, 50)
	parser.suite = make([]Test, 3)
	return parser
}


func (p *Parser) nextLine() (line Line, err os.Error) {
	var isprefix bool 
	var by []byte
	var str string
	by, isprefix, err = p.reader.ReadLine()
	if err != nil { return }
	for isprefix {
		str += string(by)
		by, isprefix, err = p.reader.ReadLine()
		if err != nil { return }
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
		if err != nil { return }
		p.line = append(p.line, line)
		trace("%-3d: %s", line.no, line.line)
	}
}


func hp(s, p string) bool {
	return strings.HasPrefix(s, p)
}

func trim(s string) string {
	return strings.Trim(s, " \t")
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
		j := strings.Index(line, " ")
		if j == -1 {
			error("No value on line %d", no)
		} else {
			k := trim(line[:j])
			v := trim(line[j:])
			(*m)[k] = v
			trace("Added to map (line %d): %s: %s", no, k, v)
		}
	}
}

func (p *Parser) readCond(body bool) ([]Condition) {
	var list []Condition = make([]Condition, 0, 3)

	for p.i < len(p.line)-1 {
		p.i++
		line, no := p.line[p.i].line, p.line[p.i].no
		if !hp(line, "\t") {
			p.i--
			return list
		}
		line = trim(line)
		j := strings.Index(line, " ")
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
				cond := Condition{Key: k, Val: line, Neg: neg}
				list = append(list, cond)
				trace("Added to condition (line %d): %s", no, cond.String())
				continue
			}
		} 
		j = strings.Index(line, " ")
		if j == -1 {
			error("No value on line %d", no)
			continue
		}
		op := trim(line[:j])
		v := trim(line[j:])
		cond := Condition{Key: k, Op: op, Val: v, Neg: neg}
		list = append(list, cond)
		trace("Added to condition (line %d): %s", no, cond.String())
	}
	return list
}

func (p *Parser) ReadSuite() (suite *Suite, err os.Error) {
	p.readLines()
	
	var test *Test
	suite = NewSuite()
	
	for p.i=0; p.i<len(p.line); p.i++ {
		line, no := p.line[p.i].line, p.line[p.i].no
		
		// sart of test
		if hp(line,"---------") {
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
			if !hp(line,"---------") {
				error("Title lower border missing in line %d", no)
				err = ParserError{"Title lower border missing."}
				return
			}
			p.i++
			line, no = trim(p.line[p.i].line), p.line[p.i].no
			if hp(line, "GET") {
				test.Method, test.Url = "GET", trim(line[3:])
			} else if hp(line, "GET") {
				test.Method, test.Url = "GET", trim(line[3:])
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
			p.readMap( &test.Header )
		case "RESPONSE":
			 test.RespCond = p.readCond( false )
		case "BODY":
			test.BodyCond = p.readCond( true )
		case "PARAM":
			p.readMap( &test.Param )
		case "SETTING":
			p.readMap( &test.Setting )
		case "CONST":
			p.readMap( &test.Const )
		}
		
	}
	
	if test != nil {
		suite.Test = append(suite.Test, *test)
		trace("Append test to suite: \n%s", test.String())
		trace("len(suite.Test) == %d", len(suite.Test))
	}
	return
}
