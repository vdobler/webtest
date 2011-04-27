package suite

import (
	"strings"
	"rand"
)

var Random *rand.Rand

func init() {
	Random = rand.New(rand.NewSource(12345))
}


func isLetter(x uint8) bool {
	return (x >= 'a' && x <= 'z') || (x >= 'A' && x <= 'Z')
}

func usedVars(str string) (vars []string) {
	m := len(str)-1
	for i:=0; i<m; i++ {
		if str[i] == '$' && str[i+1] == '{' {
			a := i+2
			for i=a; i<m && isLetter(str[i]); i++ { } 
			// debug("Start %d, End %d, Name %s", a, i, str[a:i])
			if str[i] == '}' {
				vars = append(vars, str[a:i])
			}
		}
	}
	return
}

func randomVar(list []string) string {
	n := len(list)
	return list[Random.Intn(n)]
}

func nextVar(list []string, v string, t *Test) (val string) {
	/*
	if t.SeqCnt == nil {
		t.SeqCnt = make(map[string] int, len(t.Seq))
		for k, _ := range t.Seq {
			t.SeqCnt[k] = 0
		}
	}
	*/
	i, _ := t.SeqCnt[v]; 
	val = list[i]
	i++
	i = i % len(list)
	t.SeqCnt[v] = i
	return
}

func varValue(v string, test, global, orig *Test) (value string) {
	if val, ok := orig.Vars[v]; ok {
		value = val
		trace("Reusing '%s' for var '%s'.", val, v)
	} else 	if val, ok := test.Const[v]; ok {  
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
		value = nextVar(seq, v, orig) 
	} else {
		error("Cannot find value for variable '%s'!", v)
	}
	
	// Save value
	orig.Vars[v] = value
	
	return 
}


func substitute(str string, test, global, orig *Test) string {
	trace("Substitute '%s'", str)
	used := usedVars(str)
	for _, v := range used {
		val := varValue(v, test, global, orig)
		trace("Will use '%s' as value for var %s.", val, v)
		str = strings.Replace(str, "${" + v + "}", val, 1)
		trace("str now '%s'", str)
	}
	trace("Substituted %d variables: '%s'.", len(used), str)
	return str
}

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
}
