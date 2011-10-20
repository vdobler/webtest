//
// Webtest - A domain specific language (DSL) to test websites.
//
// Copyright 2011 Volker Dobler. All rights reserved.
//
package main

import (
	"fmt"
	"flag"
	"os"
	"strings"
	"log"
	"path"
	"rand"
	"runtime"
	"time"

	"github.com/vdobler/webtest/suite"
	"github.com/vdobler/webtest/tag"
	"github.com/vdobler/webtest/stat"
)

// General operation modes and overall settings
var checkOnly bool = false
var testmode bool = true
var benchmarkMode bool = false
var stresstestMode bool = false
var tagspec string
var outputPath = "./"
var LogLevel int = 2 // 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace
var tagLogLevel int = -1
var suiteLogLevel int = -1

// Test settings
var validateMask int = 0
var showLinks bool = false
var testsToRun string = "*"
var randomSeed int64 = -1
var dumpTalk string = ""
var junitFile = ""

// Benchmark
var numRuns int = 15

// Parameters for stresstesting
var rampStart int = 5      // Start with that many parallel background requests
var rampStep int = 5       // Increase number of parallel background requests
var rampSleep int64 = 1000 // Time in ms to sleep before and after testing
var rampRep int = 1        // Number of repetitions of tests in one ramp level.

// Parameters determing the end of a stresstest: If any condition is reached, the stresstests stops
var stopFF float64 = 0.1       // 10% Failures --> stop
var stopART int64 = 120 * 1000 // two minutes Average Response Time
var stopMRT int64 = 240 * 1000 // four minutes Maximum Response Time
var stopRTJ int = 5            // five fold increase in avg resp time in _one_ ramp step   
var stopRTI int = 50           // fifty fold increase in avg resp time from value without background load 
var stopMPR = 250              // maximum number of parallel background requests

// Some logging stuff
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

// Determine whether test no of suite s (number sn) should be run based on testsToRun.
// Formats of testsToRun is a comma separated list if entries:
//   -  <num>          run this test (only in first/only) suite
//   -  <snum>.<tnum>  run test number tnum in suite number snum
//   -  <pattern>      pattern with * and ? in the usual meaning
// E.g. "2,4.5,Search*"
func shouldRun(s *suite.Suite, sn, no int) bool {
	if testsToRun == "" {
		return true
	}
	sp := strings.Split(testsToRun, ",")
	title := s.Test[no-1].Title
	for _, x := range sp {
		if x == fmt.Sprintf("%d.%d", sn, no) {
			return true
		}
		if sn == 1 && x == fmt.Sprintf("%d", no) {
			return true
		}
		matches, err := tag.Match(x, title)
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

// Print usage information and exit.
func help() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "\twebtest [-test] [common options] [test options] <suite>...\n")
	fmt.Fprintf(os.Stderr, "\twebtest -check [common options] <suite>...\n")
	fmt.Fprintf(os.Stderr, "\twebtest -bench [common options] [bench options] <suite>...\n")
	fmt.Fprintf(os.Stderr, "\twebtest -stress [common options] [stress options] <bg-suite> <suite>\n")
	fmt.Fprintf(os.Stderr, "\twebtest -tag <tagSpec> <htmlFile>\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Test is the default mode and will run alls test in the given suites.\n")
	fmt.Fprintf(os.Stderr, "Check will just read the testsuite(s), parse them and output the \n")
	fmt.Fprintf(os.Stderr, "warning/erros found and the suite(s) as read.\n")
	fmt.Fprintf(os.Stderr, "Benchmarking and Stress-Test are selected by -bench or -stres.\n")
	fmt.Fprintf(os.Stderr, "Debuging tag-specs matching against a html file is done by -tag.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "During benchmarking the selected tests are run repeatedly and\n")
	fmt.Fprintf(os.Stderr, "some simple statistics about the response times is collected.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "During stresstesting the request in the <bg-suite> are used to\n")
	fmt.Fprintf(os.Stderr, "generate appropriate background load (no tests are performed).\n")
	fmt.Fprintf(os.Stderr, "The <suite> itself is executed and testes on top of this load.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Common Options:\n")
	fmt.Fprintf(os.Stderr, "\t-log <n>          General Log Level: 0=off, 1=err, 2=warn, 3=info\n")
	fmt.Fprintf(os.Stderr, "\t                  4=debug, 5=trace. [%d]\n", LogLevel)
	fmt.Fprintf(os.Stderr, "\t-log.tag <n>      Log level for Tag test (tag package).\n")
	fmt.Fprintf(os.Stderr, "\t-log.suite <n>    Log level for suite package.\n")
	fmt.Fprintf(os.Stderr, "\t-tests <list>     select which tests to run: Comma separated list of\n")
	fmt.Fprintf(os.Stderr, "\t                  numbers or name patterns.  E.g. '3,7,External*' or\n")
	fmt.Fprintf(os.Stderr, "\t                  '9,*-Special-??,15'.  [%s]\n", testsToRun)
	fmt.Fprintf(os.Stderr, "\t-seed <n>         use n as random seed (instead of current time).\n")
	fmt.Fprintf(os.Stderr, "\t-D <n>=<v>        Set/override const variable named <n> to value <v>.\n")
	fmt.Fprintf(os.Stderr, "\t-od <path>        Set output path to <path>. [%s]\n", outputPath)
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Test Options:\n")
	fmt.Fprintf(os.Stderr, "\t-dump <mode>      Dump for debuggin purpose according to <mode>:\n")
	fmt.Fprintf(os.Stderr, "\t                  'all' dump wiretalk, 'body' dump response body\n")
	fmt.Fprintf(os.Stderr, "\t                  'none' don't dump anything.\n")
	fmt.Fprintf(os.Stderr, "\t                  If unused respect individual setting of each test.\n")
	fmt.Fprintf(os.Stderr, "\t-validate <n>     Allow checking links (1), validating html (2),\n")
	fmt.Fprintf(os.Stderr, "\t                  both (3).\n")
	fmt.Fprintf(os.Stderr, "\t-junit <file>     Write results as junit xml to <file>.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Benchmark Options:\n")
	fmt.Fprintf(os.Stderr, "\t-runs <n>         Number of repetitions of each test.\n")
	fmt.Fprintf(os.Stderr, "\t                  Must be >= 5. [%d]\n", numRuns)
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Stress Test Options:\n")
	fmt.Fprintf(os.Stderr, "\t-ramp.start <n>   Start with background load of <n> parallel\n")
	fmt.Fprintf(os.Stderr, "\t                  backgrounf requests. [%d]\n", rampStart)
	fmt.Fprintf(os.Stderr, "\t-ramp.step <n>    Increase parallel background load by <n> on each\n")
	fmt.Fprintf(os.Stderr, "\t                  iteration. [%d]\n", rampStep)
	fmt.Fprintf(os.Stderr, "\t-ramp.sleep <ms>  Sleep time in ms around iterations. [%d]\n", rampSleep)
	fmt.Fprintf(os.Stderr, "\t-ramp.rep <n>     Number of repetitions of whole testsuite during one\n")
	fmt.Fprintf(os.Stderr, "\t                  ramp step. [%d]\n", rampRep)
	fmt.Fprintf(os.Stderr, "\t-stop.ff <frac>   Stop stresstest if fraction (e.g. 0.2) of conditions\n")
	fmt.Fprintf(os.Stderr, "\t                  fail. [%.3f]\n", stopFF)
	fmt.Fprintf(os.Stderr, "\t-stop.art <ms>    Stop if Average Response Time exeeds <ms>. [%d]\n", stopART)
	fmt.Fprintf(os.Stderr, "\t-stop.mrt <ms>    Stop if Maximum Response Time exeeds <m>s. [%d]\n", stopMRT)
	fmt.Fprintf(os.Stderr, "\t-stop.rtj <n>     Stop if response time jumps by factor of\n")
	fmt.Fprintf(os.Stderr, "\t                  at least <n>. [%d]\n", stopRTJ)
	fmt.Fprintf(os.Stderr, "\t-stop.rti <n>     Stop if response time exeeds <n> * plain/initial\n")
	fmt.Fprintf(os.Stderr, "\t                  response time. [%d]\n", stopRTI)
	fmt.Fprintf(os.Stderr, "\t-stop.mpr <n>     Stop if n maximum parallel requests reached. [%d]\n", stopMPR)
	fmt.Fprintf(os.Stderr, "\t\n")
	os.Exit(1)
}

// Const-variables which can be set via the command line. Statisfied flag.Value interface.
type cmdlVar struct{ m map[string]string }

func (c cmdlVar) String() (s string) { return "" }
func (c cmdlVar) Set(s string) bool {
	part := strings.SplitN(s, "=", 2)
	if len(part) != 2 {
		warn("Bad argument '%s' to -D commandline parameter", s)
		return false
	}
	c.m[part[0]] = part[1]
	return true
}

// Set up internal state from command line.
func globalInitialization() {
	logger = log.New(os.Stderr, "Webtest ", log.Ldate|log.Ltime)

	var helpme bool
	var variables cmdlVar = cmdlVar{map[string]string{}}

	flag.BoolVar(&helpme, "help", false, "Print usage info and exit.")
	flag.BoolVar(&checkOnly, "check", false, "Read test suite and output without testing.")
	flag.BoolVar(&benchmarkMode, "bench", false, "Benchmark suit: Run each test <runs> often.")
	flag.BoolVar(&testmode, "test", true, "Perform normal testing")
	flag.BoolVar(&stresstestMode, "stress", false, "Use background-suite as stress suite for tests.")
	flag.IntVar(&validateMask, "validate", 0, "Bit mask which is ANDed to individual test setting.")
	flag.StringVar(&junitFile, "junit", "", "Write results as junit xml to file.")
	flag.IntVar(&LogLevel, "log", 3, "General log level: 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace")
	flag.IntVar(&tagLogLevel, "log.tag", -1,
		"Log level for tag: -1: std level, 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace")
	flag.IntVar(&suiteLogLevel, "log.suite", -1,
		"Log level for suite: -1: std level, 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace")
	flag.IntVar(&numRuns, "runs", 15, "Number of runs for each test in benchmark.")
	flag.StringVar(&testsToRun, "tests", "*", "Run just some tests (numbers or name)")
	flag.StringVar(&dumpTalk, "dump", "", "Dump wire talk.")
	flag.Int64Var(&randomSeed, "seed", -1, "Seed for random number generator.")
	flag.Var(variables, "D", "Set/Overwrite a const variable in the suite e.g. '-D HOST=localhost'")
	flag.StringVar(&outputPath, "od", outputPath, "Output into given directory.")
	flag.StringVar(&tagspec, "tag", "", "Check tag against html file.")

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

	if tagLogLevel < 0 {
		tagLogLevel = LogLevel
	}
	if suiteLogLevel < 0 {
		suiteLogLevel = LogLevel
	}

	suite.LogLevel = suiteLogLevel
	tag.LogLevel = tagLogLevel

	outputPath = path.Clean(outputPath)
	if !strings.HasSuffix(outputPath, "/") {
		outputPath += "/"
	}
	suite.OutputPath = outputPath

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

	if testmode && !(dumpTalk == "" || dumpTalk == "all" || dumpTalk == "none" || dumpTalk == "body") {
		fmt.Fprintf(os.Stderr, "Illegal argument to dump.")
		os.Exit(2)
	}

	runtime.GOMAXPROCS(2) // use more than one thread
}

// Main method for webtest.
func main() {

	globalInitialization()

	if tagspec != "" {
		if flag.NArg() != 1 {
			error("TagSpec debugging requires one file")
			os.Exit(2)
		}
		tagDebug(tagspec, flag.Args()[0])
	}

	if stresstestMode {
		if flag.NArg() != 2 {
			error("Stresstest requires excatly two suites.")
			os.Exit(2)
		}
		stresstest(flag.Args()[0], flag.Args()[1])
		os.Exit(0)
	} else {
		if flag.NArg() == 0 {
			error("No webtest file given. (Run 'webtest -help' for usage.)\n")
			os.Exit(2)
		}
		testOrBenchmark(flag.Args())
	}
}

// Helps debuging tagspecs.
func tagDebug(tagspec, filename string) {
	f, err := os.Open(filename)
	if err != nil {
		error("Cannot read from %s: %s", filename, err.String())
		os.Exit(2)
	}

	var result []byte
	buf := make([]byte, 100)
	for {
		n, err := f.Read(buf[0:])
		result = append(result, buf[0:n]...)
		if err != nil {
			if err == os.EOF {
				break
			}
			error("Problems: %s", err.String())
			os.Exit(2)
		}
	}

	html := string(result)

	fts := strings.Replace(tagspec, "||", "\n", -1)
	ts, err := tag.ParseTagSpec(fts)
	if err != nil {
		error("Invalid ts: %s", err.String())
		os.Exit(2)
	}

	doc, err := tag.ParseHtml(html)
	if err != nil {
		error("Cannot parse html: %s", err.String())
		os.Exit(2)
	}

	fmt.Printf("TagSpec:\n%s\n", strings.Replace(ts.String(), "\n", "\n    ", -1))
	all := tag.RankNodes(ts, doc)
	fmt.Printf("Rank CO RA XA RC XC SN DN   Tag\n--------------------------------------------\n")
	for n, q := range all {
		fmt.Printf("%2d:  %2d %2d %2d %2d %2d %2d %2d  %s\n", n, q.Content, q.ReqAttr, q.ForbAttr, q.ReqClass, q.ForbClass, q.Sub, q.Deep, q.Node.String())
	}

	os.Exit(0)
}

// Standard test or benchmarking.
func testOrBenchmark(filenames []string) {

	var suites []*suite.Suite = make([]*suite.Suite, 0, 20)
	var allReadable bool = true

	for _, filename := range filenames {
		s, _, err := readSuite(filename)
		if err != nil {
			allReadable = false
		} else {
			suites = append(suites, s)
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
			if s.Global != nil {
				fmt.Printf("\n%s\n", s.Global.String())
			}
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

	if benchmarkMode {
		benchmark(suites)
	} else {
		test(suites)
	}
}

func abbrevTitle(n int, title string) string {
	if len(title) > 25 {
		title = title[0:23] + ".."
	}
	return fmt.Sprintf("Test %2d: %-25s", n, title)
}

// Benchmarking
func benchmark(suites []*suite.Suite) {
	var result string = "\n======== Results ===============================================================\n"
	var charts string = "\n======== Charts ================================================================\n"

	for sn, s := range suites {
		var headline string

		if len(suites) > 1 {
			headline = "Suite " + s.Name + ":\n-----------------------------------\n"
			result += headline
			charts += headline
		}

		for i, t := range s.Test {
			if !shouldRun(s, sn+1, i+1) {
				info("Skipped test %d.", i+1)
				continue
			}
			abbrTitle := abbrevTitle(i+1, t.Title)

			// Benchmarking
			dur, f, err := s.BenchTest(i, numRuns)
			if err != nil {
				result += fmt.Sprintf("%s: Unable to bench: %s\n", abbrTitle, err.String())
			} else {
				min, lq, med, avg, uq, max := stat.SixvalInt(dur, 25)
				result += fmt.Sprintf("%s:  min= %-4d , 25= %-4d , med= %-4d , avg= %-4d , 75= %-4d , max= %4d (in ms, %d runs, %d failures)\n",
					abbrTitle, min, lq, med, avg, uq, max, len(dur), f)
				charts += stat.HistogramChartUrlInt(dur, t.Title, "Response Time [ms]") + "\n"
			}
		}
		result += "\n"
		charts += "\n"

	}

	filename := outputPath + "wtresults_" + time.LocalTime().Format("2006-01-02_15-04-05") + ".txt"
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		error("Cannot write to " + filename)
	}

	fmt.Print(result)
	fmt.Print(charts)
	if file != nil {
		file.Write([]byte(result))
		file.Write([]byte(charts))
	}
}

// Junit output uses the following mapping
//   file my-suite.wt   <-->  testsuite
//   test ----Name----  <-->  testcase
//   check/condition    <-->  assertion
// 
func xmlEnc(s string) string {
	s = strings.Replace(s, "&", "&amp;", -1)
	s = strings.Replace(s, "<", "&lt;", -1)
	return strings.Replace(s, ">", "&gt;", -1)
}

func xmlAttr(name, value string) string {
	value = strings.Replace(xmlEnc(value), "\"", "&quot;", -1)
	return fmt.Sprintf(`%s="%s"`, name, value)
}

// Testting
func test(suites []*suite.Suite) {
	var result string = "\n======== Results ===============================================================\n"
	var fails string = "\n======== Failures ==============================================================\n"
	var errors string = "\n======== Errors ================================================================\n"
	var passed bool = true
	var hasFailures, hasErrors bool // global over all suites


	var junit = "<?xml version=\"1.0\" encoding=\"UTF-8\" ?>\n"
	junit += "<testsuites>\n"
	for sn, s := range suites {
		var headline string
		var failed, erred bool // this suite

		if len(suites) > 1 {
			headline = "Suite " + s.Name + ":\n-----------------------------------\n"
			result += headline
		}

		// Junit stuff
		var milliseconds int64 = time.UTC().Nanoseconds() / 1e6
		tnump, tnumf, tnume, tnums := 0, 0, 0, 0 // total number of pass, fail, err, skipped
		timestamp := time.LocalTime().Format("2006-01-02T15:04:05")
		hostname, _ := os.Hostname()
		testcases := ""

		for i, t := range s.Test {
			if !shouldRun(s, sn+1, i+1) {
				info("Skipped test %d.", i+1)
				continue
			}
			abbrTitle := abbrevTitle(i+1, t.Title)

			origDump, _ := s.Test[i].Setting["Dump"]
			switch dumpTalk {
			case "all":
				s.Test[i].Setting["Dump"] = 1
			case "none":
				s.Test[i].Setting["Dump"] = 0
			case "body":
				s.Test[i].Setting["Dump"] = 3
			}

			// clear unwanted validation
			s.Test[i].Setting["Validate"] = s.Test[i].Validate() & validateMask

			s.RunTest(i)

			result += fmt.Sprintf("%s: %s\n", abbrTitle, s.Test[i].Status())
			nump, numf, nume := s.Test[i].Stat()
			tnump += nump
			tnumf += numf
			tnume += nume

			if numf+nume > 0 {
				passed = false
			}
			for _, res := range s.Test[i].Result {
				if res.Status == suite.TestFailed {
					if !failed {
						fails += headline
					}
					hasFailures, failed = true, true
					fails += fmt.Sprintf("%s: %s\n", abbrTitle, res.Cause)
					if len(res.Message) > 0 {
						for _, x := range strings.Split(res.Message, "\n") {
							fails += "                                    "
							fails += x + "\n"
						}
					}
				} else if res.Status == suite.TestErrored {
					if !erred {
						errors += headline
					}
					hasErrors, erred = true, true
					errors += fmt.Sprintf("%s: %s\n", abbrTitle, res)
				}

				// Junit stuff
				testcases += fmt.Sprintf("     <testcase %s %s>",
					xmlAttr("classname", t.Title), xmlAttr("name", t.Title+":"+res.Id))
				if res.Status == suite.TestPassed {
					// Noop
				} else if res.Status == suite.TestFailed {
					testcases += fmt.Sprintf("\n       <failure %s>\n",
						xmlAttr("type", res.Cause))
					testcases += fmt.Sprintf("         %s\n", xmlEnc(res.Message))
					testcases += fmt.Sprintf("       </failure>\n     ")
				} else if res.Status == suite.TestErrored {
					testcases += fmt.Sprintf("\n       <error type=\"%s\">\n",
						res.Cause)
					testcases += fmt.Sprintf("         %s\n", res.Message)
					testcases += fmt.Sprintf("       </error>\n    ")
				}
				testcases += fmt.Sprintf("</testcase>\n")

			}
			if s.Test[i].Abort() == 1 {
				fmt.Printf("Aborting suite.\n")
				errors += fmt.Sprintf("%s: Aborted whole suite.\n", abbrTitle)
				break
			}
			s.Test[i].Setting["Dump"] = origDump

		}
		result += "\n"

		// Junit stuff
		seconds := float64(time.UTC().Nanoseconds()/1e6-milliseconds) / 1000
		junit += fmt.Sprintf("  <testsuite %s %s %s", xmlAttr("name", s.Name),
			xmlAttr("hostname", hostname), xmlAttr("timestamp", timestamp))
		junit += fmt.Sprintf(" tests=\"%d\" failures=\"%d\" errors=\"%d\"",
			tnump+tnumf+tnume, tnumf, tnume)
		junit += fmt.Sprintf(" disabled=\"%d\" time=\"%.2f\" >\n", tnums, seconds)
		junit += testcases
		junit += "  </testsuite>\n"

	}

	junit += "<testsuites>\n"

	filename := outputPath + "wtresults_" + time.LocalTime().Format("2006-01-02_15-04-05") + ".txt"
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

	if hasFailures {
		fmt.Print(fails)
		if file != nil {
			file.Write([]byte(fails))
		}
	}
	if hasErrors {
		fmt.Print(errors)
		if file != nil {
			file.Write([]byte(errors))
		}
	}
	file.Sync()

	if junitFile != "" {
		fmt.Println()
		fmt.Println(junit)
	}

	if passed {
		fmt.Printf("\nPASS\n")
		os.Exit(0)
	} else {
		fmt.Printf("\nFAIL\n")
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
	parser := suite.NewParser(file, basename)
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
		runtime.GC()
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
	testsuite, _, serr := readSuite(testfilename)
	if berr != nil || serr != nil {
		error("Cannot parse given suites.")
		return
	}

	// Disable test which should not run by setting their Repeat to 0
	for i := 0; i < len(testsuite.Test); i++ {
		if !shouldRun(testsuite, 0, i+1) {
			warn("Disabeling test %s", testsuite.Test[i].Title)
			testsuite.Test[i].Setting["Repeat"] = 0
		}
	}

	// perform increasing stresstests
	stressramp(background, testsuite, suite.ConstantStep{rampStart, rampStep})
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
