package suite

import (
	"strings"
	"rand"
	"time"
	"dobler/webtest/tag"
)

// Random for RAND sections
var Random *rand.Rand = rand.New(rand.NewSource(time.Seconds()))

// Global CONST variables
var Const map[string]string = map[string]string{}


func isLetter(x uint8) bool {
	return (x >= 'a' && x <= 'z') || (x >= 'A' && x <= 'Z')
}

// Return a list of all occurences of all variables in str.
func usedVars(str string) (vars []string) {
	m := len(str) - 1
	for i := 0; i < m; i++ {
		if str[i] == '$' && str[i+1] == '{' {
			a := i + 2
			for i = a; i < m && isLetter(str[i]); i++ {
			}
			// debug("Start %d, End %d, Name %s", a, i, str[a:i])
			if str[i] == '}' {
				vars = append(vars, str[a:i])
			}
		}
	}
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


func varValueFallback(v string, test, global, orig *Test) (value string) {
	if val, ok := orig.Vars[v]; ok {
		value = val
		trace("Reusing '%s' for var '%s'.", val, v)
	} else if val, ok := Const[v]; ok {
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

	// Save value
	orig.Vars[v] = value
	return
}

func varValue(v string, test, orig *Test) (value string) {
	if val, ok := orig.Vars[v]; ok {
		value = val
		trace("Reusing '%s' for var '%s'.", val, v)
	} else if val, ok := test.Const[v]; ok {
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

	// Save value
	orig.Vars[v] = value
	return
}

// Subsitute all variables in str with their appropriate values.
// If global is non nil, than global woll be used as fallback if test
// does not provide the variable.
func substitute(str string, test, global, orig *Test) string {
	used := usedVars(str)
	for _, v := range used {
		var val string
		if global != nil {
			val = varValueFallback(v, test, global, orig)
		} else {
			val = varValue(v, test, orig)
		}
		trace("Will use '%s' as value for var %s.", val, v)
		str = strings.Replace(str, "${"+v+"}", val, 1)
	}
	if len(used) > 0 {
		trace("Substituted %d variables: '%s'.", len(used), str)
	}
	return str
}

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
