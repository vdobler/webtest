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
		Header: map[string]string{
			"User-Agent":      "Mozilla/5.0 (Windows; U; Windows NT 6.1; en-US; ${varR}/${varS}) Gecko/20110319 Firefox/3.6.16",
			"Accept-Language": "en-us,en;q=0.5",
		},
		RespCond: []suite.Condition{suite.Condition{Key: "StatusCode", Op: "==", Val: "200"},
			suite.Condition{Key: "Url", Op: "_=", Val: "http://www.unic.com/${varR}/${varS}", Neg: true}},
		BodyCond: []suite.Condition{suite.Condition{Key: "Txt", Op: "~=", Val: "e"}},
		Const:    map[string]string{},
		Rand:     map[string][]string{},
		Seq:      map[string][]string{},
	}

	test := suite.Test{Title: "Demo Test", Method: "GET", Url: "${UNIC}",
		RespCond: []suite.Condition{suite.Condition{Key: "StatusCode", Op: ">=", Val: "100"},
			suite.Condition{Key: "Url", Op: "_=", Val: "${UNIC}/ch/de.html"}},
		BodyCond: []suite.Condition{suite.Condition{Key: "Txt", Op: "~=", Val: "Unic AG"},
			suite.Condition{Key: "Tag", Val: "a href=/ch/de/leistungen.${varR}.html == Leistungen ${varS}", Neg: true},
			suite.Condition{Key: "Tag", Val: "h2 class=home == Qualität für Sie!"}},
		Const: map[string]string{"UNIC": "http://www.unic.com"},
		Rand:  map[string][]string{"varR": []string{"AA", "BB", "CC", "DD", "EE"}},
		Seq:   map[string][]string{"varS": []string{"first ", "second", "third", "forth"}},
		Param: map[string]string{"Repeat": "4", "Sleep": "1200"},
	}

	pdf := suite.Test{Title: "PDF (Binary) Test", Method: "GET",
		Url:      "http://www.fin.be.ch/fin/de/index/steuern/steuererklaerung/publikationen/merkblaetter.assetref/content/dam/documents/FIN/SV/de/Merkblaetter/Einkommens_Vermoegenssteuer/Aktuelles_Steuerjahr/mb_ev_erben-und-miteigentuemergemeinschaften_de.pdf",
		RespCond: []suite.Condition{suite.Condition{Key: "StatusCode", Op: "==", Val: "200"}},
		BodyCond: []suite.Condition{suite.Condition{Key: "Bin", Op: "_=", Val: "255044462d312e360d25e2e3cfd30d"}},
		Const:    map[string]string{},
		Rand:     map[string][]string{},
		Seq:      map[string][]string{},
		Param:    map[string]string{"Repeat": "1"},
	}

	// h2 class=home == Qualität für Sie!
	suite := suite.Suite{Test: []suite.Test{global, test, pdf}}
	suite.RunTest(1)
	pdf.Run(nil)
}
