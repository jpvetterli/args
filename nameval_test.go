package args

import (
	"bytes"
	"testing"
)

var nvGoodTestData = []struct {
	input  string
	expect string
}{
	{"a b c", "<a> <b> <c> "},
	{`a\ b\ c`, "<a b c> "},
	{"a=b", "<a>=<b> "},
	{"a=b c=d e f", "<a>=<b> <c>=<d> <e> <f> "},
	{" a = b ", "<a>=<b> "},
	{"a = [b c]", "<a>=<b c> "},
	{"日本語 = [b c⌘]", "<日本語>=<b c⌘> "},
	{`a \= [b c]`, "<a> <=> <b c> "},
	{`[a b \c]`, `<a b \c> `},
	{"[a [b] c]", "<a [b] c> "},
	{`[a b\] c]`, "<a b] c> "},
	{`[a \[b c]`, "<a [b c> "},
	{`[a b] = x`, "<a b>=<x> "}, // funny names
	{`[a \]b] = x`, "<a ]b>=<x> "},
	{`$[ a \]b] = x`, "<$ a ]b>=<x> "},
}

var nvCustomTestData = []struct {
	input  string
	expect string
}{
	{"a b c", "<a> <b> <c> "},
	{`a! b! c`, "<a b c> "},
	{"a:b", "<a>=<b> "},
	{"a : {b c}", "<a>=<b c> "},
}

var nvBadTestData = []struct {
	input  string
	expect string
}{
	{"a==b c", `at "a==": "=" unexpected`},
	{"a=b c=", `at "a=b c=": premature end of input`},
}

var nvGoodValuesTestData = []struct {
	input  string
	expect string
}{
	{"a b c", "<a> <b> <c> "},
	{`a\=b\ c`, "<a=b c> "},
	{`a \= b\ c`, "<a> <=> <b c> "},
}

var nvBadValuesTestData = []struct {
	input  string
	expect string
}{
	{"a=b c", `at "a=": the input must contain only values`},
	{`a b = \=\ c`, `at "a b =": the input must contain only values`},
}

func TestScannerWithGoodData(t *testing.T) {
	for _, data := range nvGoodTestData {
		result, err := pairs(NewConfig(), []byte(data.input))
		if err != nil {
			t.Errorf(`with "%s" unexpected error: "%s"`, data.input, err.Error())
			return
		}
		if result == nil {
			t.Errorf(`with "%s" no result, expected: "%s"`, data.input, data.expect)
		} else {
			c := compact(result)
			if c != data.expect {
				t.Errorf(`with "%s" result: "%s" expected: "%s"`, data.input, c, data.expect)
			}
		}
	}
}

func TestScannerWithGoodValuesData(t *testing.T) {
	for _, data := range nvGoodValuesTestData {
		result, err := values(NewConfig(), []byte(data.input))
		if err != nil {
			t.Errorf(`with "%s" unexpected error: "%s"`, data.input, err.Error())
			return
		}
		if result == nil {
			t.Errorf(`with "%s" no result, expected: "%s"`, data.input, data.expect)
		} else {
			c := compactValues(result)
			if c != data.expect {
				t.Errorf(`with "%s" result: "%s" expected: "%s"`, data.input, c, data.expect)
			}
		}
	}
}

func TestScannerWithCustomData(t *testing.T) {
	config := NewConfig()
	config.SetSpecial(SpecSymbolPrefix, '?')
	config.SetSpecial(SpecOpenQuote, '{')
	config.SetSpecial(SpecCloseQuote, '}')
	config.SetSpecial(SpecSeparator, ':')
	config.SetSpecial(SpecEscape, '!')
	for _, data := range nvCustomTestData {
		result, err := pairs(config, []byte(data.input))
		if err != nil {
			t.Errorf(`with "%s" unexpected error: "%s"`, data.input, err.Error())
			return
		}
		if result == nil {
			t.Errorf(`with "%s" no result expected: "%s"`, data.input, data.expect)
		} else {
			c := compact(result)
			if c != data.expect {
				t.Errorf(`with "%s" result: "%s" expected: "%s"`, data.input, c, data.expect)
			}
		}
	}
}

func TestScannerWithBadData(t *testing.T) {
	for _, data := range nvBadTestData {
		result, err := pairs(NewConfig(), []byte(data.input))
		if err == nil {
			if result != nil {
				t.Errorf(`with "%s" error missing: "%s" unexpected result: "%s"`, data.input, data.expect, compact(result))
			} else {
				t.Errorf(`with "%s" error missing: "%s"`, data.input, data.expect)
			}
			return
		}
		if err.Error() != data.expect {
			t.Errorf(`with "%s" error: "%s" expected: "%s"`, data.input, err.Error(), data.expect)
		}
	}
}

func TestScannerWithBadValuesData(t *testing.T) {
	for _, data := range nvBadValuesTestData {
		result, err := values(NewConfig(), []byte(data.input))
		if err == nil {
			if result != nil {
				t.Errorf(`with "%s" error missing: "%s" unexpected result: "%s"`, data.input, data.expect, compactValues(result))
			} else {
				t.Errorf(`with "%s" error missing: "%s"`, data.input, data.expect)
			}
			return
		}
		if err.Error() != data.expect {
			t.Errorf(`with "%s" error: "%s" expected: "%s"`, data.input, err.Error(), data.expect)
		}
	}
}

func compact(pairs []*nameValue) string {
	buf := new(bytes.Buffer)
	for _, p := range pairs {
		if len(p.Name) > 0 {
			buf.WriteString("<")
			buf.WriteString(p.Name)
			buf.WriteString(">=<")
			buf.WriteString(p.Value)
			buf.WriteString("> ")
		} else {
			buf.WriteString("<")
			buf.WriteString(p.Value)
			buf.WriteString("> ")
		}
	}
	return string(buf.Bytes())
}

func compactValues(values []string) string {
	buf := new(bytes.Buffer)
	for _, v := range values {
		buf.WriteString("<")
		buf.WriteString(v)
		buf.WriteString("> ")
	}
	return string(buf.Bytes())
}
