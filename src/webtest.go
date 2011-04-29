package main

import (
	"fmt"
	"flag"
	"os"
	"strings"
	"strconv"
	"log"
	"./suite/suite"
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
	if toRun == ""  {
		return true
	}
	sp := strings.Split(toRun, ",", -1)
	title := s.Test[no].Title
	for _, x := range sp {
		i, err := strconv.Atoi(x)
		if err == nil {
			if (i==0 && s.Test[0].Title!="Global") || (i>0 && i<len(s.Test)) && no==i {
				return true
			}
		} else if strings.HasSuffix(x, "...") {
			x = x[0:len(x)-3]
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
	
	for _, filename := range(flag.Args()) {
		file, err := os.Open(filename, os.O_RDONLY, 777)
		if err != nil {
			fmt.Printf("Cannot read from '%s': %s\n", filename, err.String())
			continue
		}
		parser := suite.NewParser(file)
		s , _ := parser.ReadSuite()
		file.Close()
		
		result += "\nSuite " + filename + ":\n------------------------------------\n"
		
		for i, t := range s.Test {
			if checkOnly {
				fmt.Printf("\n%s\n", t.String())
				continue
			}
			// do not run global
			if i==0 && t.Title=="Global" { 
				continue 
			}
			if shouldRun(s, i) {
				debug("shouldRun(s, %d) == %v", i, shouldRun(s, i))
				s.RunTest(i)
			} else {
				info("Will not run test %d.", i)
			}
			at := t.Title
			if len(at) > 20 {
				at = at[0:18] + ".."
			}
			result += fmt.Sprintf("Test %2d: '%-20s': %s\n", i, at, s.Test[i].Status())
		}
		
	}
	// Summary
	if !checkOnly {
		fmt.Printf(result)
	}
}