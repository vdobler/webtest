package tag

/*
	This package helps testing for occurenc of tags in html/xml documents.

	Tags are described by a plaintext strings. The following examples show
	most of the possibilities

	  tagspec:
		tagname [ { attr } ] [ contentOp content ]

	  attr:
	    [ class | attribute ]

	  class:
	  	[ '!' ] 'class' [ '=' fixed ]

	  attribute:
	  	[ '!' ] name [ '=' content ]

	  contentOp:
		[ '==' | '=D=' ]               '==' is normal matching of text content
		                               'wheras '=D=' is deep matching of nested
									   text content.

	  content:
		[ pattern | '/' regexp '/' ]   pattern may contain '*' and '?' and works
		                               like shell globing. regexp is what it is.

	Only specified classes, attributes and content is considered when finding
	tags in a html/xml document. E.g.:
	  "p lang=en"
	will match any p-tag with lang="en" regardless of any other classes, 
	attributes and content of the p-tag.

	Values for attributes may be ommitted: Such test just check wether the
	tag has the attribute (value does not matter).

	The difference between class and "normal" attribute testing is: Attributes
	may be specified only once and their value is optional wheras classes can
	be specified multiple times and must contain a value. Think of a tag like
	  <p class="important news wide">Some Text</p>
	As beeing something like
	  <p class="important" class="news" class="wide">Some Text</p>
	For finding tags.





*/

import (
	"os"
	//	"strings"
	"utf8"
)

var ErrBadPattern = os.NewError("Syntax error in pattern")

// Match returns true if name matches the shell file name pattern.
// The pattern syntax is:
//
//	pattern:
//		{ term }
//	term:
//		'*'         matches any sequence of non-/ characters
//		'?'         matches any single non-/ character
//		c           matches character c (c != '*', '?', '', '')
//		'\\' c      matches character c
//
// Match requires pattern to match all of name, not just a substring.
// The only possible error return is when pattern is malformed.
//
func Match(pattern, name string) (matched bool, err os.Error) {
Pattern:
	for len(pattern) > 0 {
		var star bool
		var chunk string
		star, chunk, pattern = scanChunk(pattern)
		if star && chunk == "" {
			// Trailing * matches rest of string.
			return true, nil
		}
		// Look for match at current position.
		t, ok, err := matchChunk(chunk, name)
		// if we're the last chunk, make sure we've exhausted the name
		// otherwise we'll give a false result even if we could still match
		// using the star
		if ok && (len(t) == 0 || len(pattern) > 0) {
			name = t
			continue
		}
		if err != nil {
			return false, err
		}
		if star {
			// Look for match skipping i+1 bytes.
			for i := 0; i < len(name); i++ {
				t, ok, err := matchChunk(chunk, name[i+1:])
				if ok {
					// if we're the last chunk, make sure we exhausted the name
					if len(pattern) == 0 && len(t) > 0 {
						continue
					}
					name = t
					continue Pattern
				}
				if err != nil {
					return false, err
				}
			}
		}
		return false, nil
	}
	return len(name) == 0, nil
}

// scanChunk gets the next segment of pattern, which is a non-star string
// possibly preceded by a star.
func scanChunk(pattern string) (star bool, chunk, rest string) {
	for len(pattern) > 0 && pattern[0] == '*' {
		pattern = pattern[1:]
		star = true
	}
	inrange := false
	var i int
Scan:
	for i = 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			// error check handled in matchChunk: bad pattern.
			if i+1 < len(pattern) {
				i++
			}
		case '*':
			if !inrange {
				break Scan
			}
		}
	}
	return star, pattern[0:i], pattern[i:]
}

// matchChunk checks whether chunk matches the beginning of s.
// If so, it returns the remainder of s (after the match).
// Chunk is all single-character operators: literals, char classes, and ?.
func matchChunk(chunk, s string) (rest string, ok bool, err os.Error) {
	for len(chunk) > 0 {
		if len(s) == 0 {
			return
		}
		switch chunk[0] {
		case '?':
			_, n := utf8.DecodeRuneInString(s)
			s = s[n:]
			chunk = chunk[1:]

		case '\\':
			chunk = chunk[1:]
			if len(chunk) == 0 {
				err = ErrBadPattern
				return
			}
			fallthrough

		default:
			if chunk[0] != s[0] {
				return
			}
			s = s[1:]
			chunk = chunk[1:]
		}
	}
	return s, true, nil
}
