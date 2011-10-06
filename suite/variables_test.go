package suite

import (
	"http"
	"testing"
)


func TestNextPart(t *testing.T) {
	var nextPartER [][4]string = [][4]string{[4]string{"Hallo", "Hallo", "", ""},
		[4]string{"Hallo ${abc}", "Hallo ", "abc", ""},
		[4]string{"Hallo ${abc", "Hallo ${abc", "", ""},
		[4]string{"Hallo ${abc.}", "Hallo ${abc.}", "", ""},
		[4]string{"Hallo ${a} du", "Hallo ", "a", " du"},
		[4]string{"Hallo ${abc} du ${da} welt", "Hallo ", "abc", " du ${da} welt"},
		[4]string{"${xyz}", "", "xyz", ""},
		[4]string{"${xyz} 123", "", "xyz", " 123"},
		[4]string{"Time ${NOW +3minutes-1hour+12days} UTC", "Time ", "NOW +3minutes-1hour+12days", " UTC"},
	}
	for _, exp := range nextPartER {
		pre, vn, post := nextPart(exp[0])
		// fmt.Printf("%s:\n", exp[0])
		if pre != exp[1] || vn != exp[2] || post != exp[3] {
			t.Error("Expected " + exp[0] + ", " + exp[1] + ", " + exp[2] + " but got " + pre + ", " + vn + ", " + post)
		}
	}
}

func TestNowValue(t *testing.T) {
	type tft struct {
		f string
		i int
	}
	// Fri, 03 Jun 2011 21:20:05 UTC
	//           1         2
	// 01234567890123456789012345678
	testNowValues := []struct {
		d string
		i int
	}{{"", 0},
		{"+1hour", 19},
		{"+10 hours", 18},
		{"+2 days", 11},
		{"+40days", 11},
		{"+10days - 2hours + 10 seconds", 24},
		{"+ 1 month", 11},
		{"+ 12 month", 16},
		{"+ 13 month", 16},
		{"- 4 months", 16},
		{"- 13 month", 16},
		{"+ 1 year", 16},
		{"+ 12 year", 16},
		{"- 11 years", 16},
	}
	for _, x := range testNowValues {
		now := nowValue("", http.TimeFormat, true)
		then := nowValue(x.d, http.TimeFormat, true)
		if !(now[x.i:] == then[x.i:]) {
			t.Error(now + " " + x.d + " unexpected " + then)
		}
	}
}

