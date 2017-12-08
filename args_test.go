package args

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestSymbol(t *testing.T) {
	a := NewParser(NewSpecials(""))
	if s := a.symbol("$foo"); s != "foo" {
		t.Errorf(`not "foo", but "%s"`, s)
	}
	if s := a.symbol("$$foo"); s != "" {
		t.Errorf(`not "", but "%s"`, s)
	}
	if s := a.symbol("$$"); s != "" {
		t.Errorf(`not "", but "%s"`, s)
	}
	if s := a.symbol("$"); s != "" {
		t.Errorf(`not "", but "%s"`, s)
	}
	if s := a.symbol("aaa"); s != "" {
		t.Errorf(`not "", but "%s"`, s)
	}
	if s := a.symbol(""); s != "" {
		t.Errorf(`not "", but "%s"`, s)
	}
}

func TestParamDuplicate(t *testing.T) {
	a := NewParser(NewSpecials(""))
	defer panicHandler(`parameter "a" already defined`, t)
	i := 1
	a.Def("a", &i)
	a.Def("a", &i)
}

func TestReservedPrefix(t *testing.T) {
	a := NewParser(NewSpecials(""))
	defer panicHandler(`"$a" cannot be used as parameter name or alias because it starts with the symbol prefix`, t)
	i := 1
	a.Def("$a", &i)
}

func TestParamDuplicateAlias(t *testing.T) {
	a := NewParser(NewSpecials(""))
	defer panicHandler(`synonym "A" clashes with an existing parameter name or synonym`, t)
	i := 1
	s := ""
	a.Def("a", &i).Aka("A")
	a.Def("b", &s).Aka("A")
}

func TestParamDuplicateTarget(t *testing.T) {
	a := NewParser(NewSpecials(""))
	defer panicHandler(`target for parameter "b" is already assigned`, t)
	i := 1
	a.Def("a", &i)
	a.Def("b", &i)
}

func TestParamNotPointer(t *testing.T) {
	a := NewParser(NewSpecials(""))
	defer panicHandler(`target for parameter "a" is not a pointer`, t)
	i := 1
	a.Def("a", i)
}

func TestParamSplit1(t *testing.T) {
	a := NewParser(NewSpecials(""))
	defer panicHandler(`cannot split values of parameter "x" which is not multi-valued`, t)
	var x uint8
	a.Def("x", &x).Split("foo")
}

func TestParamSplit2(t *testing.T) {
	a := NewParser(NewSpecials(""))
	defer panicHandler("compilation of split expression \"***\" for parameter \"x\" failed (error parsing regexp: missing argument to repetition operator: `*`)", t)
	var x []uint8
	a.Def("x", &x).Split("***")
}

func TestArgsMisc(t *testing.T) {

	a := NewParser(NewSpecials(""))
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

	a := NewParser(NewSpecials(""))
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

	a = NewParser(NewSpecials(""))
	var arr [3]int
	a.Def("arr", &arr)
	if err := matchErrorMessage(
		a.Parse("arr=1"),
		"Parse error on arr: 1 value specified, but exactly 3 expected",
	); err != nil {
		t.Error(err.Error())
	}

	if err := matchErrorMessage(
		a.Parse("arr=1 arr=2 arr=3 arr=4"),
		"Parse error on arr: 4 values specified, but exactly 3 expected",
	); err != nil {
		t.Error(err.Error())
	}

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

	a = NewParser(NewSpecials(""))
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

	a = NewParser(NewSpecials(""))
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

	a := NewParser(NewSpecials(""))
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
	a := NewParser(NewSpecials(""))
	var x []uint8
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
			// !?? go does NOT short circuit if len(x) tested directly: panic on x[0]
			if nothing := len(x) == 0; nothing || x[0] != 255 {
				return fmt.Errorf("x not 255, but %d", x)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	// zero length array (useless but valid)
	a = NewParser(NewSpecials(""))
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
	a := NewParser(NewSpecials(""))
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

	a = NewParser(NewSpecials(""))
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

	// the last wins
	a = NewParser(NewSpecials(""))
	y = make([]string, 0)
	a.Def("y", &y).Split(`\s*[\s:,;]\s*`)
	if err := matchResult(
		a.Parse("y=[a:b:c] y=[1:2]"),
		func() error {
			if len(y) != 2 || y[0] != "1" || y[1] != "2" {
				return fmt.Errorf("unexpected values: %v", y)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestArgsStandalone(t *testing.T) {

	a := NewParser(NewSpecials(""))
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

func TestArgsTimeScanner(t *testing.T) {
	a := NewParser(NewSpecials(""))
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
	a := NewParser(NewSpecials(""))
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
		`Parse error on s: fooBarScanner error: "b", expecting "foo" or "bar"`,
	); err != nil {
		t.Error(err.Error())
	}

	if err := matchErrorMessage(
		a.Parse("s=quux"),
		`Parse error on s: fooBarScanner error: "quux", expecting "foo" or "bar"`,
	); err != nil {
		t.Error(err.Error())
	}

	a = NewParser(NewSpecials(""))
	a.Def("s", &s).Opt().Scan(fooBarScanner)
	s = "quux"
	if err := matchErrorMessage(
		a.Parse(""),
		`Parse error on s: invalid default value: fooBarScanner error: "quux", expecting "foo" or "bar"`,
	); err != nil {
		t.Error(err.Error())
	}

	a = NewParser(NewSpecials(""))
	var i int
	a.Def("i", &i).Scan(fooBarScanner)
	if err := matchErrorMessage(
		a.Parse("i=1"),
		`Parse error on i: fooBarScanner error: "1", *string target required, not *int`,
	); err != nil {
		t.Error(err.Error())
	}

	a = NewParser(NewSpecials(""))
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
		"Parse error on anonymous parameter: 5 values specified, but exactly 3 expected",
	); err != nil {
		t.Error(err.Error())
	}

	a = NewParser(NewSpecials(""))
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

	sl = []string{"a", "b", "c", "d"}
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

	sl = make([]string, 2, 3)
	if err := matchErrorMessage(
		a.Parse("sl=foo"),
		`Parse error on sl: invalid default value at offset 1: fooBarScanner error: "", expecting "foo" or "bar"`,
	); err != nil {
		t.Error(err.Error())
	}
}

func TestArgsCustomScannerSpecialCases(t *testing.T) {

	a := NewParser(NewSpecials(""))
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

func TestArgsPrintDoc(t *testing.T) {
	a := NewParser(NewSpecials(""))
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
           foo takes any number of file names.
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
           type: int, any number of values
  array    array is a parameter taking 4 values
           type: float64, exactly 4 values
  undoc, -u
           type: float64
`
	if b.String() != expected {
		t.Errorf("PrintDoc output does not match")
		// NOTE:
		fmt.Println(
			"=== diff (begin) ===\n" +
				commonPrefix(b.String(), expected) + "\n" +
				"=== diff (end) ===")
	}
	// NOTE:	a.PrintDoc(os.Stdout)
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
func setupTestArgsPrintDoc(a *Parser) (err error) {
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
	a.Def("", &files).Aka("file").Doc("foo takes any number of file names.")
	a.Def("short", &short).Aka("bar").Opt().Doc("short is a parameter with a short name")
	a.Def("help", &help).Aka("-h").Doc("provide help")
	a.Def("yelp", &yelp).Aka("-y").Opt()
	a.Def("long-name", &long).Aka("42").Doc(
		"long-name is a parameter with a name longer than 8",
		"It also has a long explanation.")
	a.Def("slice", &sl).Doc("slice is a parameter taking any number of values")
	a.Def("array", &ar).Doc("array is a parameter taking 4 values")
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
		{"$a=b $c=$$a foo=[ $$cx ]", " $$cx "},
		{"$a=b $c=$$a foo=[ $$$c$x ]", " bx "},
		// escaping has no effect on symbols:
		{`$a=b $c=\$$a foo=$$c`, `\b`},
		{`$c=$$a foo=$$c`, `$$a`},
		// first wins:
		{"$a=b $a=x $c=$$a foo=$$c", "b"},
		{"$VAR=w3 foo=[n=a pd=$$VAR cn=[svc = $$VAR] co=2]", "n=a pd=w3 cn=[svc = w3] co=2"},
	}

	for _, data := range testData {
		a := NewParser(NewSpecials(""))
		foo := ""
		a.Def("foo", &foo)
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
