
main.exe: main.8 tag.8
	8l -o main.exe main.8

main.8: main.go tag.8
	8g main.go
	
tag.8: tagspec.go tag.go parser.go
	8g -o $@ $+

clean:
	rm -f *.8 *.exe

PHONY: clean
