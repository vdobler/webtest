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

doc: reference-suite.pdf webtest.pdf

%.pdf: %.wt
	cd format && $(MAKE)
	./format/wtformat $<
	pdflatex "\batchmode\input{$<.tex}"
	pdflatex "\batchmode\input{$<.tex}"
	pdflatex "\batchmode\input{$<.tex}"

todo:
	grep -n TODO `find . -name "*.go" | sort`
