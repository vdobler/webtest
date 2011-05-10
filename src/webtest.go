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
var testsToRun string = ""
var randomSeed int = -1

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
			if (i == 0 && s.Test[0].Title != "Global") || (i > 0 && i < len(s.Test)) && no == i {
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

func fiveval(data []int) (min, lq, med, avg, uq, max int) {
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
	fmt.Fprintf(os.Stderr, "Benchmarking and Stress-Test (not implemented) are selected by -test or -stres.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Common Options:\n")
	fmt.Fprintf(os.Stderr, "\tcheck      do not run \n")
	fmt.Fprintf(os.Stderr, "\tlog <n>        General Log Level 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace\n")
	fmt.Fprintf(os.Stderr, "\tlog.tag <n>    Log level for Tag test (tag package)\n")
	fmt.Fprintf(os.Stderr, "\tlog.suite <n>  Log level for suite package\n")
	fmt.Fprintf(os.Stderr, "\ttests <list>   select which tests to run: Comma seperated list of numbers or\n")
	fmt.Fprintf(os.Stderr, "\t               uniq test names prefixes.\n")
	fmt.Fprintf(os.Stderr, "\tseed <n>       use n as random ssed (instead of current time)\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Test Options:\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Benchmark Options:\n")
	fmt.Fprintf(os.Stderr, "\truns\tnumber of repetitions of each test (mus be >= 5).\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Stress Test Options:\n")
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
	flag.StringVar(&testsToRun, "tests", "", "Run just some tests (numbers or name)")
	flag.IntVar(&randomSeed, "seed", 15, "Number of runs for each test in benchmark.")
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
	
	var result string = "\n================================= Results =================================\n"

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
	
		result += "\nSuite " + basenames[sn] + ":\n-----------------------------------------\n"

		for i, t := range s.Test {
			at := t.Title
			if len(at) > 20 {
				at = at[0:18] + ".."
			}
			at = fmt.Sprintf("Test %2d: %-20s", i, at)

			if shouldRun(s, i) {
				if benchmarkMode {
					dur, f, err := s.BenchTest(i, numRuns)
					if err != nil {
						result += fmt.Sprintf("%s: Unable to bench: %s\n", at, err.String())
					} else {
						min, lq, med, avg, uq, max := fiveval(dur)
						result += fmt.Sprintf("%s:  min= %-4d , 25= %-4d , med= %-4d , avg= %-4d , 75= %-4d , max= %4d (in ms, %d runs, %d failures)\n",
							at, min, lq, med, avg, uq, max, len(dur), f)
					}
				} else {
					s.RunTest(i)
					result += fmt.Sprintf("%s: %s\n", at, s.Test[i].Status())
					if _, _, failed := s.Test[i].Stat(); failed > 0 {
						passed = false
					}
				}
			} else {
				info("Skipped test %d.", i)
			}
		}

	}
	// Summary
	fmt.Printf(result)
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

func stresstest(bgfilename, testfilename string) {
	
	background, _, berr := readSuite(bgfilename)
	testsuite, _, serr := readSuite(bgfilename)
	if berr != nil || serr != nil {
		error("Cannot parse given suites.")
		return
	}
	testsuite.Stress(background, suite.ConstantStep{10, 10})
}