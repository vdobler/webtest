package suite

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Log level of suite. 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace, 6:supertrace
var LogLevel int = 2


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

func supertrace(f string, m ...interface{}) {
	if LogLevel >= 6 {
		logger.Print("*SUPER* " + fmt.Sprintf(f, m...))
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
		error("No such test.")
		return
	}
	trace("Running test no %d (global = %p)", n, s.Global)
	s.Test[n].Run(s.Global)
}


func (s *Suite) BenchTest(n, count int) (dur []int, f int, err os.Error) {
	if n < 0 || n >= len(s.Test) {
		error("No such test")
	}

	dur, f, err = s.Test[n].Bench(s.Global, count)
	return
}


// Run the request (while skipping tests) of test and reply on done when finished.
func bgRun(test, global *Test, done chan bool) {
	trace("Started background test %s", test.Title)
	test.RunWithoutTest(global)
	done <- true
	trace("Finished background test %s.", test.Title)
}

// Make n request of bg suite in parallel until shot down via kill.
func bgnoise(n int, bg *Suite, kill chan bool) {
	m := len(bg.Test)
	var done chan bool
	done = make(chan bool)
	var i int = 0
	debug("Initializing bgnoise")
	for ; i < n; i++ {
		go bgRun(&(bg.Test[i%m]), bg.Global, done)
	}

	var killed bool
	for {
		select {
		case killed = <-kill:
			debug("bgnoise killed")
		case _ = <-done:
			// trace("bg process done")
			if !killed {
				i++
				// trace("Next bg test %s.", bg.Test[i % m].Title)
				go bgRun(&(bg.Test[i%m]), bg.Global, done)
			} else {
				n--
				// trace("Left running %d:", n)
				if n == 0 {
					// trace("begnois done all work: returning")
					return
				}
			}
		}
	}
}


type StressResult struct {
	Load  int
	N     int
	Err   int
	AvgRT int64
	MaxRT int64
	MinRT int64
	Total int
	Pass  int
	Fail  int
}

func (s *Suite) Stresstest(bg *Suite, load, reps int, rampSleep int64) (result StressResult) {
	var kill chan bool
	kill = make(chan bool)
	if load > 0 {
		go bgnoise(load, bg, kill)
		// start bg load
	}

	result.MaxRT = -9999999999999
	result.MinRT = 100000000000
	result.Load = load

	for rep := 1; rep <= reps; rep++ {
		info("Repetition %d of %d of test suite:", rep, reps)
		for _, t := range s.Test {
			if t.Repeat() == 0 {
				info("Test no '%s' is disabled.", t.Title)
				continue
			}
			time.Sleep(rampSleep * 1000000)
			tc := t.Copy()
			duration, err := tc.RunSingle(s.Global, false)
			if err != nil {
				result.Err++
			}
			rt := int64(duration)
			total, passed, failed := tc.Stat()

			result.N++
			if rt < result.MinRT {
				result.MinRT = rt
			}
			if rt > result.MaxRT {
				result.MaxRT = rt
			}
			result.AvgRT += rt
			result.Pass += passed
			result.Total += total
			result.Fail += failed
		}
	}
	result.AvgRT /= int64(result.N)

	debug("Load %d: Response Time %d / %d (avg/max). Status %d / %d / %d (err/pass/fail). %d / %d (tests/checks).",
		load, result.AvgRT, result.MaxRT, result.Err, result.Pass, result.Fail, result.N, result.Total)

	if load > 0 {
		kill <- true
	}

	time.Sleep(rampSleep * 1000000)
	return
}


type Stepper interface {
	Next(current int) int
}

type ConstantStep struct {
	Start int
	Step  int
}

func (cs ConstantStep) Next(current int) int {
	if current == 0 {
		return cs.Start
	}
	return current + cs.Step
}

type FactorStep struct {
	Start  int
	Factor float32
}

func (fs FactorStep) Next(current int) int {
	if current == 0 {
		return fs.Start
	}
	n := int((fs.Factor - 1) * float32(current))
	if n <= current {
		n = current + 1
	}
	return n
}
