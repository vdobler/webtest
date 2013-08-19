package main

import (
	"testing"
)

func TestBeautifulX(t *testing.T) {
	for _, m := range []float64{95, 115, 130, 240, 567, 987, 1234, 1999, 2400, 2834, 5021} {
		max, step := beautifulX(m, m/4)
		t.Logf("%4.0f  -->  %d  %4.0f  %4.0f\n", m, int(max/step+1.5), step, max)
	}
}
