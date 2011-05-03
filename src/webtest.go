package main

import (
	"fmt"
	"flag"
	"os"
	"strings"
	"strconv"
	"log"
	"dobler/webtest/suite"
	"sort"
)

var benchmark bool = false
var numRuns int = 15
var LogLevel int = 2 // 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace
var toRun string = ""
var checkOnly bool

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


func shouldRun(s *suite.Suite, no int) bool {
	if toRun == "" {
		return true
	}
	sp := strings.Split(toRun, ",", -1)
	title := s.Test[no].Title
	for _, x := range sp {
		i, err := strconv.Atoi(x)
		if err == nil {
			if (i == 0 && s.Test[0].Title != "Global") || (i > 0 && i < len(s.Test)) && no == i {
				return true
			}
		} else if strings.HasSuffix(x, "...") {
			x = x[0 : len(x)-3]
			if strings.HasPrefix(title, x) {
				return true
			}
		} else {
			if title == x {
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
		if v < min { min = v }
		if v > max { max = v }
		sum += v
	}
	
	avg = sum/n

	sort.SortInts(data)
	qi := n/4
	lq, uq = data[qi], data[n-qi]
	med = data[n/2]
	
	return
}



func main() {
	flag.BoolVar(&benchmark, "bench", false, "Benchmark suit: Run each test <runs> often.")
	flag.IntVar(&numRuns, "runs", 15, "Number of runs for each test in benchmark.")
	flag.IntVar(&LogLevel, "log", 2, "Log level: 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace")
	flag.StringVar(&toRun, "test", "", "Run just some tests (numbers or name)")
	flag.BoolVar(&checkOnly, "check", false, "Read test suite and output without testing.")

	flag.Parse()

	suite.LogLevel = LogLevel

	if flag.NArg() == 0 {
		fmt.Printf("No webtest file given.\n")
		return
	}

	var result string = "\n============================ Results ============================\n"

	for _, filename := range flag.Args() {
		file, err := os.Open(filename, os.O_RDONLY, 777)
		if err != nil {
			fmt.Printf("Cannot read from '%s': %s\n", filename, err.String())
			continue
		}
		parser := suite.NewParser(file)
		s, _ := parser.ReadSuite()
		file.Close()

		result += "\nSuite " + filename + ":\n------------------------------------\n"

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
						result += fmt.Sprintf("%s: min: %-4d 25: %-4d med: %-4d avg: %-4d 75: %-4d max: %4d (in ms, %d runs, %d failures)\n", 
												at, min, lq, med, avg, uq, max, len(dur), f)
					}
				} else {
					s.RunTest(i)
					result += fmt.Sprintf("%s: %s\n", at, s.Test[i].Status())
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
}
