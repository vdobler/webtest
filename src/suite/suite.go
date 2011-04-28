package suite

import (
	"fmt"
	"log"
)

var logLevel int = 3 // 0: none, 1:err, 2:warn, 3:info, 4:debug, 5:trace


func error(f string, m ...interface{}) {
	if logLevel >= 1 {
		log.Print("*ERROR* " + fmt.Sprintf(f, m...))
	}
}
func warn(f string, m ...interface{}) {
	if logLevel >= 2 {
		log.Print("*WARN * " + fmt.Sprintf(f, m...))
	}
}
func info(f string, m ...interface{}) {
	if logLevel >= 3 {
		log.Print("*INFO * " + fmt.Sprintf(f, m...))
	}
}
func debug(f string, m ...interface{}) {
	if logLevel >= 4 {
		log.Print("*DEBUG* " + fmt.Sprintf(f, m...))
	}
}
func trace(f string, m ...interface{}) {
	if logLevel >= 5 {
		log.Print("*TRACE* " + fmt.Sprintf(f, m...))
	}
}


// TODO: Results?
type Suite struct {
	Test   []Test
	Result map[string]int // 0: not run jet, 1: pass, 2: fail, 3: err
}

func (s *Suite) RunTest(n int) {
	if n < 0 || n >= len(s.Test) {
		error("No such test")
	}

	global := &s.Test[0] // TODO use if correct

	s.Test[n].Run(global)

}
