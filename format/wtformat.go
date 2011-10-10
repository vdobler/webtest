//
// wtformat - prettyprinter for webtest suites
//
// Copyright 2011 Volker Dobler. All rights reserved.
//
package main

import (
	"fmt"
	"flag"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"
	"path"

	"github.com/vdobler/webtest/suite"
)

// Return true iff filename points to a parsabele webtest suite.
func checkSuite(filename string) bool {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Cannot read from '%s': %s\n", filename, err.String())
		return false
	}
	defer file.Close()
	basename := path.Base(filename)
	parser := suite.NewParser(file, basename)
	_, serr := parser.ReadSuite()
	if serr != nil {
		fmt.Printf("Problems parsing '%s': %s\n", filename, err.String())
		return false
	}
	return true
}

// The header of the generated LaTeX File
func makeHeader(filename string) string {
	texf := `\documentclass[11pt,utf8]{article}
\usepackage{url}
\usepackage[utf8]{inputenc}
\setlength{\oddsidemargin}{0mm} \setlength{\evensidemargin}{0mm} \setlength{\textwidth}{160mm}
\setlength{\topmargin}{-15mm} \setlength{\textheight}{240mm}
\setlength{\parindent}{0in}
\itemsep0pt\parsep0pt\parskip0pt\partopsep0pt
\topsep0pt \partopsep0pt
\reversemarginpar
\newenvironment{indcmt}%%
  {\par\vspace{3mm}\makebox[9mm]{}\begingroup\noindent\small\it{}}%%
  {\endgroup\vspace{1mm}}

\begin{document}
\title{%s}\author{%s}\date{%s}
\maketitle
\tableofcontents
\newpage
`
	basename := path.Base(filename)
	date := time.LocalTime().Format("02. Jan 2006")
	return fmt.Sprintf(texf, basename, filename, date)
}

var (
	ree = regexp.MustCompile(`\\_[a-zA-ZäöüÄÖÜéèáàóòç]*\\_`)
	reb = regexp.MustCompile(`\*[a-zA-ZäöüÄÖÜéèáàóòç]*\*`)
	ret = regexp.MustCompile(`\|[^ ]*\|`)
)

func typewriter(s string) string {
	s = s[1 : len(s)-1]
	if hp(s, "http://", "https://", "file://") {
		fmt.Printf("\n\nXXXXXX\n\n")
		s = "\\url{" + s + "}"
	} else {
		s = "\\emph{\\texttt{" + s + "}}"
	}
	return s
}

// Hack (works properly most time): Replace LaTeX special chars with command sequences.
func quoteTex(s string) string {

	for _, x := range [][2]string{
		{"\\", "§!§|§°§+§-§"},
		{"$", "\\$"},
		{"%", "\\%"},
		{"_", "\\_"},
		{"^", "\\^"},
		{"~", "\\~"},
		{"{", "\\{"},
		{"}", "\\}"},
		{"<", "$<$"},
		{">", "$>$"},
		{"§!§|§°§+§-§", "$\\backslash$"},
	} {
		s = strings.Replace(s, x[0], x[1], -1)
	}

	// handle "_emphasis_", "*boldface*" and "|typewriter|"
	s = ree.ReplaceAllStringFunc(s, func(a string) string { return "\\emph{" + a[2:len(a)-2] + "}" })
	s = reb.ReplaceAllStringFunc(s, func(a string) string { return "\\textbf{" + a[1:len(a)-1] + "}" })
	s = ret.ReplaceAllStringFunc(s, typewriter)

	return s
}

func trim(s string) string {
	return strings.Trim(s, " \t")
}

func hp(s string, prefix ...string) bool {
	for _, p := range prefix {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

var maxlength = 76  // How many characters in tt font
var contlength = 60 // How many characters in tt font on continuation line

// format s verbatim
func verbatim(s string) (tex string) {
	// if strings.Index(s, "§") == -1 { return "\\verb§"+s+"§" }
	if strings.Index(s, "|") == -1 {
		return "\\verb|" + s + "|"
	}
	if strings.Index(s, "+") == -1 {
		return "\\verb+" + s + "+"
	}
	if strings.Index(s, "!") == -1 {
		return "\\verb!" + s + "!"
	}

	sp := strings.Split(s, "|")
	fmt.Printf("Alles drin: %v\n", sp)
	for i, p := range sp {
		if i > 0 {
			tex += "\\verb+|+"
		}
		tex += "\\verb|" + p + "|"
		fmt.Printf("%d: %s\n", i, tex)
	}
	return
}

// format s verbatim, fold lines if too long into first line and list of remaining lines.
func foldVerbatim(s string) (first string, rest []string) {
	r := ""
	if len(s) > maxlength+4 {
		s, r = s[:maxlength], s[maxlength:]
	}
	first = verbatim(s)
	for len(r) > 0 {
		if len(r) > contlength+4 {
			s, r = r[:contlength], r[contlength:]
		} else {
			s, r = r, ""
		}
		s = fmt.Sprintf(`\\\makebox[25mm]{}$\hookrightarrow\quad$%s`, verbatim(s)) + "\n"
		rest = append(rest, s)
	}
	return

}

// format a "test" line: all the indented lines
func formatTest(s string, lineno int) (tex string) {
	n := 0
	fmt.Printf("line=%s\n", s)
	for i := 0; s[i] == ' ' || s[i] == '\t'; i++ {
		if s[i] == ' ' {
			n++
		} else {
			n += 4
		}

	}
	indent := strings.Repeat(" ", n)
	s = indent + trim(s)
	first, rest := foldVerbatim(s)
	tex = fmt.Sprintf("\n%s\\marginpar{\\hfil \\scriptsize %d}\n", first, lineno)
	for _, r := range rest {
		tex += r
	}
	return
}

func endMode(mode string) string {
	switch mode {
	case "par":
		return "\n"
	case "item":
		return "\\end{itemize}\n"
	case "enum":
		return "\\end{enumerate}\n"
	case "desc":
		return "\\end{description}\n"
	}
	panic("No such mode to end " + mode)
}

func startMode(mode, current string) string {
	if mode == current {
		return ""
	}
	endMode(current)
	switch mode {
	case "item":
		return "\\begin{itemize}\\itemsep0pt\\parsep0pt\\parskip0pt\\partopsep0pt\n"
	case "enum":
		return "\\begin{enumerate}\\itemsep0pt\\parsep0pt\n"
	case "desc":
		return "\\begin{description}\\itemsep0pt\\parsep0pt\n"
	}
	panic("No such mode to start " + mode)
}

/********************
func findDesc(s string) string {
	s = s[3:]
	i := strings.Index(s, " ")
	if i == -1 { return "" }
	s = s[:i]
	if len(s)>=3 && strings.HasSuffix(s,"::") {
		return s[:len(s)-3]
	}
	return ""
}
**********************/

var (
	begindcmt = `\verb|    |\parbox{12cm}{\vspace{2mm}`
	endindcmt = "\\vspace{1mm}}\n"
)

func formatComment(comments []string) (tex string) {
	mode := "par"
	ind := false
	comment := ""
	real := false

	tex = "{\\it"
	for len(comments) > 0 {
		comment, comments = comments[0], comments[1:]

		if comment[0] == '#' {
			comment = comment[2:]
			if ind {
				tex += endMode(mode)
				mode = "par"
				ind = false
				tex += endindcmt
			}
		} else { // indented
			comment = trim(comment)[2:]
			if !ind {
				tex += endMode(mode)
				mode = "par"
				ind = true
				tex += begindcmt
			}

		}

		// Empty line:
		if len(trim(comment)) == 0 {
			tex += endMode(mode)
			mode = "par"
			continue
		}

		// Itemize
		if strings.HasPrefix(comment, " - ") || strings.HasPrefix(comment, " o ") ||
			strings.HasPrefix(comment, " * ") {
			/* if desc := findDesc(comment); desc != "" {
				tex += startMode("desc", mode)
				mode = "desc"
				tex += "\\item[" +quoteTex(desc) + "] " + quoteTex(comment[3:]) + "\n"
			} else { */
			tex += startMode("item", mode)
			mode = "item"
			tex += "\\item " + quoteTex(comment[3:]) + "\n"
			/* } */
			real = true
			continue
		}

		// Enumerate
		if strings.HasPrefix(comment, " # ") {
			tex += startMode("enum", mode)
			mode = "enum"
			tex += "\\item " + quoteTex(comment[3:]) + "\n"
			real = true
			continue
		}

		// Item content
		if strings.HasPrefix(comment, "   ") {
			tex += quoteTex(comment[3:]) + "\n"
			continue
		}

		// Verbatim stuff
		if strings.HasPrefix(comment, "  ") {
			tex += "\\\\\\verb|  " + trim(comment) + "|\\\\\n"
			real = true
			continue
		}

		// Any other text
		tc := trim(comment)
		if len(tc) > 0 {
			real = true
		}
		tex += quoteTex(tc) + "\n"

	}
	tex += endMode(mode)
	if ind {
		tex += endindcmt
	}

	tex += "}\n"
	if real {
		// tex = "\n\\smallskip\n" + tex
	}
	return
}

// Works for valid suites only
func formatSuite(lines []string) (tex string) {
	var line string
	comments := make([]string, 0, 10)
	lineno := 0

	for len(lines) > 0 {
		line, lines = lines[0], lines[1:]
		lineno++

		// Next test
		if strings.HasPrefix(line, "------") {
			tex += formatComment(comments)
			comments = comments[0:1]
			line, lines = lines[0], lines[2:]
			lineno += 2
			tl := trim(line)
			if tl == "Global" {
				tex += "\\section{" + quoteTex(tl) + "}\n"
			} else {
				tex += "\\subsection{" + quoteTex(tl) + "}\n"
			}
			tex += "\n\\verb|------------------------------------------------|\\\\\n"
			tex += "\\verb|" + trim(line) + "|\\\\\n"
			tex += "\\verb|------------------------------------------------|\n\n"
			continue
		}

		// Empty lines --> empty comment
		if len(trim(line)) == 0 {
			comments = append(comments, "# ")
			continue
		}

		// Section titles
		if strings.HasPrefix(line, "###############") {
			// TODO safeguard....
			line, lines = trim(lines[0][1:]), lines[1:]
			lineno++
			tex += "\\section{" + quoteTex(line) + "}\n"
			continue
		}

		// Comments
		if strings.HasPrefix(trim(line), "#") {
			if len(trim(line)) == 1 {
				line = "# "
			}
			comments = append(comments, line)
			continue
		}

		tex += formatComment(comments)
		comments = comments[0:1]

		if strings.HasPrefix(line, "\t") {
			// Indented test stuff
			tex += formatTest(line, lineno)
			continue
		} else {
			// A test section
			first, rest := foldVerbatim(trim(line))
			tex += "\n\\vspace{2mm}\n" + first + "\n"
			for _, r := range rest {
				tex += r
			}
			continue
		}

		panic("This should not happen!")
	}

	return tex
}

func formatFile(filename string) {
	if !checkSuite(filename) {
		return
	}
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Cannot read from '%s': %s\n", filename, err.String())
		return
	}
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("Cannot read from %s: %s", filename, err.String())
		return
	}
	file.Close()
	sbuf := string(buf)
	lines := strings.Split(sbuf, "\n")
	tex := makeHeader(filename)
	tex += formatSuite(lines)
	tex += "\n\\end{document}\n"

	filename += ".tex"
	ofile, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Cannot create file '%s': %s\n", filename, err.String())
		return
	}
	defer ofile.Close()
	ofile.Write([]byte(tex))
}

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Println("Usage: wtformat <file>...")
		os.Exit(1)
	}

	for _, filename := range flag.Args() {
		formatFile(filename)
	}
}
