include $(GOROOT)/src/Make.inc

TARG=github.com/vdobler/webtest/stat
GOFILES=\
	stat.go\

include $(GOROOT)/src/Make.pkg

format: $(GOFILES) stat_test.go 
	gofmt -w $^