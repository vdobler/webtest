package main

// Copyright 2011 Dr. Volker Dobler. All rights reserved.

import (
	"fmt"
	"flag"
	"os"
	"strings"
	"log"
	"sort"
	"path"
	"rand"
	"time"
	"http"
	"html"
	"dobler/webtest/suite"
	"dobler/webtest/tag"
)

var checkOnly bool = false
var testmode bool = true
var benchmarkMode bool = false
var stresstestMode bool = false
var validateFlag bool = false
var showLinks bool = false

var numRuns int = 15
var LogLevel int = 2 // 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace
var tagLogLevel int = -1
var suiteLogLevel int = -1
var testsToRun string = "*"
var randomSeed int64 = -1
var dumpTalk string = ""

// Stresstesting
var rampStart int = 5      // Start with that many parallel background requests
var rampStep int = 5       // Increase in number of parallel background requests
var rampSleep int64 = 1000 // Time in ms to sleep before and after testing
var rampRep int = 1        // Number of repetitions of tests in one ramp level.

// Parameters determing stresstest end
var stopFF float64 = 0.1       // 10% Failures --> stop
var stopART int64 = 120 * 1000 // two minutes Average Response Time
var stopMRT int64 = 240 * 1000 // four minutes Maximum Response Time
var stopRTJ int = 5            // five fold increase in avg resp time in one ramp step   
var stopRTI int = 50           // fifty fold increase in avg resp time from value without background load 
var stopMPR = 250              // maximum number of parallel background requests

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

// Determine whether test no of suite s (number sn) should be run based on testsToRun
func shouldRun(s *suite.Suite, sn, no int) bool {
	if testsToRun == "" {
		return true
	}
	sp := strings.Split(testsToRun, ",", -1)
	title := s.Test[no-1].Title
	for _, x := range sp {
		if x == fmt.Sprintf("%d.%d", sn, no) {
			return true
		}
		if sn == 1 && x == fmt.Sprintf("%d", no) {
			return true
		}
		matches, err := path.Match(x, title)
		if err != nil {
			error("Malformed pattern '%s' in -tests.", x)
			continue
		}
		if matches {
			return true
		}
	}
	return false
}

const MaxInt = int(^uint(0) >> 1)
const MinInt = -MaxInt - 1

// Compute minimum, 0.25 percentil, median, average, 75% percentil and maximum of values in data.
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


// Print usage information and exit.
func help() {
	fmt.Fprintf(os.Stderr, "\nUsage:\n")
	fmt.Fprintf(os.Stderr, "\twebtest [-test] [common options] [test options] <suite>...\n")
	fmt.Fprintf(os.Stderr, "\twebtest -check [common options] <suite>...\n")
	fmt.Fprintf(os.Stderr, "\twebtest -bench [common options] [bench options] <suite>...\n")
	fmt.Fprintf(os.Stderr, "\twebtest -stress [common options] [stress options] <background-suite> <test-suite>\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Test is the default mode and will run all test in the given suites.\n")
	fmt.Fprintf(os.Stderr, "Check will just read the testsuite(s), parse it and output warning/erros found.\n")
	fmt.Fprintf(os.Stderr, "Benchmarking and Stress-Test are selected by -bench or -stres.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Common Options:\n")
	fmt.Fprintf(os.Stderr, "\t-check            Do not run, just print suites.\n")
	fmt.Fprintf(os.Stderr, "\t-log <n>          General Log Level 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace.\n")
	fmt.Fprintf(os.Stderr, "\t-log.tag <n>      Log level for Tag test (tag package).\n")
	fmt.Fprintf(os.Stderr, "\t-log.suite <n>    Log level for suite package.\n")
	fmt.Fprintf(os.Stderr, "\t-tests <list>     select which tests to run: Comma seperated list of numbers or\n")
	fmt.Fprintf(os.Stderr, "\t                  namepattern. E.g. '3,7,External*,9,*-Special-??,15'.\n")
	fmt.Fprintf(os.Stderr, "\t-seed <n>         use n as random seed (instead of current time).\n")
	fmt.Fprintf(os.Stderr, "\t-D <n>=<v>        Set/override const variable named <n> to value <v>.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Test Options:\n")
	fmt.Fprintf(os.Stderr, "\t-dump [all|none]  Dump all wire talk or none. If unused respect indiv setting.\n")
	fmt.Fprintf(os.Stderr, "\t-validate         Validate html of test passes.\n")
	fmt.Fprintf(os.Stderr, "\t-links            Show links and their status if test passes.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Benchmark Options:\n")
	fmt.Fprintf(os.Stderr, "\t-runs <n>         Number of repetitions of each test (must be >= 5).\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Stress Test Options:\n")
	fmt.Fprintf(os.Stderr, "\t-ramp.start <n>   Start with background load of n parallel background requests.\n")
	fmt.Fprintf(os.Stderr, "\t-ramp.step <n>    Increase parallel background load by n on each iteration.\n")
	fmt.Fprintf(os.Stderr, "\t-ramp.sleep <ms>  Sleep time in ms around iterations.\n")
	fmt.Fprintf(os.Stderr, "\t-ramp.rep <n>     Number of repetitions of whole testsuite during one ramp step.\n")
	fmt.Fprintf(os.Stderr, "\t-stop.ff <frac>   Stop stresstest if fraction (e.g. 0.2) of conditions fail.\n")
	fmt.Fprintf(os.Stderr, "\t-stop.art <ms>    Stop if Average Response Time exeeds ms.\n")
	fmt.Fprintf(os.Stderr, "\t-stop.mrt <ms>    Stop if Maximum Response Time exeeds ms. \n")
	fmt.Fprintf(os.Stderr, "\t-stop.rtj <n>     Stop if response time jumps by factor of at least n. \n")
	fmt.Fprintf(os.Stderr, "\t-stop.rti <n>     Stop if response time exeeds n * plain resp. time.\n")
	fmt.Fprintf(os.Stderr, "\t-stop.mpr <n>     Stop if n maximum parallel requests reached.\n")
	fmt.Fprintf(os.Stderr, "\t\n")
	fmt.Fprintf(os.Stderr, "Defaults:\n")
	flag.PrintDefaults()
	os.Exit(1)
}

// Const-variables which can be set via the command line. Statisfied flag.Value interface.
type cmdlVar struct {
	m map[string]string
}

func (c cmdlVar) String() (s string) {
	return ""
}

func (c cmdlVar) Set(s string) bool {
	part := strings.Split(s, "=", 2)
	if len(part) != 2 {
		warn("Bad argument '%s' to -D commandline parameter", s)
		return false
	}
	c.m[part[0]] = part[1]
	return true
}


func globalInitialization() {
	logger = log.New(os.Stderr, "Webtest ", log.Ldate|log.Ltime)

	var helpme bool
	flag.BoolVar(&helpme, "help", false, "Print usage info and exit.")
	flag.BoolVar(&checkOnly, "check", false, "Read test suite and output without testing.")
	flag.BoolVar(&benchmarkMode, "bench", false, "Benchmark suit: Run each test <runs> often.")
	flag.BoolVar(&testmode, "test", true, "Perform normal testing")
	flag.BoolVar(&stresstestMode, "stress", false, "Use background-suite as stress suite for tests.")
	flag.BoolVar(&validateFlag, "validate", false, "Validate html if test passes.")
	flag.BoolVar(&showLinks, "links", false, "Show links and stati if tests passes.")
	flag.IntVar(&LogLevel, "log", 3, "General log level: 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace")
	flag.IntVar(&tagLogLevel, "log.tag", -1, "Log level for tag: -1: std level, 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace")
	flag.IntVar(&suiteLogLevel, "log.suite", -1, "Log level for suite: -1: std level, 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace")
	flag.IntVar(&numRuns, "runs", 15, "Number of runs for each test in benchmark.")
	flag.StringVar(&testsToRun, "tests", "*", "Run just some tests (numbers or name)")
	flag.StringVar(&dumpTalk, "dump", "", "Dump wire talk.")
	flag.Int64Var(&randomSeed, "seed", -1, "Seed for random number generator.")
	var variables cmdlVar = cmdlVar{map[string]string{}}
	flag.Var(variables, "D", "Set/Overwrite a const variable in the suite e.g. '-D HOST=localhost'")

	flag.IntVar(&rampStart, "ramp.start", 5, "Ramp start")
	flag.IntVar(&rampStep, "ramp.step", 5, "Ramp step")
	flag.Int64Var(&rampSleep, "ramp.sleep", 1000, "Ramp sleep in ms")
	flag.IntVar(&rampRep, "ramp.rep", 1, "Ramp repetition")
	flag.Float64Var(&stopFF, "stop.FF", 0.2, "Stop failed fraction limit")
	flag.Int64Var(&stopART, "stop.art", 120*1000, "Stop average repsonse time limit")
	flag.Int64Var(&stopMRT, "stop.mrt", 240*1000, "Stop maximum response time limit ")
	flag.IntVar(&stopRTJ, "stop.rtj", 5, "Stop avg resptime step increase factor limit")
	flag.IntVar(&stopRTI, "stop.rti", 50, "Stop avg resptime total increase factor limit")
	flag.IntVar(&stopMPR, "stop.mpr", 250, "Stop if parallel request exceds this limit.")

	flag.Usage = help

	flag.Parse()
	if helpme {
		help()
	}
	for vn, vv := range variables.m {
		suite.Const[vn] = vv
	}

	if benchmarkMode && stresstestMode {
		fmt.Fprintf(os.Stderr, "Illegal combination of -stress, and -bench")
		os.Exit(2)
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

	if testmode && !(dumpTalk == "" || dumpTalk == "all" || dumpTalk == "none") {
		fmt.Fprintf(os.Stderr, "Illegal argument to dump.")
		os.Exit(2)
	}

	if tagLogLevel < 0 {
		tagLogLevel = LogLevel
	}
	if suiteLogLevel < 0 {
		suiteLogLevel = LogLevel
	}

	suite.LogLevel = suiteLogLevel
	tag.LogLevel = tagLogLevel

}


// Main method for webtest.
func main() {
	globalInitialization()

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


func getAttr(a []html.Attribute, name string) string {
	for _, at := range a {
		if at.Key == name {
			return at.Val
		}
	}
	return ""
}


// Standard test or benchmarking.
func testOrBenchmark(filenames []string) {

	var result string = "\n======== Results ===============================================================\n"
	var charts string = "\n======== Charts ================================================================\n"
	var fails string = "\n======== Failures ==============================================================\n"

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
		var list string = "# List of tests:\n"
		var w int
		for _, s := range suites {
			if len(s.Name) > w {
				w = len(s.Name)
			}
		}
		for sn, s := range suites {
			for tn, t := range s.Test {
				fmt.Printf("\n%s\n", t.String())
				no := fmt.Sprintf("%d.%d", sn+1, tn+1)
				list += fmt.Sprintf("# %4s: %-*s: %s\n", no, w, s.Name, t.Title)
			}
		}
		fmt.Printf("\n%s", list)
		return
	}
	if !allReadable {
		os.Exit(2)
	}

	for sn, s := range suites {

		result += "Suite " + basenames[sn] + ":\n-----------------------------------\n"
		charts += "Suite " + basenames[sn] + ":\n-----------------------------------\n"
		fails += "Suite " + basenames[sn] + ":\n-----------------------------------\n"

		for i, t := range s.Test {
			if !shouldRun(s, sn+1, i+1) {
				info("Skipped test %d.", i+1)
				continue
			}

			abbrTitle := t.Title
			if len(abbrTitle) > 25 {
				abbrTitle = abbrTitle[0:23] + ".."
			}
			abbrTitle = fmt.Sprintf("Test %2d: %-25s", i+1, abbrTitle)

			if benchmarkMode {
				dur, f, err := s.BenchTest(i, numRuns)
				if err != nil {
					result += fmt.Sprintf("%s: Unable to bench: %s\n", abbrTitle, err.String())
				} else {
					min, lq, med, avg, uq, max := sixval(dur)
					result += fmt.Sprintf("%s:  min= %-4d , 25= %-4d , med= %-4d , avg= %-4d , 75= %-4d , max= %4d (in ms, %d runs, %d failures)\n",
						abbrTitle, min, lq, med, avg, uq, max, len(dur), f)
					charts += benchChartUrl(dur, t.Title) + "\n"
				}
			} else {
				origDump, _ := s.Test[i].Setting["Dump"]
				if dumpTalk == "all" {
					s.Test[i].Setting["Dump"] = "1"
				} else if dumpTalk == "none" {
					s.Test[i].Setting["Dump"] = "0"
				}
				s.RunTest(i)
				result += fmt.Sprintf("%s: %s\n", abbrTitle, s.Test[i].Status())
				if _, _, failed := s.Test[i].Stat(); failed > 0 {
					passed = false
					for _, res := range s.Test[i].Result {
						if !strings.HasPrefix(res, "Passed") {
							fails += fmt.Sprintf("%s: %s\n", abbrTitle, res)
						}
					}
					if s.Test[i].Abort() {
						fmt.Printf("Aborting suite.\n")
						break
					}
				}
				s.Test[i].Setting["Dump"] = origDump

				if showLinks {
					if s.Test[i].Body != nil {
					}
				}

			}
		}
		result += "\n"
		charts += "\n"

	}

	filename := "wtresults_" + time.LocalTime().Format("2006-01-02_15-04-05") + ".txt"
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		error("Cannot write to " + filename)
	}

	// Summary
	fmt.Print(result)
	if file != nil {
		file.Write([]byte(result))
	}

	if benchmarkMode {
		fmt.Print(charts)
		if file != nil {
			file.Write([]byte(charts))
		}
	} else if !passed {
		fmt.Print(fails)
		if file != nil {
			file.Write([]byte(fails))
		}
	}

	if passed {
		fmt.Printf("PASS\n")
		os.Exit(0)
	} else {
		fmt.Printf("FAIL\n")
		os.Exit(1)
	}
}


// Read a webtest suite from file.
func readSuite(filename string) (s *suite.Suite, basename string, err os.Error) {
	var file *os.File
	file, err = os.Open(filename)
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
	s.Name = basename
	return
}


// Real stresstest: Ramp up load until "collaps".
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


//  Perform stresstest.
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
		if !shouldRun(testsuite, 0, i) {
			warn("Disabeling test %s", testsuite.Test[i].Title)
			testsuite.Test[i].Setting["Repeat"] = "0"
		}
	}

	// perform increasing stresstests
	stressramp(background, testsuite, suite.ConstantStep{rampStart, rampStep})
}


// Generate Google chart for benchmark results.
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


// Generate Google chart for stresstest results.
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
