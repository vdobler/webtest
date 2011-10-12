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
		fmt.Printf("Problems parsing '%s': %s\n", filename, serr.String())
		return false
	}
	return true
}

// The header of the generated LaTeX File
func makeHeader(filename string) string {
	texf := `\documentclass[11pt]{article}
\usepackage{url}
\usepackage{enumitem}
\setlength{\oddsidemargin}{0mm} \setlength{\evensidemargin}{0mm} \setlength{\textwidth}{160mm}
\setlength{\topmargin}{-15mm} \setlength{\textheight}{240mm}
\setlength{\parindent}{0in}
\itemsep0pt\parsep0pt\parskip0pt\partopsep0pt
\topsep0pt \partopsep0pt
\reversemarginpar

\begin{document}
\textsf{\textbf{\Huge %s}}\\[1ex]
\makebox[12mm][l]{File:} \texttt{%s}\\
\makebox[12mm][l]{Date:} %s

\tableofcontents
\newpage
`
	basename := quoteSpecial(path.Base(filename))
	filename = quoteSpecial(filename)
	date := time.LocalTime().Format("02. Jan 2006, 15:04")
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
		// fmt.Printf("\n\nXXXXXX\n\n")
		s = "\\url{" + s + "}"
	} else {
		s = "\\emph{\\texttt{" + s + "}}"
	}
	return s
}

// quote special TeX characters
func quoteSpecial(s string) string {
	for _, x := range [][2]string{
		{"\\", "§!§!§°§+§-§"},
		{"$", "\\$"},
		{"%", "\\%"},
		{"_", "\\_"},
		{"#", "\\#"},
		{"{", "\\{"},
		{"}", "\\}"},
		{"~", "\\~{}"},
		{"^", "\\^{}"},
		{"<", "$<$"},
		{">", "$>$"},
		{"|", "$|$"},
		{"§!§!§°§+§-§", "\\textbackslash{}"},
	} {
		s = strings.Replace(s, x[0], x[1], -1)
	}
	return s
}

// Hack (works properly most time): Replace LaTeX special chars with command sequences.
func quoteTex(s string) string {
	s = quoteSpecial(s)

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
	for i, p := range sp {
		if i > 0 {
			tex += "\\verb+|+"
		}
		tex += "\\verb|" + p + "|"
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
	tex += "\n"
	return
}

func endMode(lastmode string) string {
	switch lastmode {
	case "par":
		return "\n"
	case "empty":
		return ""
	case "verb":
		return "\n"
	case "item":
		return "\\end{itemize}\n"
	case "enum":
		return "\\end{enumerate}\n"
	}
	panic("No such mode to end " + lastmode)
}

func startMode(mode, current string) string {
	if mode == current {
		return ""
	}
	switch mode {
	case "item":
		return "\\begin{itemize}[noitemsep,nolistsep]\n"
	case "enum":
		return "\\begin{enumerate}[noitemsep,nolistsep]\n"
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
	// begindcmt = `\verb|    |\parbox{152mm}{\rule{0pt}{3ex}\it{}`
	// endindcmt = "\\rule[-1ex]{0pt}{1ex}}\n"
	begindcmt = `\verb|    |\parbox{152mm}{\vspace{10pt}\it{}`
	endindcmt = "\\vspace{5pt}}\n"
)

func endIndentedComment(tex string) string {
	if strings.HasSuffix(tex, "\n\n") {
		tex = tex[:len(tex)-2]
	}
	tex += endindcmt
	return tex
}

func allEmpty(comments []string) bool {
	for _, c := range comments {
		c = trim(c)
		if c != "#" {
			return false
		}
	}
	return true
}

func formatComment(comments []string) (tex string) {
	// fmt.Printf("\nFormating %d comment lines.\n", len(comments))
	if allEmpty(comments) {
		// fmt.Printf("All empty.\n") #   Verbatim
		return "\n\n"
	}

	lastMode, lastIndent := "par", false
	comment := ""

	for len(comments) > 0 {
		comment, comments = comments[0], comments[1:]
		// fmt.Printf("Comment: >>%s<<\n", comment)
		curIndent := false
		if comment[0] == '#' {
			curIndent = false
			comment = comment[2:] // "# content" --> "content"
		} else {
			curIndent = true
			comment = strings.TrimLeft(comment, " \t")[2:] // "\t # content" --> "content"
		}

		var curMode string
		switch true {
		case len(trim(comment)) == 0:
			curMode = "empty"
		case hp(comment, " - ", " o ", " * "):
			curMode = "item"
		case hp(comment, " # "):
			curMode = "enum"
		case hp(comment, "    "):
			curMode = "verb"
		case hp(comment, "   "):
			curMode = "cont"
		case hp(comment, "  "):
			curMode = "verb"
		default:
			curMode = "par"
		}
		// fmt.Printf("Current: %s %t;  Last: %s %t\n", curMode, curIndent, lastMode, lastIndent)

		// Verb is not allowed as parameter: Handle special
		if curMode == "verb" {
			if lastMode != "verb" {
				tex += endMode(lastMode)
				if lastIndent {
					tex += endIndent()
				}
			}
			comment = comment[2:]
			if curIndent {
				tex += "\n" + verbatim("      "+comment) + "\n\n"
			} else {
				tex += "\n" + verbatim("  "+comment) + "\n\n"
			}
			lastMode, lastIndent = "verb", false
			continue
		}

		// Handle change in indent and/or mode
		if lastIndent != curIndent {
			fmt.Printf("Indent changed.\n")
			tex += endMode(lastMode)
			if curIndent {
				tex += startIndent()
			} else {
				tex += endIndent()
			}
		}

		if curMode == "cont" {
			tex += quoteTex(comment[3:]) + "\n"
			lastIndent = curIndent
			continue
		}

		if curMode == "empty" {
			tex += endMode(lastMode)
			lastMode, lastIndent = curMode, curIndent
			continue
		}

		if curMode != lastMode {
			tex += endMode(lastMode)
		}

		if curMode == "item" {
			tex += startMode("item", lastMode)
			tex += "\\item " + quoteTex(comment[3:]) + "\n"
		} else if curMode == "enum" {
			tex += startMode("enum", lastMode)
			tex += "\\item " + quoteTex(comment[3:]) + "\n"
		} else if curMode == "par" {
			tex += quoteTex(trim(comment)) + "\n"
		} else {
			panic("No such mode " + curMode)
		}

		lastMode, lastIndent = curMode, curIndent
	}

	tex += endMode(lastMode)

	if lastIndent {
		tex += endIndent()
	}
	return
}

func endIndent() string   { return endindcmt }
func startIndent() string { return begindcmt }

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
			tex += fmt.Sprintf(`\marginpar{\hfil \scriptsize %d}`, lineno)
			tex += verbatim(trim(line)) + "\\\\\n"
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
			tex += formatComment(comments)
			comments = comments[0:1]
			// TODO safeguard....

			line, lines = trim(lines[0][1:]), lines[1:]
			lineno++
			tex += "\\section{" + quoteTex(line) + "}\n"
			continue
		}

		// Comments
		if strings.HasPrefix(trim(line), "#") {
			// all lines are either of the form
			//   "# <optional content>"
			// or
			//   "\t# <optional content>"
			//
			if line[0] == '#' {
				line = line + " "
			} else {
				line = "\t" + trim(line) + " "
			}
			comments = append(comments, line)
			continue
		}

		tex += formatComment(comments)
		comments = comments[0:1]

		// Sections and tests
		if strings.HasPrefix(line, "\t") {
			// Indented test stuff
			tex += formatTest("    "+line[1:], lineno)
			continue
		} else {
			// A test section
			first, rest := foldVerbatim(trim(line))
			tex += "\n\\vspace{2mm}\n" + first + "\n"
			for _, r := range rest {
				tex += r
			}
			tex += "\n"
			continue
		}

		panic("This should not happen!")
	}

	tex += formatComment(comments)
	return tex
}

func formatFile(filename string) bool {
	if !checkSuite(filename) {
		return false
	}
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Cannot read from '%s': %s\n", filename, err.String())
		return false
	}
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("Cannot read from %s: %s", filename, err.String())
		return false
	}
	file.Close()
	sbuf := string(buf)
	lines := strings.Split(sbuf, "\n")
	fullpath, err := os.Getwd()
	if err != nil {
		fullpath = "./"
	}
	fullpath = path.Clean(path.Join(fullpath, filename))
	tex := makeHeader(fullpath)
	tex += formatSuite(lines)
	tex += "\n\\end{document}\n"

	filename += ".tex"
	ofile, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Cannot create file '%s': %s\n", filename, err.String())
		return false
	}
	defer ofile.Close()
	ofile.Write([]byte(tex))
	return true
}

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Println("Usage: wtformat <file>...")
		os.Exit(1)
	}

	okay := true
	for _, filename := range flag.Args() {
		okay = okay && formatFile(filename)
	}
	if !okay {
		os.Exit(1)
	}
	os.Exit(0)
}
