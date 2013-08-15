//
// Stat - Little helper package to Webtest with statistical functions.
//
// Copyright 2011 Volker Dobler. All rights reserved.
//
package stat

import (
	"math"
	"sort"
	"strings"

	"fmt"
	"net/url"
)

// Return p percentil of pre-sorted integer data. 0 <= p <= 100.
func PercentilInt(data []int, p int) int {
	n := len(data)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return data[0]
	}

	pos := float64(p) * float64(n+1) / 100
	fpos := math.Floor(pos)
	intPos := int(fpos)
	dif := pos - fpos
	if intPos < 1 {
		return data[0]
	}
	if intPos >= n {
		return data[n-1]
	}
	lower := data[intPos-1]
	upper := data[intPos]
	val := float64(lower) + dif*float64(upper-lower)
	return int(math.Floor(val + 0.5))
}

// Return p percentil of pre-sorted float64 data. 0 <= p <= 100.
func percentilFloat64(data []float64, p int) float64 {
	n := len(data)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return data[0]
	}

	pos := float64(p) * float64(n+1) / 100
	fpos := math.Floor(pos)
	intPos := int(fpos)
	dif := pos - fpos
	if intPos < 1 {
		return data[0]
	}
	if intPos >= n {
		return data[n-1]
	}
	lower := data[intPos-1]
	upper := data[intPos]
	val := lower + dif*(upper-lower)
	return val
}

// Compute minimum, p percentil, median, average, 100-p percentil and maximum of values in data.
func SixvalInt(data []int, p int) (min, lq, med, avg, uq, max int) {
	min, max = math.MaxInt32, math.MinInt32
	sum, n := 0, len(data)
	if n == 0 {
		return
	}
	if n == 1 {
		min = data[0]
		lq = data[0]
		med = data[0]
		avg = data[0]
		uq = data[0]
		max = data[0]
		return
	}
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

	sort.Ints(data)

	if n%2 == 1 {
		med = data[(n-1)/2]
	} else {
		med = (data[n/2] + data[n/2-1]) / 2
	}

	lq = PercentilInt(data, p)
	uq = PercentilInt(data, 100-p)
	return
}

func DistributionInt(data []int, levels []int) (p []int) {
	sort.Ints(data)
	for _, q := range levels {
		p = append(p,PercentilInt(data, q))
	}
	return p
}

// Compute minimum, p percentil, median, average, 100-p percentil and maximum of values in data.
func SixvalFloat64(data []float64, p int) (min, lq, med, avg, uq, max float64) {
	n := len(data)

	// Special cases 0 and 1
	if n == 0 {
		return
	}

	if n == 1 {
		min = data[0]
		lq = data[0]
		med = data[0]
		avg = data[0]
		uq = data[0]
		max = data[0]
		return
	}

	// First pass (min, max, coarse average)
	var sum float64
	min, max = math.MaxFloat64, -math.MaxFloat64
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}
	avg = sum / float64(n)

	// Second pass: Correct average
	var corr float64
	for _, v := range data {
		corr += v - avg
	}
	avg += corr / float64(n)

	// Median
	sort.Float64s(data)
	if n%2 == 1 {
		med = data[(n-1)/2]
	} else {
		med = (data[n/2] + data[n/2-1]) / 2
	}

	// Percentiles
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	lq = percentilFloat64(data, p)
	uq = percentilFloat64(data, 100-p)
	return
}

// Generate histogram via google charts api).
func HistogramChartUrlInt(d []int, title, label string) (url_ string) {
	url_ = "http://chart.googleapis.com/chart?cht=bvg&chs=600x300&chxs=0,676767,11.5,0,lt,676767&chxt=x&chdlp=b"
	url_ += "&chbh=a&chco=404040&chtt=" + url.QueryEscape(strings.Trim(title, " \t\n"))
	url_ += "&chdl=" + url.QueryEscape(strings.Trim(label, " \t\n"))

	// Decide on number of bins
	min, _, _, _, _, max := SixvalInt(d, 25)
	cnt := 10
	if len(d) <= 10 {
		cnt = 3
	} else if len(d) <= 15 {
		cnt = 5
	} else if len(d) > 40 {
		cnt = 15
	}
	step := float64(max-min) / float64(cnt) // inital

	// make step multiple of 2, 5 or 10
	pow := 0
	for step > 10 {
		pow++
		step /= 10
	}
	var width int
	switch true {
	case step < 1.5:
		width = 1
	case step < 3:
		width = 2
	case step < 8:
		width = 5
	default:
		width = 10
	}
	for ; pow > 0; pow-- {
		width *= 10
	}
	low := (min / width) * width
	high := (max/width + 1) * width
	cnt = (high - low) / width

	// Binify and scale largest bar to 100
	var bin []int = make([]int, cnt)
	mc := 0
	for _, n := range d {
		b := (n - low) / width
		if b < 0 {
			b = 0
		} else if b >= cnt {
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
	url_ += fmt.Sprintf("&chxr=0,%d,%d", low+width/2, high-width/2)
	url_ += "&chd=t:"
	for i, n := range bin {
		if i > 0 {
			url_ += ","
		}
		url_ += fmt.Sprintf("%d", n)
	}
	return
}
