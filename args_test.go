package args_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jpvetterli/args"
)

func TestParamDuplicate(t *testing.T) {
	a := getParser()
	defer panicHandler(`parameter "a" already defined`, t)
	i := 1
	a.Def("a", &i)
	a.Def("a", &i)
}

func TestReservedPrefix(t *testing.T) {
	a := getParser()
	defer panicHandler(`"$a" cannot be used as parameter name or alias because it includes the character '$'`, t)
	i := 1
	a.Def("$a", &i)
}

func TestInvalidChar(t *testing.T) {
	a := getParser()
	defer panicHandler(`"a b" cannot be used as parameter name or alias because it includes the character ' '`, t)
	i := 1
	a.Def("a b", &i)
}

func TestParamDuplicateAlias(t *testing.T) {
	a := getParser()
	defer panicHandler(`synonym "A" clashes with an existing parameter name or synonym`, t)
	i := 1
	s := ""
	a.Def("a", &i).Aka("A")
	a.Def("b", &s).Aka("A")
}

func TestParamDuplicateTarget(t *testing.T) {
	a := getParser()
	defer panicHandler(`target for parameter "b" is already assigned`, t)
	i := 1
	a.Def("a", &i)
	a.Def("b", &i)
}

func TestParamNotPointer(t *testing.T) {
	a := getParser()
	defer panicHandler(`target for parameter "a" is not a pointer`, t)
	i := 1
	a.Def("a", i)
}

func TestParamSplit1(t *testing.T) {
	a := getParser()
	defer panicHandler(`cannot split values of parameter "x" which is not multi-valued`, t)
	var x uint8
	a.Def("x", &x).Split("foo")
}

func TestParamSplit2(t *testing.T) {
	a := getParser()
	defer panicHandler("compilation of split expression \"***\" for parameter \"x\" failed (error parsing regexp: missing argument to repetition operator: `*`)", t)
	var x []uint8
	a.Def("x", &x).Split("***")
}

func TestParamOperator1(t *testing.T) {
	a := getParser()
	defer panicHandler(`parameter name "--" is the name of an operator`, t)
	i := 1
	a.Def("--", &i)
}

func TestParamOperator2(t *testing.T) {
	a := getParser()
	defer panicHandler(`parameter name "include" is the name of an operator`, t)
	i := 1
	a.Def("include", &i)
}

func TestArgsMisc(t *testing.T) {

	a := getParser()
	foo := false
	bar := true
	a.Def("foo", &foo)
	a.Def("bar", &bar)

	if err := matchResult(
		a.ParseStrings([]string{"foo=true", "bar=false"}),
		func() error {
			if !foo || bar {
				return fmt.Errorf("foo and/or bar not set")
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

}

func TestArgsOptionalAndRepeatable(t *testing.T) {

	a := getParser()
	x := 3.14
	def := a.Def("x", &x)
	if err := matchErrorMessage(
		a.Parse(""),
		"Parse error on x: mandatory parameter not set",
	); err != nil {
		t.Error(err.Error())
	}

	def.Opt()
	if err := matchResult(
		a.Parse(""),
		func() error {
			if x != 3.14 {
				return fmt.Errorf("x not 3.14, but %f", x)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	var arr [3]int
	a.Def("arr", &arr)
	if err := matchErrorMessage(
		a.Parse("arr=1"),
		"Parse error on arr: 1 value specified but exactly 3 expected",
	); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	a.Def("arr", &arr)
	if err := matchErrorMessage(
		a.Parse("arr=1 arr=2 arr=3 arr=4"),
		"Parse error on arr: too many values specified, expected 3",
	); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	a.Def("arr", &arr)
	if err := matchResult(
		a.Parse("arr=1 arr=2 arr=3"),
		func() error {
			if arr[0] != 1 || arr[1] != 2 || arr[2] != 3 {
				return fmt.Errorf("incorrect values: %v", arr)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	sli := make([]int, 3, 3)
	a.Def("sli", &sli)
	if err := matchResult(
		a.Parse("sli=1 sli=2"),
		func() error {
			if sli[0] != 1 || sli[1] != 2 || sli[2] != 0 {
				return fmt.Errorf("incorrect values: %v", sli)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	sli = make([]int, 0)
	a.Def("sli", &sli)
	if err := matchResult(
		a.Parse("sli=10 sli=20"),
		func() error {
			if len(sli) != 2 || sli[0] != 10 || sli[1] != 20 {
				return fmt.Errorf("incorrect values: %v", sli)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestArgsTypesSingleValue(t *testing.T) {

	a := getParser()
	var x uint8
	a.Def("x", &x)
	if err := matchErrorMessage(
		a.Parse("x= 1000"),
		`Parse error on x: strconv.ParseUint: parsing "1000": value out of range`,
	); err != nil {
		t.Error(err.Error())
	}

	if err := matchResult(
		a.Parse("x=255"),
		func() error {
			if x != 255 {
				return fmt.Errorf("x not 255, but %d", x)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	// the last wins
	if err := matchResult(
		a.Parse("x=255 x=100"),
		func() error {
			if x != 100 {
				return fmt.Errorf("x not 100, but %d", x)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestArgsTypesMultiValue(t *testing.T) {
	a := getParser()
	var x []uint8
	a.Def("x", &x)
	if err := matchErrorMessage(
		a.Parse("x= 1000"),
		`Parse error on x: strconv.ParseUint: parsing "1000": value out of range`,
	); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	a.Def("x", &x)
	if err := matchResult(
		a.Parse("x=255"),
		func() error {
			// !?? go does NOT short circuit if len(x) tested directly: panic on x[0]
			if nothing := len(x) == 0; nothing || x[0] != 255 {
				return fmt.Errorf("x not 255, but %d", x)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	// zero length array (useless but valid)
	a = getParser()
	var y [0]int
	a.Def("y", &y)
	if err := matchResult(
		a.Parse(""),
		func() error {
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

}

func TestArgsSplit(t *testing.T) {
	a := getParser()
	var x []uint8
	a.Def("x", &x).Split(`\s*:\s*`)

	if err := matchResult(
		a.Parse("x=[3 : 2:1]"),
		func() error {
			if len(x) != 3 || x[0] != 3 || x[1] != 2 || x[2] != 1 {
				return fmt.Errorf("unexpected values: %v", x)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	var y []string
	a.Def("y", &y).Split(`\s*[\s:,;]\s*`)

	if err := matchResult(
		a.Parse("y=[a : :c]"),
		func() error {
			if len(y) != 3 || y[0] != "a" || y[1] != "" || y[2] != "c" {
				return fmt.Errorf("unexpected values: %v", y)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	y = make([]string, 0)
	a.Def("y", &y).Split(`\s*[\s:,;]\s*`)
	if err := matchResult(
		a.Parse("y=[a b;c] y=[1,2:]"),
		func() error {
			if !reflect.DeepEqual(y, []string{"a", "b", "c", "1", "2", ""}) {
				return fmt.Errorf("unexpected values: %v", y)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	y = make([]string, 0)
	a.Def("y", &y).Split(`---`)
	if err := matchResult(
		a.Parse("y=[a---b--c---d---]"),
		func() error {
			if !reflect.DeepEqual(y, []string{"a", "b--c", "d", ""}) {
				return fmt.Errorf("unexpected values: %v", y)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestArgsStandaloneName(t *testing.T) {

	a := getParser()
	foo := false
	a.Def("foo", &foo).Aka("FOO")

	if err := matchResult(
		a.Parse("foo"),
		func() error {
			if !foo {
				return fmt.Errorf("bool parameter not set")
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	if err := matchResult(
		a.Parse("FOO"),
		func() error {
			if !foo {
				return fmt.Errorf("bool parameter not set")
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	// the last wins
	foo = false
	if err := matchResult(
		a.Parse("FOO foo foo=false foo"),
		func() error {
			if !foo {
				return fmt.Errorf("bool parameter not set")
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

}

func TestArgsStandaloneValue(t *testing.T) {

	a := getParser()
	s := []string{}
	a.Def("", &s).Aka("ANONYMOUS")
	err := a.Parse("abc ANONYMOUS=123 [] = 456")
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if !reflect.DeepEqual(s, []string{"abc", "123", "456"}) {
		t.Errorf("unexpected values: %v", s)
	}

	a = getParser()
	s = make([]string, 0, 1)
	a.Def("", &s).Aka("ANONYMOUS")
	err = a.Parse("abc ANONYMOUS=123")
	expected := "Parse error on anonymous parameter: 2 values specified, at most 1 expected"
	if err == nil {
		t.Errorf("missing error: %s", expected)
	} else if err.Error() != expected {
		t.Errorf(`unexpected error: "%s", expected: "%s"`, err.Error(), expected)
	}

	a = getParser()
	s = make([]string, 0, 0)
	a.Def("", &s)
	err = a.Parse("[contains an $$unresolved ref]")
	expected = `cannot resolve standalone value "contains an $$unresolved ref"`
	if err == nil {
		t.Errorf("missing error: %s", expected)
	} else if err.Error() != expected {
		t.Errorf(`unexpected error: "%s", expected: "%s"`, err.Error(), expected)
	}

	a = getParser()
	x := ""
	a.Def("", &x).Verbatim()
	err = a.Parse("[contains an $$unresolved ref]")
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if x != "contains an $$unresolved ref" {
		t.Errorf("unexpected value: %s", x)
	}

	a = getParser()
	s = []string{}
	a.Def("", &s).Verbatim()
	err = a.Parse("[contains an $$unresolved ref]")
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if !reflect.DeepEqual(s, []string{"contains an $$unresolved ref"}) {
		t.Errorf("unexpected values: %v", s)
	}

	a = getParser()
	arr := [1]string{}
	a.Def("", &arr).Verbatim()
	err = a.Parse("[contains an $$unresolved ref]")
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if arr[0] != "contains an $$unresolved ref" {
		t.Errorf("unexpected values: %v", s)
	}

	a = getParser()
	sl := []string{}
	a.Def("z", &sl)
	err = a.Parse(`$X = bar z=foo z=\= z=\[x: z=\$$X\]`)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if !reflect.DeepEqual(sl, []string{"foo", "=", "[x:", `\bar]`}) {
		t.Errorf("unexpected values: %v", sl)
	}

	a = getParser()
	sl = []string{}
	a.Def("", &sl)
	// same as previous but anonymous
	err = a.Parse(`$X = bar foo \= \[x: \$$X\\\]`)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if !reflect.DeepEqual(sl, []string{"foo", "=", "[x:", `\bar]`}) {
		t.Errorf("unexpected values: %v", sl)
	}

	a = getParser()
	sl = []string{}
	a.Def("", &sl)
	// same as previous but anonymous
	err = a.Parse(`$X = bar \$$X \\\= \[x:\ :x\]`)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if !reflect.DeepEqual(sl, []string{`\bar`, `\=`, "[x: :x]"}) {
		t.Errorf("unexpected values: %v", sl)
	}

	a = getParser()
	sl = []string{}
	a.Def("", &sl)
	err = a.Parse(`$X = bar foo \= \[x: \$$X\\\\`)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if !reflect.DeepEqual(sl, []string{"foo", "=", "[x:", `\bar\`}) {
		t.Errorf("unexpected values: %v", sl)
	}

	defer panicHandler(`anonymous parameter cannot be verbatim because its target of type *int cannot take a string`, t)
	a = getParser()
	y := 1
	a.Def("", &y).Verbatim()
}

func TestArgsRecursive(t *testing.T) {
	a := getParser()
	foo := ""
	a.Def("foo", &foo)
	err := a.Parse("$quux=QUUX $macro=[foo=[bar $$quux]] $$macro")
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if foo != "bar QUUX" {
		t.Errorf("unexpected value: %s", foo)
	}
}

func TestArgsTimeScanner(t *testing.T) {
	a := getParser()
	var x time.Time
	defx := a.Def("x", &x)
	if err := matchErrorMessage(
		a.Parse("x= 1000"),
		`Parse error on x: target for value "1000" has unsupported type time.Time`,
	); err != nil {
		t.Error(err.Error())
	}
	defx.Scan(func(value string, target interface{}) error {
		if s, ok := target.(*time.Time); ok {
			if t, err := time.Parse(time.RFC3339, value); err == nil {
				*s = t
			} else {
				return err
			}
			return nil
		}
		return fmt.Errorf(`timeScanner error: "%s", *time.Time target required, not %T`, value, target)
	})
	now := time.Now()
	if err := matchResult(
		a.Parse("x=["+now.Format(time.RFC3339Nano)+"]"),
		func() error {
			if !now.Equal(x) {
				return fmt.Errorf("unexpected time difference: now: <%v> x: <%v>", now, x)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

// fooBarScanner is a custom scanner accepting only foo and bar as valid inputs
func fooBarScanner(value string, target interface{}) error {

	verify := func(in string) error {
		if in == "foo" || in == "bar" {
			return nil
		}
		return fmt.Errorf(`fooBarScanner error: "%s", expecting "foo" or "bar"`, in)
	}

	if s, ok := target.(*string); ok {
		if err := verify(value); err != nil {
			return err
		}
		*s = value
		return nil
	}
	return fmt.Errorf(`fooBarScanner error: "%s", *string target required, not %T`, value, target)
}

func TestArgsCustomScanner(t *testing.T) {

	var s string
	a := getParser()
	a.Def("s", &s).Scan(fooBarScanner)
	if err := matchResult(
		a.Parse("s=foo"),
		func() error {
			if s != "foo" {
				return fmt.Errorf("not foo, but %s", s)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	// the last wins, even if the value is wrong
	if err := matchErrorMessage(
		a.Parse("s=a s=foo s = b"),
		`Parse error on s: fooBarScanner error: "a", expecting "foo" or "bar"`,
	); err != nil {
		t.Error(err.Error())
	}

	if err := matchErrorMessage(
		a.Parse("s=quux"),
		`Parse error on s: fooBarScanner error: "quux", expecting "foo" or "bar"`,
	); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	a.Def("s", &s).Opt().Scan(fooBarScanner)
	s = "quux"
	if err := matchErrorMessage(
		a.Parse(""),
		`Parse error on s: invalid default value: fooBarScanner error: "quux", expecting "foo" or "bar"`,
	); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	var i int
	a.Def("i", &i).Scan(fooBarScanner)
	if err := matchErrorMessage(
		a.Parse("i=1"),
		`Parse error on i: fooBarScanner error: "1", *string target required, not *int`,
	); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	ss := [3]string{}
	a.Def("", &ss).Scan(fooBarScanner)
	if err := matchResult(
		a.Parse("foo bar foo"),
		func() error {
			if len(ss) != 3 || ss[0] != "foo" || ss[1] != "bar" || ss[2] != "foo" {
				return fmt.Errorf("not [foo bar foo], but %v", ss)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	if err := matchErrorMessage(
		a.Parse("foo bar foo foo foo"),
		"Parse error on anonymous parameter: too many values specified, expected 3",
	); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	sl := make([]string, 0)
	a.Def("sl", &sl).Scan(fooBarScanner)
	if err := matchResult(
		a.Parse("sl=foo sl=bar sl=foo"),
		func() error {
			if len(sl) != 3 || sl[0] != "foo" || sl[1] != "bar" || sl[2] != "foo" {
				return fmt.Errorf("not [foo bar foo], but %v", sl)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	sl = make([]string, 0)
	a.Def("sl", &sl).Scan(fooBarScanner)
	sl = make([]string, 1, 3)
	if err := matchResult(
		a.Parse("sl=foo sl=bar sl=foo"),
		func() error {
			if len(sl) != 3 || sl[0] != "foo" || sl[1] != "bar" || sl[2] != "foo" {
				return fmt.Errorf("not [foo bar foo], but %v", sl)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	sl = make([]string, 1, 3)
	a.Def("sl", &sl).Scan(fooBarScanner)
	if err := matchResult(
		a.Parse("sl=foo sl=bar sl=foo"),
		func() error {
			if len(sl) != 3 || sl[0] != "foo" || sl[1] != "bar" || sl[2] != "foo" {
				return fmt.Errorf("not [foo bar foo], but %v", sl)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	sl = []string{"a", "b", "c", "d"}
	a.Def("sl", &sl).Scan(fooBarScanner)
	if err := matchResult(
		a.Parse("sl=foo sl=bar sl=foo sl=foo"),
		func() error {
			if len(sl) != 4 || sl[0] != "foo" || sl[1] != "bar" || sl[2] != "foo" || sl[3] != "foo" {
				return fmt.Errorf("not [foo bar foo], but %v", sl)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	a = getParser()
	sl = make([]string, 2, 3)
	a.Def("sl", &sl).Scan(fooBarScanner)
	if err := matchErrorMessage(
		a.Parse("sl=foo"),
		`Parse error on sl: invalid default value at offset 1: fooBarScanner error: "", expecting "foo" or "bar"`,
	); err != nil {
		t.Error(err.Error())
	}
}

func TestArgsCustomScannerSpecialCases(t *testing.T) {

	a := getParser()
	sl := make([]string, 2, 3)
	def := a.Def("sl", &sl).Scan(fooBarScanner)
	if err := matchErrorMessage(
		a.Parse(""),
		`Parse error on sl: invalid default value at offset 0: fooBarScanner error: "", expecting "foo" or "bar"`,
	); err != nil {
		t.Error(err.Error())
	}

	// multi-valued parameter made optional with zero length
	sl = sl[:0]
	if err := matchResult(
		a.Parse(""),
		func() error {
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	// multi-valued parameter cannot be made optional with Opt (panic)

	defer panicHandler(`parameter "sl" is multi-valued and cannot be optional (hint: use a slice with length 0 instead)`, t)
	def.Opt()

	t.Errorf("this statement should not have been executed (panic)")
}

func TestOperatorSkip(t *testing.T) {
	a := getParser()
	var x uint8
	a.Def("x", &x)
	if err := matchResult(
		a.Parse("x=255 --=[x=100 foo=$$UNDEF, no error because quotes are balanced]"),
		func() error {
			if x != 255 {
				return fmt.Errorf("x not 255, but %d", x)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorCond(t *testing.T) {
	a := getParser()
	foo := ""
	a.Def("foo", &foo)
	if err := matchResult(
		a.Parse("cond=[if=[UNDEF] then=[foo=foo] else=[foo=bar]]"),
		func() error {
			if foo != "bar" {
				return fmt.Errorf(`expected "bar" not "%s"`, foo)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	if err := matchResult(
		a.Parse("$DEF=1 cond=[if=[DEF] then=[foo=foo] else=[foo=bar]]"),
		func() error {
			if foo != "foo" {
				return fmt.Errorf(`expected "foo" not "%s"`, foo)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorInclude(t *testing.T) {
	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar)
	if err := matchResult(
		a.Parse("include=[testdata/include.test]"),
		func() error {
			if foo != "value of foo" || bar != "value of bar" {
				return fmt.Errorf(`unexpected results: foo="%s" bar="%s"`, foo, bar)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

}

func TestOperatorIncludeCycle(t *testing.T) {
	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar)

	if err := matchErrorMessage(
		a.Parse("include=[testdata/cycle.test]"),
		`cyclical include dependency with file "testdata/cycle.test"`,
	); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorReset(t *testing.T) {
	a := getParser()
	var x uint8
	a.Def("x", &x)
	if err := matchResult(
		a.Parse("$X=42 reset=[X] $X=255 x=$$X --=[x=100]"),
		func() error {
			if x != 255 {
				return fmt.Errorf("x not 255, but %d", x)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorImport(t *testing.T) {
	a := getParser()
	gopath := ""
	path := ""
	a.Def("gopath", &gopath)
	a.Def("path", &path)
	if err := matchResult(
		a.Parse("import=[PATH GOPATH GOBBLEDYGOOK] gopath=$$GOPATH path=$$PATH"),
		func() error {
			// just testing for no error
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	// capture standard error into a string
	stderr := os.Stderr
	r, w, e := os.Pipe()
	if e != nil {
		t.Errorf("meta error opening pipe: %v", e)
	}
	os.Stderr = w
	ch := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, e := io.Copy(&buf, r)
		r.Close()
		if e != nil {
			t.Errorf("meta error copying from pipe: %v", e)
		}
		ch <- buf.String()
	}()
	defer func() {
		w.Close()
		os.Stderr = stderr
		out := <-ch
		if strings.Index(out, "testing...") != 0 ||
			strings.Index(out, "GOPATH (R)") < 0 ||
			strings.Index(out, "PATH (R)") < 0 ||
			strings.Index(out, "GOBBLEDYGOOK (U)") < 0 {
			t.Errorf("unexpected output of dump: %s", out)
		}
	}()

	err := a.Parse("dump=[comment=[testing...] GOPATH PATH GOBBLEDYGOOK]")
	if err != nil {
		t.Errorf("unexpected error: " + err.Error())
	}
}

func TestArgsPrintDoc(t *testing.T) {
	a := getParser()
	err := setupTestArgsPrintDoc(a)
	if err != nil {
		t.Errorf(`unexpected error: "%s"`, err.Error())
		return
	}
	b := bytes.Buffer{}
	a.PrintDoc(&b)
	expected := `Usage: foo <parameters> <filename> ...
The foo command does things.

Parameters:
  (nameless), file
           foo takes any number of file names
           type: string, any number of values
  short, bar
           short is a parameter with a short name
           type: string, optional (default: )
  help, -h provide help
           type: bool
  yelp, -y type: bool, optional (default: true)
  long-name, 42
           long-name is a parameter with a name longer than 8
           It also has a long explanation.
           type: string, 0-2 values (default: [a1 b2])
  slice    slice is a parameter taking any number of values
           type: int, split: ---, any number of values
  array    array is a parameter taking 4 values
           type: float64, split: \s*:\s*, exactly 4 values
  undoc, -u
           type: float64
Special characters:
  $        symbol prefix
  =        name-value separator
  [        opening quote
  ]        closing quote
  \        escape
`
	if b.String() != expected {
		t.Errorf("PrintDoc output does not match")
		// NOTE:
		fmt.Println(
			"=== diff (begin) ===\n" +
				commonPrefix(b.String(), expected) + "\n" +
				"=== diff (end) ===")
	}
	// NOTE: a.PrintDoc(os.Stdout)
	a = nil // reclaim memory
}

// helpers

// commonPrefix returns prefix common to two strings
func commonPrefix(s1, s2 string) string {
	min, max := s1, s1
	switch {
	case s2 < min:
		min = s2
	case s2 > max:
		max = s2
	}
	for i := 0; i < len(min) && i < len(max); i++ {
		if min[i] != max[i] {
			return min[:i]
		}
	}
	return min
}

// setupTestArgsPrintDoc sets up parser for TestArgsPrintDoc, catching panics
func setupTestArgsPrintDoc(a *args.Parser) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	var files []string
	var help bool
	yelp := true
	var short string
	long := []string{"a1", "b2"}
	sl := make([]int, 0)
	var ar [4]float64
	var undoc float64
	a.Doc(
		"Usage: foo <parameters> <filename> ...",
		"The foo command does things.",
		"",
		"Parameters:")
	a.Def("", &files).Aka("file").Doc("foo takes any number of file names")
	a.Def("short", &short).Aka("bar").Opt().Doc("short is a parameter with a short name")
	a.Def("help", &help).Aka("-h").Doc("provide help")
	a.Def("yelp", &yelp).Aka("-y").Opt()
	a.Def("long-name", &long).Aka("42").Doc(
		"long-name is a parameter with a name longer than 8",
		"It also has a long explanation.")
	a.Def("slice", &sl).Doc("slice is a parameter taking any number of values").Split(`---`)
	a.Def("array", &ar).Doc("array is a parameter taking 4 values").Split(`\s*:\s*`)
	a.Def("undoc", &undoc).Aka("-u")
	return
}

func TestSubsBasic(t *testing.T) {

	var testData = []struct {
		input  string
		expect string
	}{
		{"$a=b $c=$$a foo=$$c", "b"},
		{"$a=b $c=$$a foo=[ $$c ]", " b "},
		// order relevant:
		{"foo=[ $$c ] $c=$$a $a=b", " $$c "},
		{"$a=b $c=$$a foo=[ $$cx ]", " $$cx "},
		{"$a=b $c=$$a foo=[ $$$c$x ]", " bx "},
		// // escaping has no effect on symbols:
		{`$a=b $c=\$$a foo=$$c`, `\b`},
		{`$c=$$a foo=$$c`, `$$a`},
		// first wins:
		{"$a=b $a=x $c=$$a foo=$$c", "b"},
		{"$VAR=w3 foo=[n=a pd=$$VAR cn=[svc = $$VAR] co=2]", "n=a pd=w3 cn=[svc = w3] co=2"},
	}

	for _, data := range testData {
		a := getParser()
		foo := ""
		a.Def("foo", &foo).Verbatim()
		if err := matchResult(
			a.Parse(data.input),
			func() error {
				if foo != data.expect {
					return fmt.Errorf(`input: "%s" result: "%s" expect: "%s"`, data.input, foo, data.expect)
				}
				return nil
			}); err != nil {
			t.Error(err.Error())
		}
	}
}

func TestSubsCycle(t *testing.T) {

	var testData = []struct {
		input  string
		expect string
	}{
		{"$a=$$b $b=$$a foo=$$b", `Parse error on foo: cyclical symbol definition detected: "b"`},
		{"$a=$$b $b=$$a foo=$$a", `Parse error on foo: cyclical symbol definition detected: "a"`},
		{"$a=$$b $b=$$c $c=$$d $d=$$e $e=$$a foo=$$a", `Parse error on foo: cyclical symbol definition detected: "a"`},
	}
	for _, data := range testData {
		a := getParser()
		foo := ""
		a.Def("foo", &foo)
		err := a.Parse(data.input)
		if err == nil {
			t.Errorf(`expected error missing: "%s"`, data.expect)
		} else if err.Error() != data.expect {
			t.Errorf(`unexpected error message: "%s" expected: "%s"`, err.Error(), data.expect)
		}
	}
}

func TestSubsName(t *testing.T) {
	a := getParser()
	foo := ""
	a.Def("foo", &foo)
	err := a.Parse("$oo=oo $foo=f$$oo $$foo=bar")
	if err != nil {
		t.Errorf(`unexpected error message: "%s"`, err.Error())
	}
	if foo != "bar" {
		t.Errorf(`foo: "%s", instead of "bar"`, foo)
	}

	a = getParser()
	expected := `cannot resolve name in "$$quux = bar"`
	err = a.Parse("$$quux=bar")
	if err == nil {
		t.Errorf(`missing error message: "%s"`, expected)
	} else if err.Error() != expected {
		t.Errorf(`unexpected error message: "%s", expected: "%s"`, err.Error(), expected)
	}

	a = getParser()
	foo = ""
	a.Def("foo", &foo)
	expected = `cannot resolve name in "$$foo = bar"`
	err = a.Parse("$$foo=bar $foo=f$$oo $oo=oo")
	if err == nil {
		t.Errorf(`missing error message: "%s"`, expected)
	} else if err.Error() != expected {
		t.Errorf(`unexpected error message: "%s", expected: "%s"`, err.Error(), expected)
	}
}

func TestSubsMacro1(t *testing.T) {
	a := getParser()
	foox := ""
	fooa := ""
	a.Def("foox", &foox).Verbatim()
	a.Def("fooa", &fooa).Verbatim()
	err := a.Parse("$BODY = [arg1=$$ARG1 arg2=$$ARG2] " +
		"foox=[$ARG1=x $ARG2=y $$BODY] fooa=[$ARG1=a $ARG2=b $$BODY]")

	expectedFoox := "$ARG1=x $ARG2=y arg1=$$ARG1 arg2=$$ARG2"
	expectedFooa := "$ARG1=a $ARG2=b arg1=$$ARG1 arg2=$$ARG2"

	if err != nil {
		t.Errorf(`unexpected error message: "%s"`, err.Error())
	}
	if foox != expectedFoox {
		t.Errorf(`foox: "%s", instead of "%s"`, foox, expectedFoox)
	}
	if fooa != expectedFooa {
		t.Errorf(`fooa: "%s", instead of "%s"`, fooa, expectedFooa)
	}

	a = getParser()
	arg1 := ""
	arg2 := ""
	a.Def("arg1", &arg1)
	a.Def("arg2", &arg2)
	a.Parse(foox)
	if arg1 != "x" || arg2 != "y" {
		t.Errorf(`nested parsing: "%s, %s", instead of "x, y"`, arg1, arg2)
	}
	a = getParser()
	arg1 = ""
	arg2 = ""
	a.Def("arg1", &arg1)
	a.Def("arg2", &arg2)
	a.Parse(fooa)
	if arg1 != "a" || arg2 != "b" {
		t.Errorf(`nested parsing: "%s, %s", instead of "a, b"`, arg1, arg2)
	}
}

// panicHandler triggers a testing error if panic message differs from expected
func panicHandler(expected string, t *testing.T) {
	err := recover()
	if err == nil {
		if len(expected) > 0 {
			t.Errorf(`(recovery) no error caught, expected: "%s"`, expected)
		}
	} else {
		if e, ok := err.(error); !ok {
			t.Errorf("(recovery) unexpected error: %v", err)
		} else {
			if e.Error() != expected {
				t.Errorf(`(recovery) unexpected error message: "%s" expected: "%s"`, err, expected)
			}
		}
	}
}

// matchErrorMessage returns nil if the error message matches, else an error.
func matchErrorMessage(err error, expected string) error {
	if err == nil {
		return fmt.Errorf(`expected error message missing: "%s"`, expected)
	} else if err.Error() != expected {
		return fmt.Errorf(`unexpected error message: "%s", expected: "%s"`, err.Error(), expected)
	}
	return nil
}

// matchResult returns nil if error is nil and test returns nil, else an error.
func matchResult(err error, test func() error) error {
	if err != nil {
		return fmt.Errorf(`unexpected error: "%s"`, err.Error())
	}
	if e := test(); e != nil {
		return e
	}
	return nil
}

func getParser() *args.Parser {
	return args.NewParser(nil)
}
