include $(GOROOT)/src/Make.inc

TARG=github.com/vdobler/webtest/suite
GOFILES=\
	suite.go\
	test.go\
	condition.go\
	variables.go\
	parser.go\
	http.go\
	cookie.go

include $(GOROOT)/src/Make.pkg

format: $(GOFILES) suite_test.go parser_test.go condition_test.go
	gofmt -w $^