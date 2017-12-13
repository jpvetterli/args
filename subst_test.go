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

func symbolTable(prefix rune) *symtab {
	st := newSymtab(prefix)
	p := string(prefix)
	st.put(p+"foo", "ba⌘r")
	st.put(p+"1magic", "xyzzy")
	st.put(p+"-日-本_語", "nihongo <日本語>")
	st.put(p+"a+b", "invalid symbols are ignored")
	return &st
}

func TestSubstOnData(t *testing.T) {
	for _, c := range testData {
		result, _, _ := substitute(c.input, symbolTable('$'))
		if string(result.s) != c.expect {
			t.Errorf("Substitute(%q) == %q, expect: %q", c.input, result.s, c.expect)
		}
	}
}

func TestSubstOnData1(t *testing.T) {
	for _, c := range testData1 {
		result, _, _ := substitute(c.input, symbolTable('⌘'))
		if string(result.s) != c.expect {
			t.Errorf("Substitute(%q) == %q, expect: %q", c.input, result.s, c.expect)
		}
	}
}

func TestSubstWithAsciiMarker(t *testing.T) {
	input := "@@foo"
	expect := "ba⌘r"
	result, _, _ := substitute(input, symbolTable('@'))
	if string(result.s) != expect {
		t.Errorf("Substitute(%q) == %q, expect: %q", input, result.s, expect)
	}
}

func TestSubstWithMultiByteDataAndSymbols(t *testing.T) {
	input := "⌘⌘foo 日本語 ⌘⌘日本語"
	expect := "ba⌘r 日本語 nihongo <日本語>"
	symt := func(prefix rune) *symtab {
		st := newSymtab(prefix)
		p := string(prefix)
		st.put(p+"foo", "ba⌘r")
		st.put(p+"日本語", "nihongo <日本語>")
		return &st
	}
	result, _, _ := substitute(input, symt('⌘'))
	if string(result.s) != expect {
		t.Errorf("Substitute(%q) == %q, expect: %q", input, result.s, expect)
	}
}

func TestError(t *testing.T) {
	input := "$$foo $$bar\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98 etc."
	_, _, err := substitute(input, symbolTable('$'))
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

func TestByteOrderMarkError(t *testing.T) {
	// BOM invalid unless first character
	input := "\uFEFF$$foo bar\uFEFF\uFEFF\uFEFF etc."
	_, _, err := substitute(input, symbolTable('$'))
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
