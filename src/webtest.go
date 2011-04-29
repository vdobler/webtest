package main

import (
	"fmt"
	"flag"
	"os"
	"./suite/suite"
)

var benchmark bool
var numRuns int = 15
var logLevel int = 2

func main() {
	flag.BoolVar(&benchmark, "bench", false, "Benchmark suit: Run each test <runs> often.")
	flag.IntVar(&numRuns, "runs", 15, "Number of runs for each test in benchmark.")
	flag.IntVar(&logLevel, "log", 2, "Log level: 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace")

	flag.Parse()
	
	suite.LogLevel = logLevel
		
	if flag.NArg() == 0 {
		fmt.Printf("No webtest file given.\n")
		return
	}
	
	for _, filename := range(flag.Args()) {
		file, err := os.Open(filename, os.O_RDONLY, 777)
		if err != nil {
			fmt.Printf("Cannot read from '%s': %s\n", filename, err.String())
			continue
		}
		parser := suite.NewParser(file)
		s , _ := parser.ReadSuite()
		for i, t:= range s.Test {
			fmt.Printf("\n\n# Test %d\n%s\n", i, t.String())
			s.RunTest(i)
		}
		
	}
}