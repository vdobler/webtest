package suite

import (
	"strings"
	"rand"
	"time"
	"dobler/webtest/tag"
	"regexp"
	"strconv"
	//	"fmt"
)

// Random for RAND sections
var Random *rand.Rand = rand.New(rand.NewSource(time.Seconds()))

// Global CONST variables
var Const map[string]string = map[string]string{}


func isLetter(x uint8) bool {
	return (x >= 'a' && x <= 'z') || (x >= 'A' && x <= 'Z')
}
func isDigit(x uint8) bool {
	return (x >= '0' && x <= '9')
}

type Variable struct {
	Name  string
	Start int
	End   int
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
		j = strings.Index(str, "}") // TODO: read variables or NOW only
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
	trace("Will use %d from list of %d.", r, n)
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
		error("Cannot find value for variable '%s'!", v)
	}

	return
}

// Get value for variable v.
func varValue(v string, test, orig *Test) (value string) {
	if val, ok := test.Const[v]; ok {
		value = val
		trace("Using const '%s' for var '%s'.", val, v)
	} else if rnd, ok := test.Rand[v]; ok {
		value = randomVar(rnd)
		trace("Using random '%s' for var '%s'.", value, v)
	} else if seq, ok := test.Seq[v]; ok {
		value = nextVar(seq, v, orig)
	} else {
		error("Cannot find value for variable '%s'!", v)
	}

	return
}


// Compute value for NOW variable adjusted by rel (e.g. "+1minute-3hours") and return it as utc/local 
// time formated as tf.
func nowValue(rel, tf string, utc bool) string {
	rel = strings.Replace(rel, " ", "", -1)

	rx, err := regexp.Compile("([+\\-])([0-9]+)(second|minute|hour|day|week|month|year)s?")
	if err != nil {
		error("Ooooops: " + err.String())
	}
	all := rx.FindAllStringSubmatch(rel, -1)

	var ds, dm, dy int64 // total delta for seconds, month and years
	for _, delta := range all {
		n, _ := strconv.Atoi64(delta[2])
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
			error("Oooops: %s", delta[3])
		}
	}
	var t *time.Time
	if utc {
		t = time.SecondsToUTC(time.UTC().Seconds() + ds)
	} else {
		t = time.SecondsToLocalTime(time.LocalTime().Seconds() + ds)
	}

	dy, dm = dy+(dm/12), dm%12
	t.Month += int(dm)
	t.Year += dy

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
		trace("Reusing '%s' for var '%s'.", val, vn)
	} else {
		if strings.HasPrefix(vn, "NOW") && (len(vn) == 3 || !isLetter(vn[3])) {
			tf := time.RFC1123
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
	trace("Will use '%s' as value for var %s.", val, vn)
	return pre + val + substitute(post, test, global, orig)
}


// Special function to dig into nested tags and replace variables with their value there too.
func substituteTagContent(ts *tag.TagSpec, test, global, orig *Test) {
	ts.Content = substitute(ts.Content, test, global, orig)
	for _, sub := range ts.Sub {
		substituteTagContent(sub, test, global, orig)
	}
}

// Replace all variables in test with their appropriate values.
func substituteVariables(test, global, orig *Test) {
	test.Url = substitute(test.Url, test, global, orig)
	for k, v := range test.Header {
		test.Header[k] = substitute(v, test, global, orig)
	}
	for i, c := range test.RespCond {
		test.RespCond[i].Val = substitute(c.Val, test, global, orig)
	}
	for i, c := range test.BodyCond {
		test.BodyCond[i].Val = substitute(c.Val, test, global, orig)
	}

	for k, vl := range test.Param {
		trace("Param %s: %v", k, vl)
		sl := make([]string, len(vl))
		for i, v := range vl {
			sl[i] = substitute(v, test, global, orig)
		}
		test.Param[k] = sl
	}

	for i, tc := range test.Tag {
		trace("Replacing tag content %d", i)
		test.Tag[i].Spec.Content = substitute(tc.Spec.Content, test, global, orig)
		for _, subts := range tc.Spec.Sub {
			substituteTagContent(subts, test, global, orig)
		}
	}
}
