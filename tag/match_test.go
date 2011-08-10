// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tag

import (
	"os"
	"testing"
)

type MatchTest struct {
	pattern, s string
	match      bool
	err        os.Error
}

var matchTests = []MatchTest{
	{"abc", "abc", true, nil},
	{"*", "abc", true, nil},
	{"*c", "abc", true, nil},
	{"a*", "a", true, nil},
	{"a*", "abc", true, nil},
	{"a*", "ab/c", true, nil},
	{"a*/b", "abc/b", true, nil},
	{"a*/b", "a/c/b", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxe/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, nil},
	{"a*b?c*x", "abxbbxdbxebxczzx", true, nil},
	{"a*b?c*x", "abxbbxdbxebxczzy", false, nil},
	{"ab[c]", "abc", false, nil},
	{"ab[c]", "ab[c]", true, nil},
	{"ab[b-d]", "abc", false, nil},
	{"ab[b-d]", "ab[b-d]", true, nil},
	{"ab[^c]", "abc", false, nil},
	{"ab[^c]", "ab[^c]", true, nil},
	{"ab[^b-d]", "abc", false, nil},
	{"ab[^b-d]", "ab[^b-d]", true, nil},
	{"a\\*b", "a*b", true, nil},
	{"a\\*b", "ab", false, nil},
	{"a?b", "a☺b", true, nil},
	{"a???b", "a☺b", false, nil},
	{"a?b", "a/b", true, nil},
	{"a*b", "a/b", true, nil},
	{"\\", "a", false, ErrBadPattern},
	{"*x", "xxx", true, nil},
}

func TestMatch(t *testing.T) {
	for _, tt := range matchTests {
		ok, err := Match(tt.pattern, tt.s)
		if ok != tt.match || err != tt.err {
			t.Errorf("Match(%#q, %#q) = %v, %v want %v, nil", tt.pattern, tt.s, ok, err, tt.match)
		}
	}
}
