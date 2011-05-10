package suite

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Log level of suite. 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace, 6:supertrace
var LogLevel int = 3 

// Time in ms to sleep between runs with different background loads during stresstests
var SleepMs1 int64 = 1000  

// Allowed factor increase of average response times from one run to the next in stresstest
var AllowedSingleRtFactor int = 5

// Allowed general factor increase of average response times from no background request in stresstest
var AllowedGeneralRtFactor int = 20

// Allowed maximum response time in ms.
var AllowedMaximumRt int = 30 * 1000

// Maximum allowed fraction of tests to fail
var AllowedFailingFraction float32 = 0.2

// Maximum allowed number of background request
var AllowedMaximumBgReq int = 250


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

	s.Test[n].Run(s.Global, false)
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
	test.Run(global, true)
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
	for ; i<n; i++ {
		go bgRun(&(bg.Test[i % m]), bg.Global, done)
	}

	var killed bool
	for {
		select {
		case killed = <- kill:
			debug("bgnoise killed")
		case _ = <- done:
			// trace("bg process done")
			if !killed {
				i++
				// trace("Next bg test %s.", bg.Test[i % m].Title)
				go bgRun(&(bg.Test[i % m]), bg.Global, done)
			} else {
				n--
				// trace("Left running %d:", n)
				if n==0 {
					// trace("begnois done all work: returning")
					return
				}
			}
		}
	}
}


func (s *Suite) stresstest(bg *Suite, load int) (okay bool, responsetime, fail, tot int) {
	okay = true
	var kill chan bool
	kill = make(chan bool)
	if load > 0 {
		go bgnoise(load, bg, kill)
		// start bg load
	}
	
	var totaltime int
	
	// TODO: repeate tests n times
	for _, t := range s.Test {
		tc := t.Copy()
		duration, err := tc.RunSingle(s.Global, false)
		if err != nil {
			okay = false
		}
		total, _, failed := tc.Stat()
		totaltime += duration
		fail += failed
		tot += total
	}

	responsetime = totaltime/len(s.Test)
	info("Load %d: Average response time %d [ms]. Failures %d\n", load, responsetime, fail)
	
	if load > 0 {
		kill <- true
	}
	
	return
}


type Stepper interface {
	Next(current int) int
}

type ConstantStep struct {
	Start int
	Step int
}

func (cs ConstantStep) Next(current int) int {
	if current == 0 {
		return cs.Start
	}
	return current + cs.Step
}

type FactorStep struct {
	Start int
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




func (s *Suite) Stress(bg *Suite, stepper Stepper) {
	var load int = 0
	var lastRespTime int = -1
	var plainRespTime int = -1
	var result string = "================== Stresstest Results =====================\n"
	
	for {
		info("Stresstesting with background load of %d || requests.", load)
		okay, resptime, failed, total := s.stresstest(bg, load)
		if plainRespTime == -1 {
			plainRespTime = resptime
		}
		result += fmt.Sprintf("Load %3d : AverageRespTime/ms = %4d , Failed = %3d , Total = %3d \n", load, resptime, failed, total)
		fmt.Printf(result)
		if !okay {
			info("Test Error: Aborting Stresstest.")
			break
		}
		if lastRespTime != -1 && resptime > AllowedSingleRtFactor*lastRespTime {
			info("Dramatic Single Response Time Increase: Aborting Stresstest.")
			break
		}
		if resptime > AllowedGeneralRtFactor*plainRespTime {
			info("Response Time Increased Too Much: Aborting Stresstest.")
			break
		}
		if resptime > AllowedMaximumRt {
			info("Response Time Long: Aborting Stresstest.")
			break
		}
		if failed > int(AllowedFailingFraction*float32(total)) {
			info("To Many Failures: Aborting Stresstest.")
			break
		}
		
		lastRespTime = resptime
		load = stepper.Next(load)
		
		if load > AllowedMaximumBgReq {
			info("To Many Background Request: Aborting Stresstest.")
			break
		}
		
		time.Sleep(SleepMs1 * 1000000)
	}

	fmt.Printf(result)
}
