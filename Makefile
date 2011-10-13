include $(GOROOT)/src/Make.inc

TARG=webtest
GOFILES=\
	webtest.go\

SUBPACKAGES=tag suite stat format

PACKAGES:
	c=`pwd`; for d in $(SUBPACKAGES); do cd $$c/$$d && $(MAKE) install; done

CLEAN:
	c=`pwd`; for d in $(SUBPACKAGES); do cd $$c/$$d && $(MAKE) clean; done
	$(MAKE) clean 


include $(GOROOT)/src/Make.cmd

format: $(GOFILES)
	c=`pwd`; for d in $(SUBPACKAGES); do cd $$c/$$d && $(MAKE) format; done
	gofmt -w $^

doc: reference-suite.wt
	cd format && $(MAKE)
	./format/wtformat reference-suite.wt
	pdflatex "\batchmode\input{reference-suite.wt}"
	pdflatex "\batchmode\input{reference-suite.wt}"
	pdflatex "\batchmode\input{reference-suite.wt}"

todo:
	grep -n TODO `find . -name "*.go" | sort`
