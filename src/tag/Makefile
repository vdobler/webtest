include $(GOROOT)/src/Make.inc

TARG=dobler/webtest/tag
GOFILES=\
	tag.go\
	tagspec.go\
	parser.go\
	match.go\
	debug.go

include $(GOROOT)/src/Make.pkg

format: $(GOFILES) match_test.go parser_test.go tag_test.go tagspec_test.go
	gofmt -w $^