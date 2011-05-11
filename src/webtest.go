package main

import (
	"fmt"
	"flag"
	"os"
	"strings"
	"strconv"
	"log"
	"sort"
	"path"
	"rand"
	"time"
	"http"
	"dobler/webtest/suite"
	"dobler/webtest/tag"
)

var checkOnly bool = false
var testmode bool = true
var benchmarkMode bool = false
var stresstestMode bool = false
var numRuns int = 15
var LogLevel int = 2 // 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace
var tagLogLevel int = -1
var suiteLogLevel int = -1
var testsToRun string = "*"
var randomSeed int64 = -1

var rampStart int = 5
var rampStep int = 5
var rampSleep int64 = 1000 // one second
var rampRep int = 1

var stopFF float64 = 0.1       // 10% Failures --> stop
var stopART int64 = 120 * 1000 // two minutes
var stopMRT int64 = 240 * 1000 // four minutes

var stopRTJ int = 5  // five fold increase in avg resp time in one ramp step   
var stopRTI int = 50 // 

var stopMPR = 250

var logger *log.Logger

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

// Determine wether test no should be run based on testsToRun
func shouldRun(s *suite.Suite, no int) bool {
	if testsToRun == "" {
		return true
	}
	sp := strings.Split(testsToRun, ",", -1)
	title := s.Test[no].Title
	for _, x := range sp {
		i, err := strconv.Atoi(x)
		if err == nil {
			if i > 0 && i < len(s.Test) && no == i {
				return true
			}
		} else {
			matches, err := path.Match(x, title)
			if err != nil {
				error("Malformed pattern '%s'.", x)
				return false
			}
			if matches {
				return true
			}
		}
	}
	return false
}

const MaxInt = int(^uint(0) >> 1)
const MinInt = -MaxInt - 1

func sixval(data []int) (min, lq, med, avg, uq, max int) {
	min, max = MaxInt, MinInt
	sum, n := 0, len(data)
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}

	avg = sum / n

	sort.SortInts(data)
	qi := n / 4
	lq, uq = data[qi], data[n-qi]
	med = data[n/2]

	return
}

func help() {
	fmt.Fprintf(os.Stderr, "\nUsage:\n")
	fmt.Fprintf(os.Stderr, "\twebtest -check [common options] <suite>...\n")
	fmt.Fprintf(os.Stderr, "\twebtest [-test] [common options] [test options] <suite>...\n")
	fmt.Fprintf(os.Stderr, "\twebtest -bench [common options] [bench options] <suite>...\n")
	fmt.Fprintf(os.Stderr, "\twebtest -stress [common options] [stress options] <background-suite> <test-suite>\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Test is the default mode and will run all test in the given suites.\n")
	fmt.Fprintf(os.Stderr, "Check will just read the testsuite, parse ist and output warning/erros found.\n")
	fmt.Fprintf(os.Stderr, "Benchmarking and Stress-Test are selected by -test or -stres.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Common Options:\n")
	fmt.Fprintf(os.Stderr, "\tcheck            do not run, just print suites \n")
	fmt.Fprintf(os.Stderr, "\tlog <n>          General Log Level 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace\n")
	fmt.Fprintf(os.Stderr, "\tlog.tag <n>      Log level for Tag test (tag package)\n")
	fmt.Fprintf(os.Stderr, "\tlog.suite <n>    Log level for suite package\n")
	fmt.Fprintf(os.Stderr, "\ttests <list>     select which tests to run: Comma seperated list of numbers or\n")
	fmt.Fprintf(os.Stderr, "\t                 namepattern. E.g. '3,7,External*,9,*-Special-??,15'\n")
	fmt.Fprintf(os.Stderr, "\tseed <n>         use n as random ssed (instead of current time)\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Test Options:\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Benchmark Options:\n")
	fmt.Fprintf(os.Stderr, "\truns             number of repetitions of each test (mus be >= 5).\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Stress Test Options:\n")
	fmt.Fprintf(os.Stderr, "\tramp.start <n>   Start with n as parallel background load\n")
	fmt.Fprintf(os.Stderr, "\tramp.step <n>    Increase parallel background load by n on each iteration\n")
	fmt.Fprintf(os.Stderr, "\tramp.sleep <ms>  Sleep tim in ms around iterations.\n")
	fmt.Fprintf(os.Stderr, "\tramp.rep <n>     Repetitions of testsuite during one ramp step\n")
	fmt.Fprintf(os.Stderr, "\tstop.ff <frac>   Stop stresstest if fraction (e.g. 0.2) of conditions fail\n")
	fmt.Fprintf(os.Stderr, "\tstop.art <ms>    Stop if Average Response Time exeeds ms.\n")
	fmt.Fprintf(os.Stderr, "\tstop.mrt <ms>    Stop if Maximum Response Time exeeds ms. \n")
	fmt.Fprintf(os.Stderr, "\tstop.rtj <n>     Stop if resp.time jumbs by factor of at least n. \n")
	fmt.Fprintf(os.Stderr, "\tstop.rti <n>     Stop if resp.time exeeds n * plain resp. time.\n")
	fmt.Fprintf(os.Stderr, "\tstop.mpr <n>     Stop if n maximum parallel requests reached.\n")
	fmt.Fprintf(os.Stderr, "\t\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Defaults:\n")
	fmt.Fprintf(os.Stderr, "\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	logger = log.New(os.Stderr, "Webtest ", log.Ldate|log.Ltime)

	var helpme bool
	flag.BoolVar(&helpme, "help", false, "Print usage info and exit.")
	flag.BoolVar(&checkOnly, "check", false, "Read test suite and output without testing.")
	flag.BoolVar(&benchmarkMode, "bench", false, "Benchmark suit: Run each test <runs> often.")
	flag.BoolVar(&testmode, "test", true, "Perform normal testing")
	flag.BoolVar(&stresstestMode, "stress", false, "Use background-suite as stress suite for tests.")
	flag.IntVar(&LogLevel, "log", 3, "General log level: 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace")
	flag.IntVar(&tagLogLevel, "log.tag", -1, "Log level for tag: -1: std level, 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace")
	flag.IntVar(&suiteLogLevel, "log.suite", -1, "Log level for suite: -1: std level, 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace")
	flag.IntVar(&numRuns, "runs", 15, "Number of runs for each test in benchmark.")
	flag.StringVar(&testsToRun, "tests", "*", "Run just some tests (numbers or name)")
	flag.Int64Var(&randomSeed, "seed", 15, "Number of runs for each test in benchmark.")

	flag.IntVar(&rampStart, "ramp.start", 5, "Ramp start")
	flag.IntVar(&rampStep, "ramp.step", 5, "Ramp step")
	flag.Int64Var(&rampSleep, "ramp.sleep", 1000, "Ramp sleep")
	flag.IntVar(&rampRep, "ramp.rep", 1, "Ramp repetition")
	flag.Float64Var(&stopFF, "stop.FF", 0.2, "Stop failed fraction limit")
	flag.Int64Var(&stopART, "stop.art", 120*1000, "Stop average repsonste time limit")
	flag.Int64Var(&stopMRT, "stop.mrt", 240*1000, "Stop maximum responste time limit ")
	flag.IntVar(&stopRTJ, "stop.rtj", 5, "Stop avg resptime step increase factor limit")
	flag.IntVar(&stopRTI, "stop.rti", 50, "Stop avg resptime total increas factor limit")
	flag.IntVar(&stopMPR, "stop.mpr", 250, "Stop if parallel request exeds this limit.")

	flag.Usage = help

	flag.Parse()
	if helpme {
		help()
	}
	if benchmarkMode && stresstestMode {
		fmt.Fprintf(os.Stderr, "Illegal combination of -stress, and -bench")
		os.Exit(1)
	}
	if benchmarkMode {
		testmode = false
	}
	if stresstestMode {
		testmode = false
	}

	if randomSeed != -1 {
		suite.Random = rand.New(rand.NewSource(int64(randomSeed)))
	}

	if tagLogLevel < 0 {
		tagLogLevel = LogLevel
	}
	if suiteLogLevel < 0 {
		suiteLogLevel = LogLevel
	}

	suite.LogLevel = suiteLogLevel
	tag.LogLevel = tagLogLevel

	if stresstestMode {
		if flag.NArg() != 2 {
			error("Stresstest requires excatly two suites.")
			os.Exit(1)
		}
		stresstest(flag.Args()[0], flag.Args()[1])
		os.Exit(0)
	} else {
		if flag.NArg() == 0 {
			error("No webtest file given.\n")
			os.Exit(1)
		}
		testOrBenchmark(flag.Args())
	}
}

func testOrBenchmark(filenames []string) {

	var result string = "\n======== Results ==========================================================\n"
	var charts string = "\n======== Charts ===========================================================\n"

	var passed bool = true
	var suites []*suite.Suite = make([]*suite.Suite, 0, 20)
	var basenames []string = make([]string, 0, 20)
	var allReadable bool = true

	for _, filename := range filenames {
		s, basename, err := readSuite(filename)
		if err != nil {
			allReadable = false
		} else {
			suites = append(suites, s)
			basenames = append(basenames, basename)
		}
	}

	if checkOnly {
		for _, s := range suites {
			for _, t := range s.Test {
				fmt.Printf("\n%s\n", t.String())
			}
		}
		return
	}
	if !allReadable {
		os.Exit(1)
	}

	for sn, s := range suites {

		result += "Suite " + basenames[sn] + ":\n----------------------------\n"
		charts += "Suite " + basenames[sn] + ":\n----------------------------\n"

		for i, t := range s.Test {
			at := t.Title
			if len(at) > 20 {
				at = at[0:18] + ".."
			}
			at = fmt.Sprintf("Test %2d: %-20s", i, at)

			if !shouldRun(s, i) {
				info("Skipped test %d.", i)
				continue
			}

			if benchmarkMode {
				dur, f, err := s.BenchTest(i, numRuns)
				if err != nil {
					result += fmt.Sprintf("%s: Unable to bench: %s\n", at, err.String())
				} else {
					min, lq, med, avg, uq, max := sixval(dur)
					result += fmt.Sprintf("%s:  min= %-4d , 25= %-4d , med= %-4d , avg= %-4d , 75= %-4d , max= %4d (in ms, %d runs, %d failures)\n",
						at, min, lq, med, avg, uq, max, len(dur), f)
					charts += benchChartUrl(dur, t.Title) + "\n"
				}
			} else {
				s.RunTest(i)
				result += fmt.Sprintf("%s: %s\n", at, s.Test[i].Status())
				if _, _, failed := s.Test[i].Stat(); failed > 0 {
					passed = false
				}
			}
		}
		result += "\n"
		charts += "\n"

	}
	// Summary
	fmt.Print(result)
	fmt.Print(charts)
	if passed {
		fmt.Printf("PASS\n")
		os.Exit(0)
	} else {
		fmt.Printf("FAIL\n")
		os.Exit(1)
	}
}

func readSuite(filename string) (s *suite.Suite, basename string, err os.Error) {
	var file *os.File
	file, err = os.Open(filename, os.O_RDONLY, 777)
	defer file.Close()

	if err != nil {
		error("Cannot read from '%s': %s\n", filename, err.String())
		return
	}
	basename = path.Base(filename)
	parser := suite.NewParser(file, filename)
	s, err = parser.ReadSuite()
	if err != nil {
		error("Problems parsing '%s': %s\n", filename, err.String())
	}
	return
}

func stressChartUrl(data []suite.StressResult) (url string) {
	url = "http://chart.googleapis.com/chart?cht=lxy&chs=500x300&chxs=0,676767,11.5,0,lt,676767&chxt=x,y,r"
	url += "&chls=2|2|2&chco=0000FF,00FF00,FF0000&chm=s,000000,0,-1,4|s,000000,1,-1,4|s,000000,2,-1,4"
	url += "&chdlp=b&chdl=Max+RT|Avg+RT|Err+Rate"
	var mrt int64 = -1
	for _, d := range data {
		if d.MaxRT > mrt {
			mrt = d.MaxRT
		}
	}
	mrt = 100 * ((mrt + 99) / 100)

	url += fmt.Sprintf("&chxr=1,0,%d", mrt)

	var chd string = "&chd=t:"
	var maxd string
	var avgd string
	var ld string
	var errd string
	for i, d := range data {
		if i > 0 {
			ld += ","
			maxd += ","
			avgd += ","
			errd += ","
		}
		ld += fmt.Sprintf("%d", d.Load)
		maxd += fmt.Sprintf("%d", int(100*d.MaxRT/mrt))
		avgd += fmt.Sprintf("%d", int(100*d.AvgRT/mrt))
		errd += fmt.Sprintf("%d", int(100*d.Fail/d.Total))
	}
	chd += ld + "|" + maxd + "|" + ld + "|" + avgd + "|" + ld + "|" + errd

	url += chd + "\n"
	return
}

func stressramp(bg, s *suite.Suite, stepper suite.Stepper) {
	var load int = 0
	var lastRespTime int64 = -1
	var plainRespTime int64 = -1
	var text string = "============================ Stresstest Results ==================================\n"
	var data []suite.StressResult = make([]suite.StressResult, 0, 5)

	for {
		info("Stresstesting with background load of %d || requests.", load)
		result := s.Stresstest(bg, load, rampRep, rampSleep)
		data = append(data, result)

		if plainRespTime == -1 {
			plainRespTime = result.AvgRT
		}
		text += fmt.Sprintf("Load %3d: Response Time %5d / %5d / %5d (min/avg/max). Status %2d / %2d / %2d (err/pass/fail). %2d / %2d (tests/checks).\n",
			load, result.MinRT, result.AvgRT, result.MaxRT, result.Err, result.Pass, result.Fail, result.N, result.Total)
		fmt.Printf(stressChartUrl(data))
		fmt.Printf(text)
		if result.Err > 0 {
			info("Test Error: Aborting Stresstest.")
			break
		}
		if lastRespTime != -1 && result.AvgRT > int64(stopRTJ)*lastRespTime {
			info("Dramatic Single Average Response Time Increase: Aborting Stresstest.")
			break
		}
		if result.AvgRT > int64(stopRTI)*plainRespTime {
			info("Average Response Time Increased Too Much: Aborting Stresstest.")
			break
		}
		if result.AvgRT > stopART {
			info("Average Response Time Long: Aborting Stresstest.")
			break
		}
		if result.MaxRT > stopMRT {
			info("Maximum Response Time Long: Aborting Stresstest.")
			break
		}
		if result.Fail > int(stopFF*float64(result.Total)) {
			info("To Many Failures: Aborting Stresstest.")
			break
		}

		lastRespTime = result.AvgRT
		load = stepper.Next(load)

		if load > stopMPR {
			info("To Many Background Request: Aborting Stresstest.")
			break
		}

		time.Sleep(rampSleep * 1000000)
	}

	fmt.Printf(stressChartUrl(data))
	fmt.Printf(text)
}


func stresstest(bgfilename, testfilename string) {
	// Read background and test suite
	background, _, berr := readSuite(bgfilename)
	testsuite, _, serr := readSuite(bgfilename)
	if berr != nil || serr != nil {
		error("Cannot parse given suites.")
		return
	}

	// Disable test which should not run by setting their Repet to 0
	for i := 0; i < len(testsuite.Test); i++ {
		if !shouldRun(testsuite, i) {
			warn("Disabeling test %s", testsuite.Test[i].Title)
			testsuite.Test[i].Setting["Repeat"] = "0"
		}
	}

	// perform increasing stresstests
	stressramp(background, testsuite, suite.ConstantStep{rampStart, rampStep})
}


func benchChartUrl(d []int, title string) (url string) {
	url = "http://chart.googleapis.com/chart?cht=bvg&chs=600x300&chxs=0,676767,11.5,0,lt,676767&chxt=x&chdlp=b"
	url += "&chbh=a&chco=404040&chtt=" + http.URLEscape(strings.Trim(title, " \t\n")) + "&chdl=Response+Times+[ms]"

	// Decide on number of bins
	min, _, _, _, _, max := sixval(d)
	cnt := 10
	if len(d) <= 10 {
		cnt = 3
	} else if len(d) <= 15 {
		cnt = 5
	} else if len(d) > 40 {
		cnt = 15
	}
	step := (max - min) / cnt

	// Binify and scale largest bar to 100
	var bin []int = make([]int, cnt)
	mc := 0
	for _, n := range d {
		b := (n - min) / step
		if b >= cnt {
			b = cnt - 1
		}
		bin[b] = bin[b] + 1
		if bin[b] > mc {
			mc = bin[b]
		}
	}
	for i, n := range bin {
		bin[i] = 100 * n / mc
	}

	// Output data to url
	url += fmt.Sprintf("&chxr=0,%d,%d", min+step/2, max-step/2)
	url += "&chd=t:"
	for i, n := range bin {
		if i > 0 {
			url += ","
		}
		url += fmt.Sprintf("%d", n)
	}
	return
}
