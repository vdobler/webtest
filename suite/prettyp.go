package suite

import (
	"fmt"
	"strings"
)

func needQuotes(s string, containedSpacesNeedQuotes bool) bool {
	if containedSpacesNeedQuotes && strings.Contains(s, " ") {
		return true
	}
	if strings.HasPrefix(s, " ") || strings.HasPrefix(s, " ") {
		return true
	}

	for _, c := range s {
		if c == '"' {
			return true
		}
		if c < ' ' || c > '~' {
			return true
		}
	}
	return false
}

func quote(s string, containedSpacesNeedQuotes bool) string {
	if !needQuotes(s, containedSpacesNeedQuotes) {
		return s
	}
	return fmt.Sprintf("%#v", s)
}

// Prety print a map m with title. 
func formatMap(title string, m *map[string]string) (f string) {
	if len(*m) == 0 {
		return
	}

	f = title + "\n"
	longest := 0
	for k, _ := range *m {
		if len(k) > longest {
			longest = len(k)
		}
	}
	for k, v := range *m {
		f += fmt.Sprintf("\t%-*s  :=  %s\n", longest, k, quote(v, false))
	}
	return
}

func formatSettings(m *map[string]int) (f string) {
	if len(*m) == 0 {
		return
	}

	f = "SETTING\n"
	longest := 0
	for k, _ := range *m {
		if len(k) > longest {
			longest = len(k)
		}
	}
	for k, v := range *m {
		f += fmt.Sprintf("\t%-*s  :=  ", longest, k)
		switch k {
		case "Dump":
			switch v {
			case 0:
				f += "false"
			case 1:
				f += "create"
			case 2:
				f += "append"
			case 3:
				f += "body"
			default:
				f += fmt.Sprintf("%d", v)
			}
		case "Validate":
			switch v {
			case 0:
				f += "false"
			case 1:
				f += "links"
			case 2:
				f += "html"
			case 3:
				f += "links+html"
			default:
				f += fmt.Sprintf("%d", v)
			}

		default:
			f += fmt.Sprintf("%d", v)
		}
		f += "\n"
	}

	return
}

// Pretty print a multi-map m.
func formatMultiMap(title string, m *map[string][]string) (f string) {
	if len(*m) > 0 {
		f = title + "\n"
		longest := 0
		for k, _ := range *m {
			if len(k) > longest {
				longest = len(k)
			}
		}
		for k, l := range *m {
			f += fmt.Sprintf("\t%-*s  :=  ", longest, k)
			for _, v := range l {
				f += " " + quote(v, true)
			}
			f += "\n"
		}
	}
	return
}

// Pretty print a list of Conditions m.
func formatCond(title string, m *[]Condition) (f string) {
	if len(*m) == 0 {
		return
	}
	f = title + "\n"
	longest := 0
	for _, c := range *m {
		k := c.Key + c.Range.String()
		if len(k) > longest {
			longest = len(k)
		}
	}
	for _, c := range *m {
		if c.Neg {
			f += "\t!"
		} else {
			f += "\t "
		}
		if c.Op != "." {
			f += fmt.Sprintf("%-*s  %2s  %s\n", longest, c.Key+c.Range.String(),
				c.Op, quote(c.Val, false))
		} else {
			f += c.Key + "\n"
		}
	}
	return
}

// "cookieName:{CURRENT}:/:value"  -->  "cookieName"
func denormalizeCC(k string) (d string) {
	k = strings.Replace(k, "{CURRENT}", "", 1)
	ks := strings.Split(k, ":")

	d = ks[0] + ":" + ks[1] + ":" + ks[2]
	if ks[3] != "value" {
		d += ":" + ks[3]
	}
	return
}

// Pretty print a list of Cookie Conditions m.
func formatSetCookies(m *[]Condition) (f string) {
	if len(*m) == 0 {
		return
	}

	f = "SET-COOKIE\n"
	longest := 0
	for _, c := range *m {
		k := denormalizeCC(c.Key)
		if len(k) > longest {
			longest = len(k)
		}
	}
	for _, c := range *m {
		if c.Neg {
			f += "\t!"
		} else {
			f += "\t "
		}
		if c.Op != "." {
			f += fmt.Sprintf("%-*s  %2s  %s\n", longest, denormalizeCC(c.Key),
				c.Op, quote(c.Val, false))
		} else {
			f += denormalizeCC(c.Key) + "\n"
		}
	}
	return
}

// Pretty print a list of commands/args
func formatCommand(title string, cmd [][]string) (f string) {
	if len(cmd) == 0 {
		return
	}
	f = title + "\n"
	for _, c := range cmd {
		f += "\t"
		for _, a := range c {
			f += quote(a, true) + " "
		}
		f += "\n"
	}
	return
}

// Pretty print a list of log conditions. 
func formatLogCond(lc []LogCondition) (f string) {
	if len(lc) == 0 {
		return
	}
	f = "LOG\n"
	for _, c := range lc {
		f += "\t" + c.String() + "\n"
	}

	return
}

// Pretty print the cookies in our jar.
func formatSendCookies(jar *CookieJar) (s string) {
	if len(jar.All()) == 0 {
		return
	}
	s = "SEND-COOKIE\n"
	for _, cookie := range jar.All() {
		s += fmt.Sprintf("\t%s:%s:%s", cookie.Name, cookie.Domain, cookie.Path)
		if cookie.Secure {
			s += ":Secure"
		}
		s += "  :=  "
		s += quote(cookie.Value, false) + "\n"
	}
	return
}

// String representation as as used by the parser.
func (t *Test) String() (s string) {
	s = "-------------------------------\n" + t.Title + "\n-------------------------------\n"
	s += t.Method + " " + t.Url + "\n"
	s += formatMap("CONST", &t.Const)
	s += formatMultiMap("SEQ", &t.Seq)
	s += formatMultiMap("RAND", &t.Rand)
	s += formatCommand("BEFORE", t.Before)
	s += formatMultiMap("PARAM", &t.Param)
	s += formatMap("HEADER", &t.Header)
	s += formatSendCookies(t.Jar)
	s += formatCond("RESPONSE", &t.RespCond)
	s += formatSetCookies(&t.CookieCond)
	s += formatCond("BODY", &t.BodyCond)
	if len(t.Tag) > 0 {
		s += "TAG\n"
		for i, tagCond := range t.Tag {
			fts := tagCond.String()
			if i > 0 && strings.Contains(fts, "\n") {
				s += "\t\n"
			}
			s += "\t" + fts + "\n"
		}
	}
	specSet := make(map[string]int) // map with non-standard settings
	for k, v := range t.Setting {
		if dflt, ok := DefaultSettings[k]; ok && v != dflt {
			specSet[k] = v
		}
	}
	s += formatCommand("AFTER", t.After)
	s += formatLogCond(t.Log)
	s += formatSettings(&specSet)

	return
}
