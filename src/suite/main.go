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
							"User-Agent": "Mozilla/5.0 (Windows; U; Windows NT 6.1; en-US; rv:1.9.2.16) Gecko/20110319 Firefox/3.6.16",
							"Accept-Language": "en-us,en;q=0.5",
						 },
						 RespCond: []*suite.Condition{ &suite.Condition{Key: "StatusCode", Op: "==", Val: "200"},
													   &suite.Condition{Key: "Url", Op: "_=", Val: "http://www.unic.com/"}},
						 Repeat: 1}

	test := suite.Test{Title: "Demo Test", Method: "GET", Url: "http://www.unic.com",
					   BodyCond: []*suite.Condition{ &suite.Condition{Key: "Text", Op: "~=", Val: "Unic Super AG"}, 
												     &suite.Condition{Key: "Tag", Val: "a href=/ch/de/leistungen.html == Leistungen"}} }

	suite := suite.Suite{Test: []*suite.Test{&global, &test}}
	suite.RunTest(1)
}