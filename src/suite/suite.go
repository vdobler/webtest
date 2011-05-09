package suite

import (
	"fmt"
	"log"
	"os"
)

var LogLevel int = 3 // 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace
var logger *log.Logger

func init() {
	logger = log.New(os.Stderr, "Suite   ", log.Ldate|log.Ltime)
}

func error(f string, m ...interface{}) {
	if LogLevel >= 1 {
		logger.Print("*ERROR* " + fmt.Sprintf(f, m...))
	}
}
func warn(f string, m ...interface{}) {
	if LogLevel >= 2 {
		logger.Print("*WARN * " + fmt.Sprintf(f, m...))
	}
}
func info(f string, m ...interface{}) {
	if LogLevel >= 3 {
		logger.Print("*INFO * " + fmt.Sprintf(f, m...))
	}
}
func debug(f string, m ...interface{}) {
	if LogLevel >= 4 {
		logger.Print("*DEBUG* " + fmt.Sprintf(f, m...))
	}
}
func trace(f string, m ...interface{}) {
	if LogLevel >= 5 {
		logger.Print("*TRACE* " + fmt.Sprintf(f, m...))
	}
}


// TODO: Results?
type Suite struct {
	Test []Test
	// Result map[string]int // 0: not run jet, 1: pass, 2: fail, 3: err 
}

func NewSuite() (suite *Suite) {
	suite = new(Suite)
	suite.Test = make([]Test, 0, 5)
	return
}

func (s *Suite) RunTest(n int) {
	if n < 0 || n >= len(s.Test) {
		error("No such test")
	}

	var global *Test
	if n > 0 && s.Test[0].Title == "Global" {
		global = &s.Test[0]
	}

	s.Test[n].Run(global)

}


func (s *Suite) BenchTest(n, count int) (dur []int, f int, err os.Error) {
	if n < 0 || n >= len(s.Test) {
		error("No such test")
	}

	var global *Test
	if n > 0 && s.Test[0].Title == "Global" {
		global = &s.Test[0]
	}

	dur, f, err = s.Test[n].Bench(global, count)
	return
}
