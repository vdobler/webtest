package suite

import (
	"fmt"
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
		d, c string
	}{{"", "fffffffffffffffffffffffffffff"},
		{"+1hour", "cccffccfcccfccccfcCffffffffff"},
		{"+10 hours", "cccffccfcccfccccfCcffffffffff"},
		{"+2 days", "cccffcCfcccfccccfffffffffffff"},
		{"+40days", "cccffcCfCCCfccccfffffffffffff"},
		{"+10days - 2hours + 10 seconds", "CCC, Cc ccc cccc cC:cc:Cf fff"},
		{"+ 1 month", "ccc, ff CCC cccc ff:ff:ff fff"},
		{"+ 12 month", "ccc, ff fff cccC ff:ff:ff fff"},
		{"+ 13 month", "ccc, ff CCC cccC ff:ff:ff fff"},
		{"- 4 months", "ccc, ff CCC cccc ff:ff:ff fff"},
		{"- 13 month", "ccc, ff CCC cccC ff:ff:ff fff"},
		{"+ 1 year", "ccc, ff fff cccC ff:ff:ff fff"},
		{"+ 12 year", "ccc, ff fff ccCC ff:ff:ff fff"},
		{"- 11 years", "ccc, ff fff ccCC ff:ff:ff fff"},
	}
	for _, x := range testNowValues {
		now := nowValue("", http.TimeFormat, true)
		then := nowValue(x.d, http.TimeFormat, true)
		for i, m := range x.c {
			switch m {
			case 'f':
				if now[i] != then[i] {
					t.Errorf("'%s' %s: Pos %d: got '%c' expected '%c'.",
						now, x.d, i, now[i], then[i])
				}
			case 'C':
				if now[i] == then[i] {
					t.Errorf("'%s' '%s': Pos %d: unchanged '%c': %s",
						now, x.d, i, now[i], then)
				}
			default:
				// might change
			}
		}
		if t.Failed() {
			fmt.Printf("'%s' %s --> '%s'\n", now, x.d, then)
		}
	}
}
