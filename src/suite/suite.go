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
	Global *Test
	Test   []Test
	bgload int
}

func NewSuite() (suite *Suite) {
	suite = new(Suite)
	suite.Test = make([]Test, 0, 5)
	suite.Global = nil
	suite.bgload = 0
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


func (s *Suite) stresstest(bg *Suite) (okay bool) {
	n := s.bgload
	if n > 0 {
		// start bg load
	}
	
	var totaltime int
	var totalfail int
	for _, t := range s.Test {
		tc := t.Copy()
		duration, err := tc.RunSingle(s.Global)
		if err != nil {
			okay = false
		}
		_, _, failed := tc.Stat()
		totaltime += duration
		totalfail += failed
	}

	info("Load %d: Average response time %d [ms]. Failures %d\n", n, totaltime/len(s.Test), totalfail)
	
	if n > 0 {
		// stop bg load
	}
	
	// decide if still okay
	return
}


func (s *Suite) Stress(bg *Suite, factor float32, step int) {
	for {
		okay := s.stresstest(bg)
		if ! okay {
			break
		}
		inc := int((1-factor) * float32(s.bgload))
		if inc <= 0 {
			inc = 1
		}
		if step > 0 && step < inc {
			inc = step
		}
		s.bgload += inc
	}

}
