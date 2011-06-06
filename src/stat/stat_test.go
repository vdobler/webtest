package stat

import (
	"testing"
	"fmt"
)

func TestIntSixval(t *testing.T) {
	min, q1, med, avg, q3, max := SixvalInt([]int{1, 3, 4, 2}, 25)
	if min != 1 || max != 4 || med != 2 || avg != 2 || q1 != 1 || q3 != 4 {
		t.Error("1, 2, 3, 4 wrong")
	}
}

func TestFloatSixval(t *testing.T) {
	min, q1, med, avg, q3, max := SixvalFloat64([]float64{1, 3, 4, 2}, 25)
	if min != 1 || max != 4 || med != 2.5 || avg != 2.5 || q1 != 1.25 || q3 != 3.75 {
		t.Error("1, 2, 3, 4 wrong")
		fmt.Printf("%.3f  %.3f  %.3f  %.3f  %.3f  %.3f  \n", min, q1, med, avg, q3, max)
	}
}
