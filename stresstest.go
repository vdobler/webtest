package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"time"

	"github.com/ajstarks/svgo"
	"github.com/vdobler/chart"
	"github.com/vdobler/chart/svgg"
	"github.com/vdobler/webtest/stat"
	"github.com/vdobler/webtest/suite"
)

var statLevels = []int{0, 25, 50, 67, 75, 80, 90, 95, 98, 100}

// Real stresstest: Ramp up load until "collaps".
func stressramp(bg, s *suite.Suite, stepper suite.Stepper, name string) {
	var load int = 0
	var lastRespTime int64 = -1
	var plainRespTime int64 = -1
	var text string = "============================ Stresstest Results ==================================\n"
	var data []suite.StressResult = make([]suite.StressResult, 0, 20)

	// Warmup of server: Make sure chaches are hot.
	s.Stresstest(bg, 0, 1, 10)

	for {
		warnf("Stresstesting with background load of %d || requests.", load)
		result := s.Stresstest(bg, load, rampRep, rampSleep)
		data = append(data, result)

		if plainRespTime == -1 {
			plainRespTime = result.AvgRT
		}
		text += fmt.Sprintf("Load %3d: Response Time %5d / %5d / %5d (min/avg/max). Status %2d / %2d / %2d (err/pass/fail). %2d / %2d (tests/checks).\n              ",
			load, result.MinRT, result.AvgRT, result.MaxRT, result.Err, result.Pass, result.Fail, result.N, result.Total)

		p := stat.DistributionInt(result.RT, statLevels)
		for i, lev := range statLevels {
			text += fmt.Sprintf("%d%%<%d  ", lev, p[i])
		}
		text += "\n"

		fmt.Print(stressChartUrl(data))
		fmt.Print(text)
		if result.Err > 0 {
			infof("Test Error: Aborting Stresstest.")
			break
		}
		if lastRespTime != -1 && result.AvgRT > int64(stopRTJ)*lastRespTime {
			infof("Dramatic Single Average Response Time Increase: Aborting Stresstest.")
			break
		}
		if result.AvgRT > int64(stopRTI)*plainRespTime {
			infof("Average Response Time Increased Too Much: Aborting Stresstest.")
			break
		}
		if result.AvgRT > stopART {
			infof("Average Response Time Long: Aborting Stresstest.")
			break
		}
		if result.MaxRT > stopMRT {
			infof("Maximum Response Time Long: Aborting Stresstest.")
			break
		}
		if result.Fail > int(stopFF*float64(result.Total)) {
			infof("To Many Failures: Aborting Stresstest.")
			break
		}

		lastRespTime = result.AvgRT
		load = stepper.Next(load)

		if load > stopMPR {
			infof("To Many Background Request: Aborting Stresstest.")
			break
		}

		time.Sleep(time.Duration(rampSleep) * time.Millisecond)
	}

	fmt.Print(stressChartUrl(data))
	fmt.Print(text)

	writeStressHistograms(data, name)
}

//  Perform stresstest.
func stresstest(bgfilename, testfilename string) {
	// Read background and test suite
	background, _, berr := readSuite(bgfilename)
	testsuite, _, serr := readSuite(testfilename)
	if berr != nil || serr != nil {
		errorf("Cannot parse given suites.")
		return
	}

	// Disable test which should not run by setting their Repeat to 0
	for i := 0; i < len(testsuite.Test); i++ {
		if !shouldRun(testsuite, 1, i+1) {
			warnf("Disabeling test %s", testsuite.Test[i].Title)
			testsuite.Test[i].Setting["Repeat"] = 0
		}
	}

	// perform increasing stresstests
	stressramp(background, testsuite, suite.ConstantStep{rampStart, rampStep}, testfilename)
}

// Generate Google chart for stresstest results.
func stressChartUrl(data []suite.StressResult) (url string) {
	url = "http://chart.googleapis.com/chart?cht=lxy&chs=500x300&chxs=0,676767,11.5,0,lt,676767&chxt=x,y,r"
	url += "&chls=2|2|2&chco=0000FF,00FF00,FF0000&chm=s,000000,0,-1,4|s,000000,1,-1,4|s,000000,2,-1,4"
	url += "&chdlp=b&chdl=Max+RT|Avg+RT|Err+Rate"

	// Determine maximum load and response time and round properly
	var mrt int64 = -1
	var mld int = -1
	for _, d := range data {
		if d.MaxRT > mrt {
			mrt = d.MaxRT
		}
		if d.Load > mld {
			mld = d.Load
		}
	}
	mrt = 100 * ((mrt + 99) / 100)
	mld = 10 * ((mld + 9) / 10)
	if mld < 10 {
		mld = 10
	}

	url += fmt.Sprintf("&chxr=1,0,%d|0,0,%d", mrt, mld)

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
		if d.Total == 0 {
			d.Total = 1
		}
		ld += fmt.Sprintf("%d", int(100*d.Load/mld))
		maxd += fmt.Sprintf("%d", int(100*d.MaxRT/mrt))
		avgd += fmt.Sprintf("%d", int(100*d.AvgRT/mrt))
		errd += fmt.Sprintf("%d", int(100*d.Fail/d.Total))
	}
	chd += ld + "|" + maxd + "|" + ld + "|" + avgd + "|" + ld + "|" + errd

	url += chd + "\n"
	return
}

const (
	svgWidth  = 300
	svgHeight = 200
	svgLeft   = 100
	svgTop    = 50
)

// helper to properly format the x-axis (response times)
func beautifulX(maxRT, step float64) (float64, float64) {
	// 4 to 5 intervals
	step = maxRT / 3
	ls := int(math.Log10(step))
	ra := step / math.Pow10(ls)
	if ra < 1.5 {
		ra = 1
	} else if ra < 2.25 {
		ra = 2
	} else if ra < 3.75 {
		ra = 2.5
	} else if ra < 7.5 {
		ra = 5
	} else {
		ra = 1
		ls++
	}
	step = ra * math.Pow10(ls)
	maxRT = math.Ceil(maxRT/step) * step

	return maxRT, step
}

// save multiplot of histograms to filename
func writeStressHistograms(data []suite.StressResult, name string) {
	numLoads := len(data)
	numTests := len(data[0].Detail)

	maxRT := float64(-1)
	for i := range data {
		for _, rt := range data[i].Detail {
			for _, t := range rt {
				if float64(t) > maxRT {
					maxRT = float64(t)
				}
			}
		}
	}
	step := maxRT / 4
	maxRT, step = beautifulX(maxRT, step)

	// A histogram
	now := time.Now()
	filename := outputPath + "stresstest_" + now.Format("2006-01-02_15-04-05") + ".svg"
	file, err := os.Create(filename)
	if err != nil {
		errorf("Cannot write to %q: %s", filename, err.Error())
		return
	}
	thesvg := svg.New(file)

	width, height := svgLeft+svgWidth*numTests, svgTop*svgHeight*numLoads+20
	thesvg.Start(width, height)
	thesvg.Title("Response Times")
	thesvg.Rect(0, 0, width, height, "fill: #ffffff")
	title := fmt.Sprintf("Distribution of response times in ms (suite: %s, date: %s)",
		name, now.Format("Mon 2. Jan. 2006 15:04:05"))
	thesvg.Text(20, svgTop/2, title, "",
		"text-anchor: begin; font-size: 20;")

	tests := []string{}
	k := 0
	for t, _ := range data[0].Detail {
		tests = append(tests, t)
		thesvg.Text(20+svgLeft+k*svgWidth, svgTop-2, t, "",
			"text-anchor: begin; font-size: 14;")
		k++
	}

	white := color.RGBA{255, 255, 255, 255}
	for i := range data {
		thesvg.Text(svgLeft-10, svgTop+svgHeight*i+0.5*svgHeight,
			fmt.Sprintf("%d || Req", data[i].Load), "",
			"text-anchor: end; font-size: 14;")
		for j, t := range tests {
			showTime, hd := false, 0
			if i == len(data)-1 {
				showTime = true
				hd += 20
			}
			thesvg.Gtransform(fmt.Sprintf("translate(%d %d)",
				svgLeft+svgWidth*j, svgTop+svgHeight*i))
			svggraphics := svgg.New(thesvg, svgWidth, svgHeight+hd, "Arial", 12, white)
			plotHistogram(svggraphics, data[i].Detail[t], maxRT, step, showTime)
			thesvg.Gend()
		}
	}
	thesvg.End()
	file.Close()
	warnf("Wrote stresstest histogram to file %s\n", filename)
}

func plotHistogram(g chart.Graphics, data []int, maxRT, step float64, showTime bool) {
	grey := color.RGBA{100, 100, 100, 255}
	style := chart.Style{LineColor: grey, FillColor: grey}
	var histogram chart.HistChart

	// Show frequencies
	histogram.Counts = false
	histogram.YRange.MinMode.Fixed = true
	histogram.YRange.MinMode.Value = 0
	histogram.YRange.MaxMode.Fixed = true
	histogram.YRange.MaxMode.Value = 100
	histogram.YRange.TicSetting.HideLabels = true
	histogram.YRange.TicSetting.Tics = 1
	histogram.YRange.TicSetting.Delta = 25

	histogram.Kernel = chart.BisquareKernel
	histogram.BinWidth = float64(maxRT) / 10
	histogram.AddDataInt("", data, style)

	histogram.XRange.MinMode.Fixed = true
	histogram.XRange.MinMode.Value = 0
	histogram.XRange.MaxMode.Fixed = true
	histogram.XRange.MaxMode.Value = maxRT
	histogram.XRange.TicSetting.Tics = 1
	histogram.XRange.TicSetting.Delta = step
	histogram.XRange.TicSetting.HideLabels = !showTime

	histogram.Plot(g)
}
