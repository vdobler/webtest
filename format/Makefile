include $(GOROOT)/src/Make.inc

TARG=wtformat
GOFILES=\
	wtformat.go\


include $(GOROOT)/src/Make.cmd

format: $(GOFILES)
	gofmt -w $^