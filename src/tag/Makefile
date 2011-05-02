
main.exe: main.8 tag.8
	8l -o main.exe main.8

main.8: main.go tag.8
	8g main.go
	
tag.8: tagspec.go tag.go parser.go
	8g -o $@ $+

test: _test/main.exe testdir
	_test/main.exe
	
testdir:
	mkdir -p _test
	
_test/main.exe: _test/main.8
	8l -o $@ $+
	
_test/main.8: tag_test.go tag.8
	8g -o _test/main.8 tag_test.go


	
clean:
	rm -f *.8 *.exe

PHONY: clean
