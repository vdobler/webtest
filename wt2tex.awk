BEGIN {
    print "\\documentclass[11pt]{article}"
    print "\\setlength{\\oddsidemargin}{0mm} \\setlength{\\evensidemargin}{0mm}"
    print "\\setlength{\\textwidth}{160mm} \\setlength{\\topmargin}{-15mm}"
    print "\\setlength{\\parindent}{0in} \\setlength{\\textheight}{240mm}"
    print "\\begin{document}"
    print "\\title{Webtest Reference}\\author{Dr. Volker Dobler}\\date{September 2011}"
    print "\\maketitle"
    print "\\tableofcontents"
    print "\\newpage"
}

function escapelt(t) {
    gsub(/\\/, "§", t)
    gsub(/\$/, "\\$", t)
    gsub(/%/, "\\%", t)
    gsub(/#/, "\\#", t) 
    gsub(/&/, "\\&", t)
    gsub(/_/, "\\_", t)
    gsub(/~/, "\\~", t)
    gsub(/{/, "\\{", t)
    gsub(/}/, "\\}", t)
    gsub(/</, "$<$", t)
    gsub(/>/, "$>$", t)
    gsub(/\^/, "\\^", t)
    gsub(/§/, "$\\backslash$", t)
    return t
}

function enditemize() {
    if (initemize) {
	print "\\end{itemize}"
	initemize = 0
    }
}

/^## *[-=]*[ \t]*$/ { # Unerlining of section title
    next 
}

/^##/ { # a new section
    t = substr($0, 3)
    # t = sub(/^ .*/, "", t)
    printf "\\section{%s}\n", escapelt(t)
    section = 1
    enditemize()
    next 
}

/^#[\t ]*$/ { # empty comment lines
    enditemize()
    print ""
    next 
}

/^#/ { # unindented comments
    t = $0 "     "
    if (section) {
	if (substr(t,1,5) == "#  - ") {
	    if (!initemize) {
		print "\\begin{itemize}"
	    }
	    initemize = 1
	    printf "\\item %s\n", escapelt(substr(t,5))
	} else if (substr(t,1,4) == "#   ") {
	    if (initemize) {
		printf "%s\n", escapelt(substr(t,3))
	    } else {
		t = substr(t, 2)
		printf "\\verb§%s§\\\\\n", t
	    }
	} else {
	    enditemize()
	    gsub(/^#[ \t]*/, "", t)
	    print escapelt(t)
	}
    } else {
	enditemize()
	printf "\\textit{\\small %s}\\\\\n", escapelt(t)
    }
    next 
}

/^[\t ]*$/ { # empty lines
    enditemize()
    print ""
    next 
}

func sprefix(t) {
    i = 1
    n = 0
    while(i<length(t)) {
	a = substr(substr(t, i), 1, 1)
	if (a == " ") { n++ }
	else if (a == "\t") { n += 4 }
	else return n
	i++
    }
    return n
}

/^\t[ \t]*#/ { # a indented comment
    t = substr($0, 2)
    s = "    "
    n = sprefix(t)
    while(n>0) {
	s = s " "
	n--
    }
    sub(/^\t[ \t]*#/, "", t)
    t = escapelt(t)
    enditemize()
    printf "\\verb+%s+\\textit{\\small %s}\\\\\n", s, t
    next
}

/^\t[ \t]*/ { # indented tests
    t = substr($0, 2)
    gsub(/\t/, "    ", t)
    printf "\\verb§    %s§\\\\\n", t
    next 
}

/^[^ \t]/ { # top level stuff of tests
    enditemize()
    if (section) { 
	print ""
	section = 0
    }
    printf "\\verb§%s§\\\\\n", $0
    next
}


END {
    print "\\end{document}"
}