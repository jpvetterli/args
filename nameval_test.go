package args

import (
	"bytes"
	"reflect"
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
	scanner := newNamevalScanner(NewSpecials(""))
	for _, data := range nvGoodTestData {
		result, err := scanner.pairs([]byte(data.input))
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
	scanner := newNamevalScanner(NewSpecials(""))
	for _, data := range nvGoodValuesTestData {
		result, err := scanner.values([]byte(data.input))
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
	// ? Symbol prefix not used
	scanner := newNamevalScanner(NewSpecials("?{}:!"))
	for _, data := range nvCustomTestData {
		result, err := scanner.pairs([]byte(data.input))
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
	scanner := newNamevalScanner(NewSpecials(""))
	for _, data := range nvBadTestData {
		result, err := scanner.pairs([]byte(data.input))
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
	scanner := newNamevalScanner(NewSpecials(""))
	for _, data := range nvBadValuesTestData {
		result, err := scanner.values([]byte(data.input))
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

func TestNameValues(t *testing.T) {
	input := "d = f a=b a=c g h i D=[D is d's synonym] A = [A is a's synonym]"
	scanner := newNamevalScanner(NewSpecials(""))
	dict, list, err := scanner.Scan([]byte(input), map[string]string{"h": "h", "": "", "a": "a", "d": "d", "A": "a", "D": "d"})
	if err != nil {
		t.Errorf(`with "%s" unexpected error from NameValues: "%s"`, input, err.Error())
	}
	if len(list) != 4 {
		t.Errorf(`with "%s", the list does not have 3 values`, input)
	}
	if list[0][0] != "d" || list[1][0] != "a" || list[2][0] != "" {
		t.Errorf(`with "%s", the list elements are not in the expected sequence`, input)
	}
	if !reflect.DeepEqual(dict["d"], list[0]) {
		t.Errorf(`with "%s", the map and list values for "d" are not identical`, input)
	}
	if !reflect.DeepEqual(dict["a"], list[1]) {
		t.Errorf(`with "%s", the map and list values for "a" are not identical`, input)
	}
	if !reflect.DeepEqual(dict[""], list[2]) {
		t.Errorf(`with "%s", the map and list values for the anonymous entry are not identical`, input)
	}

	a := dict["a"]
	if a == nil {
		t.Errorf(`with "%s", "a" not found in name-value map`, input)
	}
	if len(a) != 4 {
		t.Errorf(`with "%s", "a" does not have two values`, input)
	}
	if a[0] != "a" || a[1] != "b" || a[2] != "c" || a[3] != "A is a's synonym" {
		t.Errorf(`with "%s", "a" does has wrong values`, input)
	}
	d := dict["d"]
	if d == nil || len(d) != 3 || d[0] != "d" || d[1] != "f" || d[2] != "D is d's synonym" {
		t.Errorf(`with "%s", something wrong with "d"`, input)
	}
	h := dict["h"]
	if h == nil || len(h) != 1 || h[0] != "h" {
		t.Errorf(`with "%s", something wrong with "h"`, input)
	}
	anon := dict[""]
	if anon == nil || len(anon) != 3 || anon[0] != "" || anon[1] != "g" || anon[2] != "i" {
		t.Errorf(`with "%s", something wrong with standalone values`, input)
	}
}

func TestRepeatedStandaloneNames(t *testing.T) {
	expected := `standalone name "a" cannot be repeated`
	input := "a a b" // b is okay if empty symbol defined
	scanner := newNamevalScanner(NewSpecials(""))
	_, _, err := scanner.Scan([]byte(input), map[string]string{"a": "a", "": ""})
	if err == nil {
		t.Error("expected error missing")
	} else {
		if err.Error() != expected {
			t.Errorf(`error %q differs from expected %q`, err.Error(), expected)
		}
	}

	expected = `standalone name "A" (synonym of "a") cannot be repeated`
	input = "a A b" // b is okay if empty symbol defined
	_, _, err = scanner.Scan([]byte(input), map[string]string{"a": "a", "A": "a", "": ""})
	if err == nil {
		t.Error("expected error missing")
	} else {
		if err.Error() != expected {
			t.Errorf(`error %q differs from expected %q`, err.Error(), expected)
		}
	}

	expected = `name "A" (synonym of "a") can only be repeated with values, but not standalone`
	input = "a=x A b" // b is okay if empty symbol defined
	_, _, err = scanner.Scan([]byte(input), map[string]string{"a": "a", "A": "a", "": ""})
	if err == nil {
		t.Error("expected error missing")
	} else {
		if err.Error() != expected {
			t.Errorf(`error %q differs from expected %q`, err.Error(), expected)
		}
	}

	expected = `cannot add value "x" to standalone name "A" (synonym of "a")`
	input = "a A=x b" // b is okay if empty symbol defined
	_, _, err = scanner.Scan([]byte(input), map[string]string{"a": "a", "A": "a", "": ""})
	if err == nil {
		t.Error("expected error missing")
	} else {
		if err.Error() != expected {
			t.Errorf(`error %q differs from expected %q`, err.Error(), expected)
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
