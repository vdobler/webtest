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
	"dobler/webtest/suite"
	"dobler/webtest/tag"
)

var checkOnly bool = false
var testmode bool = true
var benchmark bool = false
var stresstest bool = false
var numRuns int = 15
var LogLevel int = 2 // 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace
var tagLogLevel int = -1
var suiteLogLevel int = -1
var testsToRun string = ""
var randomSeed int = -1

func error(f string, m ...interface{}) {
	if LogLevel >= 1 {
		log.Print("*ERROR* " + fmt.Sprintf(f, m...))
	}
}
func warn(f string, m ...interface{}) {
	if LogLevel >= 2 {
		log.Print("*WARN * " + fmt.Sprintf(f, m...))
	}
}
func info(f string, m ...interface{}) {
	if LogLevel >= 3 {
		log.Print("*INFO * " + fmt.Sprintf(f, m...))
	}
}
func debug(f string, m ...interface{}) {
	if LogLevel >= 4 {
		log.Print("*DEBUG* " + fmt.Sprintf(f, m...))
	}
}
func trace(f string, m ...interface{}) {
	if LogLevel >= 5 {
		log.Print("*TRACE* " + fmt.Sprintf(f, m...))
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
	var helpme bool
	flag.BoolVar(&helpme, "help", false, "Print usage info and exit.")
	flag.BoolVar(&checkOnly, "check", false, "Read test suite and output without testing.")
	flag.BoolVar(&benchmark, "bench", false, "Benchmark suit: Run each test <runs> often.")
	flag.BoolVar(&testmode, "test", true, "Perform normal testing")
	flag.BoolVar(&checkOnly, "stress", false, "Use background-suite as stress suite for tests.")
	flag.IntVar(&LogLevel, "log", 2, "General log level: 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace")
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
	if stresstest {
		fmt.Fprintf(os.Stderr, "Stress Testing not implemented")
		os.Exit(1)
	}
	if (benchmark && testmode) || (benchmark && stresstest) || (testmode && stresstest) {
		fmt.Fprintf(os.Stderr, "Illegal combination of -stress, -test, -bench")
		os.Exit(1)
	}
	
	if randomSeed != -1 {
		suite.Random = rand.New(rand.NewSource(randomSeed))
	}

	if tagLogLevel < 0 {
		tagLogLevel = LogLevel
	}
	if suiteLogLevel < 0 {
		suiteLogLevel = LogLevel
	}

	suite.LogLevel = suiteLogLevel
	tag.LogLevel = tagLogLevel

	if flag.NArg() == 0 {
		fmt.Printf("No webtest file given.\n")
		return
	}

	var result string = "\n================================= Results =================================\n"

	var passed bool = true

	for _, filename := range flag.Args() {
		file, err := os.Open(filename, os.O_RDONLY, 777)
		if err != nil {
			fmt.Printf("Cannot read from '%s': %s\n", filename, err.String())
			continue
		}
		basename := filename
		if j := strings.LastIndex(basename, "/"); j != -1 {
			basename = basename[j+1:]
		}
		parser := suite.NewParser(file, basename)
		s, _ := parser.ReadSuite()
		file.Close()

		result += "\nSuite " + filename + ":\n-----------------------------------------\n"

		for i, t := range s.Test {
			if checkOnly {
				fmt.Printf("\n%s\n", t.String())
				continue
			}
			// do not run global
			if i == 0 && t.Title == "Global" {
				continue
			}
			at := t.Title
			if len(at) > 20 {
				at = at[0:18] + ".."
			}
			at = fmt.Sprintf("Test %2d: %-20s", i, at)

			if shouldRun(s, i) {
				if benchmark {
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
	if !checkOnly {
		fmt.Printf(result)
	}
	if passed {
		fmt.Printf("PASS\n")
		os.Exit(0)
	} else {
		fmt.Printf("FAIL\n")
		os.Exit(1)
	}
}
