package args

import (
	"testing"
)

var testData = []struct {
	input  string
	expect string
}{
	{"", ""},
	{"foo", "foo"},
	{"$$foo", "ba⌘r"},
	{"$$foo ", "ba⌘r "},
	{"x$$foo", "xba⌘r"},
	{"$$Foo", "$$Foo"},
	{"x$$Foo", "x$$Foo"},
	{"$$$foo", "$ba⌘r"},
	{"$$$$foo", "$$ba⌘r"},
	{"$$$$foo ", "$$ba⌘r "},
	{"$$$Foo", "$$$Foo"},
	{"$$$$Foo", "$$$$Foo"},
	{"$$$foo ", "$ba⌘r "},
	{"$$$Foo ", "$$$Foo "},
	{"$$$foo$bar", "ba⌘rbar"},
	{"x$$$foo$bar", "xba⌘rbar"},
	{"$$$Foo$bar", "$$$Foo$bar"},
	{"x$$$Foo$bar", "x$$$Foo$bar"},
	{"x$$$foo$[$$1magic]", "xba⌘r[xyzzy]"},
	{"$", "$"},
	{" $ ", " $ "},
	{"$$", "$$"},
	{" $$ ", " $$ "},
	{"$$$$", "$$$$"},
	{" $$$$ ", " $$$$ "},
	{"x$$$$x", "x$$$$x"},
	{"x$$$$x", "x$$$$x"},
	{"ab$$$x$cd xyzzy $$foo[bar] ", "ab$$$x$cd xyzzy ba⌘r[bar] "},
	{"ab$$$x$cd xyzzy $$Foo[bar] ", "ab$$$x$cd xyzzy $$Foo[bar] "},
	{"symbol reference $$a+b invalid, even if in symbol map", "symbol reference $$a+b invalid, even if in symbol map"},
	// (now invalid) {"\uFEFF$$foo $$bar", "ba⌘r $$bar"}, // initial BOM skipped
}

// test data for ⌘ marker
var testData1 = []struct {
	input  string
	expect string
}{
	{"⌘⌘foo", "ba⌘r"},
	{"⌘⌘foo 日本語", "ba⌘r 日本語"},
	{"⌘⌘foo 日本語 ⌘⌘-日-本_語", "ba⌘r 日本語 nihongo <日本語>"},
}

var testCount = []struct {
	input  string
	rcount int
	ucount int
}{
	{"", 0, 0},
	{"foo", 0, 0},
	{"$$foo", 1, 0},
	{"$$bar", 0, 1},
	{"$$foo yes $$foo $$bar $$foo $$foo $$foo $$bar $$bar $$bar ", 5, 4},
	{"$$$foo$$$$foo$no$$$bar$$$$foo$$$$foo$", 4, 1},
	{"$$$foo$ whatever $$1magic $$$-日-本_語$@!)", 3, 0},
}

// some symbols
var symbols = &map[string]string{
	"foo":    "ba⌘r",
	"1magic": "xyzzy",
	"-日-本_語": "nihongo <日本語>",
	"a+b":    "invalid symbols are ignored",
}

func TestSubstOnData(t *testing.T) {
	for _, c := range testData {
		result, _, _, _ := newSubstituter('$').Substitute([]byte(c.input), symbols)
		if string(result) != c.expect {
			t.Errorf("Substitute(%q) == %q, expect: %q", c.input, result, c.expect)
		}
	}
}

func TestSubstOnData1(t *testing.T) {
	for _, c := range testData1 {
		result, _, _, _ := newSubstituter('⌘').Substitute([]byte(c.input), symbols)
		if string(result) != c.expect {
			t.Errorf("Substitute(%q) == %q, expect: %q", c.input, result, c.expect)
		}
	}
}

func TestSubstWithAsciiMarker(t *testing.T) {
	input := "@@foo"
	expect := "ba⌘r"
	result, _, _, _ := newSubstituter('@').Substitute([]byte(input), symbols)
	if string(result) != expect {
		t.Errorf("Substitute(%q) == %q, expect: %q", input, result, expect)
	}
}

func TestSubstWithMultiByteDataAndSymbols(t *testing.T) {
	input := "⌘⌘foo 日本語 ⌘⌘日本語"
	expect := "ba⌘r 日本語 nihongo <日本語>"
	result, _, _, _ := newSubstituter('⌘').Substitute([]byte(input), &map[string]string{
		"foo": "ba⌘r",
		"日本語": "nihongo <日本語>",
	})
	if string(result) != expect {
		t.Errorf("Substitute(%q) == %q, expect: %q", input, result, expect)
	}
}

func TestError(t *testing.T) {
	input := "$$foo $$bar\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98 etc."
	expect := "ba⌘r $$bar"
	result, _, _, err := newSubstituter('$').Substitute([]byte(input), symbols)
	if string(result) != expect {
		t.Errorf("Substitute(%q) == %q, expect: %q", input, result, expect)
	}
	if err == nil {
		t.Errorf("Error expected")
		return
	}
	errorString := err.Error()
	expectedErrorString := "Invalid character at offset 11"
	if errorString != expectedErrorString {
		t.Errorf("Error string: %q, expected: %q", errorString, expectedErrorString)
	}
}

func TestErrorLoose(t *testing.T) {
	input := "$$foo $$bar\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98 etc."
	expect := "ba⌘r $$bar\xbd\xb2=\xbc ⌘ etc."
	result, _, _, err := newLooseSubstituter('$').Substitute([]byte(input), symbols)
	if string(result) != expect {
		t.Errorf("Substitute(%q) == %q, expect: %q", input, result, expect)
	}
	if err != nil {
		t.Errorf("No error expected")
		return
	}
}

func TestByteOrderMarkError(t *testing.T) {
	// BOM invalid unless first character
	input := "\uFEFF$$foo bar\uFEFF\uFEFF\uFEFF etc."
	expect := "" // if BOM as 1st char allowed: "ba⌘r bar"
	result, _, _, err := newSubstituter('$').Substitute([]byte(input), symbols)
	if string(result) != expect {
		t.Errorf("Substitute(%q) == %q, expect: %q", input, result, expect)
	}
	if err == nil {
		t.Errorf("Error expected")
		return
	}
	errorString := err.Error()
	expectedErrorString := "Invalid character at offset 0"
	// if BOM as 1st char allowed:
	// expectedErrorString := "Invalid character at offset 12"
	if errorString != expectedErrorString {
		t.Errorf("Error string: %q, expected: %q", errorString, expectedErrorString)
	}
}

func TestByteOrderMarkErrorLoose(t *testing.T) {
	input := "\uFEFF$$foo bar\uFEFF\uFEFF\uFEFF etc."
	expect := "\ufeffba⌘r bar\ufeff\ufeff\ufeff etc."
	result, _, _, err := newLooseSubstituter('$').Substitute([]byte(input), symbols)
	if string(result) != expect {
		t.Errorf("Substitute(%q) == %q, expect: %q", input, result, expect)
	}
	if err != nil {
		t.Errorf("No error expected")
		return
	}
}

func TestSubstOnCountData(t *testing.T) {
	for _, c := range testCount {
		_, rcount, ucount, _ := newSubstituter('$').Substitute([]byte(c.input), symbols)
		if rcount != c.rcount || ucount != c.ucount {
			t.Errorf("Substitute(%q), r/u == %d/%d, expected: %d/%d", c.input, rcount, ucount, c.rcount, c.ucount)
		}
	}
}
