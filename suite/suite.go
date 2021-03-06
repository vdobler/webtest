// Copyright 2011 Volker Dobler. All rights reserved.
// See LICENSE file in the webtest directory.

//
// Suite - Main routines for the Webtest utility.
//
package suite

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"time"
)

// Log level of suite. 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace, 6:supertrace
var LogLevel int = 2

// Path of directory to dump stuff to
var OutputPath = "."

var logger *log.Logger

func init() {
	logger = log.New(os.Stderr, "Suite   ", log.Ldate|log.Ltime)
}

func errorf(f string, m ...interface{}) {
	if LogLevel >= 1 {
		logger.Print("*ERROR* " + fmt.Sprintf(f, m...))
	}
}
func warnf(f string, m ...interface{}) {
	if LogLevel >= 2 {
		logger.Print("*WARN * " + fmt.Sprintf(f, m...))
	}
}
func infof(f string, m ...interface{}) {
	if LogLevel >= 3 {
		logger.Print("*INFO * " + fmt.Sprintf(f, m...))
	}
}
func debugf(f string, m ...interface{}) {
	if LogLevel >= 4 {
		logger.Print("*DEBUG* " + fmt.Sprintf(f, m...))
	}
}

func tracef(f string, m ...interface{}) {
	if LogLevel >= 5 {
		logger.Print("*TRACE* " + fmt.Sprintf(f, m...))
	}
}

func supertracef(f string, m ...interface{}) {
	if LogLevel >= 6 {
		logger.Print("*SUPER* " + fmt.Sprintf(f, m...))
	}
}

// Suite is a collection of test.
type Suite struct {
	Global *Test  // the gloabl test-template and cookie jar
	Test   []Test // list of all tests
	Name   string // the name of this suite
	bgload int    // the background load
}

// NewSuite sets up an empty Suite.
func NewSuite() (suite *Suite) {
	suite = new(Suite)
	suite.Test = make([]Test, 0, 5)
	suite.Global = nil
	suite.bgload = 0
	return
}

// RunTest will execute test number n in the list of tests.
// The results if the checks performed are reported in the test.
func (s *Suite) RunTest(n int) {
	if n < 0 || n >= len(s.Test) {
		errorf("No such test.")
		return
	}
	tracef("Running test no %d (global = %p)", n, s.Global)
	s.Test[n].Run(s.Global)
}

// BenchTest will run test number n for count many times.
// Returned are the list durations (in ms)
func (s *Suite) BenchTest(n, count int) (dur []int, f int, err error) {
	if n < 0 || n >= len(s.Test) {
		errorf("No such test")
		err = errors.New("No such test")
		return
	}

	dur, f, err = s.Test[n].Bench(s.Global, count)
	return
}

// Run the request in test (do not perform checks) and reply on channel done when finished.
func bgRun(test, global *Test, done chan bool) {
	tracef("Started background test %s", test.Title)
	test.RunWithoutTest(global)
	done <- true
	tracef("Finished background test %s.", test.Title)
}

// Make n request of bg suite in parallel until shut down via signal on channel kill.
func bgnoise(n int, bg *Suite, kill chan bool) {
	m := len(bg.Test)
	var done chan bool
	done = make(chan bool)
	var i int = 0
	debugf("Initializing bgnoise")
	for ; i < n; i++ {
		go bgRun(&(bg.Test[i%m]), bg.Global, done)
	}

	var killed bool
	for {
		select {
		case killed = <-kill:
			debugf("Killed bgnoise")
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
					debugf("Finished begnois")
					return
				}
			}
		}
	}
}

// Structure to collect results from a stresstest run.
type StressResult struct {
	Load   int   // number of parallel background requests
	N      int   // total number of tests and repetitions
	Pass   int   // number of passed tests
	Fail   int   // number of failed tests
	Err    int   // number errors (e.g. unable to connect)
	AvgRT  int64 // average response time in ms
	MaxRT  int64 // maximum response time in ms
	MinRT  int64 // minimum response time in ms
	Total  int   // total number of tests performed
	RT     []int
	Detail map[string][]int // maps Test.Title to response times in ms.
}

// Perform reps runs of s while running load of load parallel background request taken from bg.
func (s *Suite) Stresstest(bg *Suite, load, reps int, rampSleep int64) (result StressResult) {
	var kill chan bool
	kill = make(chan bool)
	if load > 0 {
		go bgnoise(load, bg, kill)
		// start bg load
	}

	result.MaxRT = math.MinInt64
	result.MinRT = math.MaxInt64
	result.Load = load
	result.Detail = make(map[string][]int)

	for rep := 1; rep <= reps; rep++ {
		infof("Repetition %d of %d of test suite:", rep, reps)
		for _, t := range s.Test {
			if t.Repeat() == 0 {
				infof("Test no '%s' is disabled.", t.Title)
				continue
			}
			time.Sleep(time.Duration(rampSleep) * time.Millisecond)
			tc := t.Copy()
			runtime.GC()
			duration, _, _ := tc.RunSingle(s.Global, false)

			rt := int64(duration)
			result.RT = append(result.RT, int(rt))
			result.Detail[t.Title] = append(result.Detail[t.Title], int(rt))
			passed, failed, errored := tc.Stat()
			total := passed + failed

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
			result.Err += errored
		}
	}
	if result.N != 0 {
		result.AvgRT /= int64(result.N)
	}
	debugf("Load %d: Response Time %d / %d (avg/max). Status %d / %d / %d (err/pass/fail). %d / %d (tests/checks).",
		load, result.AvgRT, result.MaxRT, result.Err, result.Pass, result.Fail, result.N, result.Total)

	if load > 0 {
		kill <- true
	}

	time.Sleep(time.Duration(rampSleep) * time.Millisecond)
	return
}

// Stepper is a load-increaser which yields the next number of background tasks.
type Stepper interface {
	Next(current int) int
}

// Implements Stepper and increases at a constant rate (linearely).
type ConstantStep struct {
	Start int
	Step  int
}

// Next yields current + step.
func (cs ConstantStep) Next(current int) int {
	if current == 0 {
		return cs.Start
	}
	return current + cs.Step
}

// FactorStep implements Stepper and increases at a constant factor (exponential).
type FactorStep struct {
	Start  int
	Factor float32
}

// Next yields current * factor.
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
