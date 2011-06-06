include $(GOROOT)/src/Make.inc

TARG=dobler/webtest/stat
GOFILES=\
	stat.go\

include $(GOROOT)/src/Make.pkg

format: $(GOFILES) stat_test.go 
	gofmt -w $^