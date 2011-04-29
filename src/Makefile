
webtest.exe: webtest.8 
	8l -o webtest.exe  webtest.8 

webtest.8: webtest.go 
	cd tag && $(MAKE) tag.8
	cd suite && $(MAKE) suite.8
	8g webtest.go
	
clean:
	cd tag && $(MAKE) clean
	cd suite && $(MAKE) clean
	rm -f *.8 *.exe

PHONY: clean
