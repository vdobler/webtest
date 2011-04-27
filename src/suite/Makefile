
main.exe: main.8 suite.8
	8l -o main.exe main.8

main.8: main.go suite.8
	8g main.go
	
suite.8: suite.go variables.go condition.go http.go
	8g -o $@ $+

clean:
	rm -f *.8 *.exe

PHONY: clean
