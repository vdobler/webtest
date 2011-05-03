include $(GOROOT)/src/Make.inc

TARG=webtest
GOFILES=\
	webtest.go\

DEFAULT: webtest.exe
	
	
PACKAGES:
	cd tag && $(MAKE) install
	cd suite && $(MAKE) install
	
CLEAN:
	cd tag && $(MAKE) clean 
	cd suite && $(MAKE) clean 
	$(MAKE) clean 
	
	
include $(GOROOT)/src/Make.cmd
