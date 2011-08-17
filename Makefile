include $(GOROOT)/src/Make.inc

TARG=webtest
GOFILES=\
	webtest.go\

PACKAGES:
	cd tag && $(MAKE) install
	cd suite && $(MAKE) install
	cd stat && $(MAKE) install

CLEAN:
	cd tag && $(MAKE) clean 
	cd suite && $(MAKE) clean 
	cd stat && $(MAKE) clean
	$(MAKE) clean 


include $(GOROOT)/src/Make.cmd

format: $(GOFILES)
	gofmt -w $^
	cd tag && $(MAKE) format
	cd suite && $(MAKE) format
	cd stat && $(MAKE) format