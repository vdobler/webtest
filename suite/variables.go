package suite

import (
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vdobler/webtest/tag"
)

// Random for RAND sections
var Random *rand.Rand = rand.New(rand.NewSource(time.Now().Unix()))

// Global CONST variables
var Const map[string]string = map[string]string{}

func isLetter(x uint8) bool {
	return (x >= 'a' && x <= 'z') || (x >= 'A' && x <= 'Z')
}
func isDigit(x uint8) bool {
	return (x >= '0' && x <= '9')
}

// Split str into "pre${vn}rest" parts
func nextPart(str string) (pre, vn, rest string) {
	i := strings.Index(str, "${")
	if i == -1 || i >= len(str)-3 {
		pre = str
		return
	}
	pre, str = str[:i], str[i+2:]
	var j int
	if strings.HasPrefix(str, "NOW") && len(str) > 3 && !isLetter(str[3]) {
		j = strings.Index(str, "}")
	} else {
		for j = 0; j < len(str) && isLetter(str[j]); j++ {
		}
		if j == len(str) || str[j] != '}' {
			j = -1
		}
	}
	if j == -1 {
		pre += "${" + str
		return
	}
	vn, rest = str[:j], str[j+1:]
	return
}

// Choose a random one of list.
func randomVar(list []string) string {
	n := len(list)
	r := Random.Intn(n)
	tracef("Will use %d from list of %d.", r, n)
	return list[r]
}

// Retrieve the next value in list (wrapping around). The counter to decide which is the next is taken from t.
func nextVar(list []string, v string, t *Test) (val string) {
	i, _ := t.SeqCnt[v]
	val = list[i]
	i++
	i = i % len(list)
	t.SeqCnt[v] = i
	return
}

// Compute next value for variable v, fallback to global.
func varValueFallback(v string, test, global, orig *Test) (value string) {
	if val, ok := Const[v]; ok {
		value = val
	} else if val, ok := test.Const[v]; ok {
		value = val
	} else if val, ok := global.Const[v]; ok {
		value = val
	} else if rnd, ok := test.Rand[v]; ok {
		value = randomVar(rnd)
	} else if rnd, ok := global.Rand[v]; ok {
		value = randomVar(rnd)
	} else if seq, ok := test.Seq[v]; ok {
		value = nextVar(seq, v, orig)
	} else if seq, ok := global.Seq[v]; ok {
		value = nextVar(seq, v, global)
	} else {
		errorf("Cannot find value for variable '%s'!", v)
	}

	return
}

// Get value for variable v.
func varValue(v string, test, orig *Test) (value string) {
	if val, ok := test.Const[v]; ok {
		value = val
		tracef("Using const '%s' for var '%s'.", val, v)
	} else if rnd, ok := test.Rand[v]; ok {
		value = randomVar(rnd)
		tracef("Using random '%s' for var '%s'.", value, v)
	} else if seq, ok := test.Seq[v]; ok {
		value = nextVar(seq, v, orig)
	} else {
		errorf("Cannot find value for variable '%s'!", v)
	}

	return
}

// Compute value for NOW variable adjusted by rel (e.g. "+1minute-3hours") and return it as utc/local
// time formated as tf.
func nowValue(rel, tf string, utc bool) string {
	rel = strings.Replace(rel, " ", "", -1)

	rx := regexp.MustCompile("([+\\-])([0-9]+)(second|minute|hour|day|week|month|year)s?")
	all := rx.FindAllStringSubmatch(rel, -1)

	var ds, dm, dy int64 // total delta for seconds, month and years
	for _, delta := range all {
		n, _ := strconv.ParseInt(delta[2], 10, 64)
		if delta[1] == "-" {
			n = -n
		}
		switch delta[3] {
		case "second":
			ds += n
		case "minute":
			ds += n * 60
		case "hour":
			ds += n * 3600
		case "day":
			ds += n * 24 * 3600
		case "week":
			ds += n * 7 * 24 * 3600
		case "month":
			dm += n
		case "year":
			dy += n
		default:
			errorf("Oooops: Unknown NOW modifier '%s'. Ignored.", delta[3])
		}
	}
	dy, dm = dy+(dm/12), dm%12
	t := time.Now().Add(time.Duration(ds) * time.Second)
	loc := time.UTC
	if !utc {
		loc = t.Location()
	}
	t = time.Date(t.Year()+int(dy), t.Month()+time.Month(dm), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)

	return t.Format(tf)
}

// Substitute variables in str with their right values.
func substitute(str string, test, global, orig *Test) string {
	pre, vn, post := nextPart(str)
	if vn == "" {
		return pre
	}

	var val string
	if v, ok := orig.Vars[vn]; ok {
		val = v
		tracef("Reusing '%s' for var '%s'.", val, vn)
	} else {
		if strings.HasPrefix(vn, "NOW") && (len(vn) == 3 || !isLetter(vn[3])) {
			tf := http.TimeFormat
			s := strings.Trim(vn[3:], " \t")
			if i := strings.Index(s, "|"); i != -1 {
				tf = strings.Trim(s[i+1:], " ")
				s = strings.Trim(s[:i], " ")
			}
			val = nowValue(s, tf, true)
		} else {
			if global != nil {
				val = varValueFallback(vn, test, global, orig)
			} else {
				val = varValue(vn, test, orig)
			}
		}
		orig.Vars[vn] = val // Save value for further use in this test
	}
	tracef("Will use '%s' as value for var %s.", val, vn)
	return pre + val + substitute(post, test, global, orig)
}

// Special function to dig into nested tags and replace variables with their value there too.
func substituteTagContent(ts *tag.TagSpec, test, global, orig *Test) {
	if ts.Content != nil {
		ocs := ts.Content.String()
		ncs := substitute(ocs, test, global, orig)
		if ncs != ocs {
			if nc, err := tag.MakeContent(ncs); err == nil {
				ts.Content = nc
			} else {
				errorf("Tag AA text content or attribute value is malformed after variable substitution! %s\nocs=%s\nncs=%s", err.Error(), ocs, ncs)
			}
		}
	}
	for _, sub := range ts.Sub {
		substituteTagContent(sub, test, global, orig)
	}
}

// Replace all variables in test with their appropriate values.
func substituteVariables(test, global, orig *Test) {
	// test.Url = substitute(test.Url, test, global, orig) done in prepare test

	for k, v := range test.Header {
		test.Header[k] = substitute(v, test, global, orig)
	}
	for i, c := range test.RespCond {
		test.RespCond[i].Val = substitute(c.Val, test, global, orig)
	}
	for i, c := range test.BodyCond {
		test.BodyCond[i].Val = substitute(c.Val, test, global, orig)
	}
	for i, c := range test.CookieCond {
		test.CookieCond[i].Val = substitute(c.Val, test, global, orig)
	}

	for k, vl := range test.Param {
		tracef("Param %s: %v", k, vl)
		sl := make([]string, len(vl))
		for i, v := range vl {
			sl[i] = substitute(v, test, global, orig)
		}
		test.Param[k] = sl
	}

	tracef("Replacing tag content")
	for i, tc := range test.Tag {
		if tc.Spec.Content != nil {
			ocs := tc.Spec.Content.String()
			ncs := substitute(ocs, test, global, orig)
			if ocs != ncs {
				if nc, err := tag.MakeContent(ncs); err == nil {
					test.Tag[i].Spec.Content = nc
				} else {
					errorf("Tag text content or attribute value is malformed after variable substitution! %s", err.Error())
				}
			}
		}
		for j, subts := range tc.Spec.Sub {
			tracef("Replacing sub tag content %d", j)
			substituteTagContent(subts, test, global, orig)
		}
	}
}
