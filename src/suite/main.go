package main

import (
	// "fmt"
	// "http"
	// "html"
	_ "strings"
	"./suite"
)
 

func main() {
	global := suite.Test{Title: "Global",
						 Header: map[string] string {
							"User-Agent": "Mozilla/5.0 (Windows; U; Windows NT 6.1; en-US; ${varR}/${varS}) Gecko/20110319 Firefox/3.6.16",
							"Accept-Language": "en-us,en;q=0.5",
						 },
						 RespCond: []suite.Condition{ suite.Condition{Key: "StatusCode", Op: "==", Val: "200"},
													  suite.Condition{Key: "Url", Op: "_=", Val: "http://www.unic.com/${varR}/${varS}", Neg:true}},
					     BodyCond: []suite.Condition{ suite.Condition{Key: "Text", Op: "~=", Val: "e"}},
					     Const: map[string]string{},
					     Rand: map[string][]string{},
					     Seq: map[string][]string{},
						 Repeat: 1}

	test := suite.Test{Title: "Demo Test", Method: "GET", Url: "http://www.unic.com/${varC}/${varR}/${varS}/more/${varR}/${varS}/extra/${varR}/${varS}",
					   RespCond: []suite.Condition{ suite.Condition{Key: "StatusCode", Op: ">=", Val: "100"}},
					   BodyCond: []suite.Condition{ suite.Condition{Key: "Text", Op: "~=", Val: "Unic AG"}, 
												    suite.Condition{Key: "Tag", Val: "a href=/ch/de/leistungen.${varR}.html == Leistungen ${varS}", Neg:true}},
					   Const: map[string]string{ "varC": "Super!" },
					   Rand: map[string][]string{ "varR": []string{"AA", "BB", "CC", "DD", "EE"} },
					   Seq: map[string][]string{ "varS": []string{"first", "second", "third", "forth"} },
					   Repeat: 2,
					  }

	suite := suite.Suite{Test: []suite.Test{global, test}}
	suite.RunTest(1)
}