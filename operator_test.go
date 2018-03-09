package args_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jpvetterli/args"
)

func TestOperatorSkip(t *testing.T) {
	a := getParser()
	var x uint8
	a.Def("x", &x)
	if err := matchResult(
		a.Parse("x=255 --=[x=100 foo=$[UNDEF], no error because quotes are balanced]"),
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
		a.Parse("cond=[if=[$UNDEF] then=[foo=foo] else=[foo=bar]]"),
		func() error {
			if foo != "bar" {
				return fmt.Errorf(`expected "bar" not "%s"`, foo)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	if err := matchResult(
		a.Parse("$DEF=1 cond=[if=[$DEF] then=[foo=foo] else=[foo=bar]]"),
		func() error {
			if foo != "foo" {
				return fmt.Errorf(`expected "foo" not "%s"`, foo)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	if err := matchErrorMessage(
		a.Parse("cond=[if=[xyz] then=[foo=foo] else=[foo=bar]]"),
		`cond/if: parameter "xyz" not defined`,
	); err != nil {
		t.Error(err.Error())
	}

	xyz := ""
	foo = ""
	a.Def("xyz", &xyz).Opt()
	if err := matchResult(
		a.Parse("cond=[if=[xyz] then=[foo=foo] else=[foo=baz]]"),
		func() error {
			if foo != "baz" {
				return fmt.Errorf(`expected "baz" not "%s"`, foo)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

	// xyz gets an empty value, but an empty value is a value
	if err := matchResult(
		a.Parse("xyz = [] cond=[if=[xyz] then=[foo=fou] else=[foo=baz]]"),
		func() error {
			if foo != "fou" {
				return fmt.Errorf(`expected "fou" not "%s"`, foo)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorCondBeforeSet(t *testing.T) {
	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar)
	if err := matchResult(
		a.Parse("cond=[if=[$UNDEF] then=[foo=foo] else=[foo=bar]] bar=quux"),
		func() error {
			if foo != "bar" {
				return fmt.Errorf(`unexpected value: foo="%s"`, foo)
			}
			if bar != "quux" {
				return fmt.Errorf(`unexpected value: bar="%s"`, bar)
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
func TestOperatorIncludeBOM(t *testing.T) {
	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar)
	if err := matchResult(
		a.Parse("include=[testdata/include-BOM.test]"),
		func() error {
			if foo != "value of foo" || bar != "value of bar" {
				return fmt.Errorf(`unexpected results: foo="%s" bar="%s"`, foo, bar)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}

}

func TestOperatorIncludeNoAccess(t *testing.T) {
	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar)

	err := a.Parse("include=[/root/foo.txt]")
	if err == nil || strings.Index(err.Error(), "permission denied") < 0 {
		t.Errorf("unexpected error message: %s", err.Error())
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

func TestOperatorIncludeMissingFile(t *testing.T) {
	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar)

	if err := matchErrorMessage(
		a.Parse("include=[testdata/missing.test]"),
		`include: open /home/jp/go/src/github.com/jpvetterli/args/testdata/missing.test: no such file or directory`,
	); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorIncludeEmptyFile(t *testing.T) {
	a := getParser()
	foo := ""
	a.Def("foo", &foo)

	if err := matchResult(
		a.Parse("foo=bar include=[testdata/include-empty.test]"),
		func() error {
			if foo != "bar" {
				return fmt.Errorf(`unexpected result: foo="%s"`, foo)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorIncludeEmptyBOMFile(t *testing.T) {

	a := getParser()
	foo := ""
	a.Def("foo", &foo)

	if err := matchResult(
		a.Parse("foo=bar include=[testdata/include-empty-BOM.test]"),
		func() error {
			if foo != "bar" {
				return fmt.Errorf(`unexpected result: foo="%s"`, foo)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorIncludeBeforeSet(t *testing.T) {

	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar)

	if err := matchResult(
		// a.Parse("include=[testdata/include-empty.test] foo=bar"),
		a.Parse("include=[testdata/include-empty.test] include=[testdata/include.test]"),
		func() error {
			if foo != "value of foo" || bar != "value of bar" {
				return fmt.Errorf(`unexpected results: foo="%s" bar="%s"`, foo, bar)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorIncludeNoExtrac(t *testing.T) {
	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar)

	if err := matchErrorMessage(
		a.Parse("include=[testdata/include.test extractor=[irrelevant]]"),
		`include: specify extractor only with keys parameter`,
	); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorIncludeKeys1(t *testing.T) {
	a := getParser()
	user := ""
	password := ""
	a.Def("user", &user)
	a.Def("password", &password)
	if err := matchResult(
		a.Parse("include=[testdata/foreign1.test keys=[user password]]"),
		func() error {
			if user != "u648" || password != "!=.sesam567" {
				return fmt.Errorf(`unexpected results: user="%s" password="%s"`, user, password)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorIncludeKeys2(t *testing.T) {
	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar)
	if err := matchResult(
		a.Parse("include=[testdata/foreign1.test keys=[user=$sym1 password=$password]] foo=$[sym1] bar=$[password]"),
		func() error {
			if foo != "u648" || bar != "!=.sesam567" {
				return fmt.Errorf(`unexpected results: foo="%s" bar="%s"`, foo, bar)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorIncludeKeys3(t *testing.T) {
	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar)
	if err := matchResult(
		a.Parse("$sym1=usym include=[testdata/foreign1.test keys=[user=$sym1 password=$password]] foo=$[sym1] bar=$[password]"),
		func() error {
			if foo != "usym" || bar != "!=.sesam567" {
				return fmt.Errorf(`unexpected results: foo="%s" bar="%s"`, foo, bar)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}
func TestOperatorIncludeKeys3a(t *testing.T) {
	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar)
	if err := matchResult(
		a.Parse("$KEY=$sym1 $sym1=usym include=[testdata/foreign1.test keys=[user=$[KEY] password=$password]] foo=$[sym1] bar=$[password]"),
		func() error {
			if foo != "usym" || bar != "!=.sesam567" {
				return fmt.Errorf(`unexpected results: foo="%s" bar="%s"`, foo, bar)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorIncludeKeys4(t *testing.T) {
	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar)
	if err := matchResult(
		a.Parse(`include=[testdata/foreign2.test extractor=[\s*"(\S+)"\s*:\s*"(\S+)"\s*] keys=[user=$sym1 password=$password]] foo=$[sym1] bar=$[password]`),
		func() error {
			if foo != "u649" || bar != "!=.sesam568" {
				return fmt.Errorf(`unexpected results: foo="%s" bar="%s"`, foo, bar)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorIncludeKeys5(t *testing.T) {
	a := getParser()
	usr := ""
	pw := ""
	a.Def("usr", &usr)
	a.Def("pw", &pw)
	input := `include=[testdata/foreign2.test extractor=[\s*"(\S+)"\s*:\s*"(\S+)"\s*] keys=[user=usr password=$PASS]] pw=$[PASS] dump=[usr $PASS]`
	expected := "usr u649\n$PASS R !=.sesam568\n"
	output, err := captureStderr(func() error { return a.Parse(input) })
	if err != nil {
		t.Errorf("unexpected error: " + err.Error())
	}
	if output != expected {
		t.Errorf("unexpected output of dump: %s", output)
	}
	if usr != "u649" || pw != "!=.sesam568" {
		t.Errorf(`unexpected results: foo="%s" bar="%s"`, usr, pw)
	}
}

func TestOperatorIncludeKeys5BADExtractor(t *testing.T) {
	a := getParser()
	usr := ""
	pw := ""
	a.Def("usr", &usr)
	a.Def("pw", &pw)
	input := `include=[testdata/foreign2.test extractor=[***] keys=[user=usr password=$PASS]] pw=$[PASS] dump=[usr $PASS]`
	expected := "compilation of extractor \"***\" failed: error parsing regexp: missing argument to repetition operator: `*`"
	err := a.Parse(input)
	if err == nil {
		t.Errorf("error missing")
	} else {
		if err.Error() != expected {
			t.Errorf(`unexpected error: "%v", expected: "%s"`, err, expected)
		}
	}
}

func TestOperatorIncludeKeys5BOM(t *testing.T) {
	a := getParser()
	usr := ""
	pw := ""
	a.Def("usr", &usr)
	a.Def("pw", &pw)
	input := `include=[testdata/foreign2-BOM.test extractor=[\s*"(\S+)"\s*:\s*"(\S+)"\s*] keys=[user=usr password=$PASS]] pw=$[PASS] dump=[usr $PASS]`
	expected := "usr u649\n$PASS R !=.sesam568\n"
	output, err := captureStderr(func() error { return a.Parse(input) })
	if err != nil {
		t.Errorf("unexpected error: " + err.Error())
	}
	if output != expected {
		t.Errorf("unexpected output of dump: %s", output)
	}
	if usr != "u649" || pw != "!=.sesam568" {
		t.Errorf(`unexpected results: foo="%s" bar="%s"`, usr, pw)
	}
}

func TestOperatorReset(t *testing.T) {
	a := getParser()
	var x uint8
	a.Def("x", &x)
	if err := matchResult(
		a.Parse("$X=42 reset=[$X] $X=255 x=$[X] --=[x=100]"),
		func() error {
			if x != 255 {
				return fmt.Errorf("x not 255, but %d", x)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorDump(t *testing.T) {
	os.Setenv("TESTENV", "value of TESTENV")

	a := getParser()
	empty := ""
	a.Def("", &empty)
	input := "import=[$TESTENV $NONESUCH] []=[$[TESTENV]] dump=[$TESTENV $NONESUCH []]"
	expected := "$TESTENV R value of TESTENV\n? $NONESUCH\n[] value of TESTENV\n"
	output, err := captureStderr(func() error { return a.Parse(input) })
	if err != nil {
		t.Errorf("unexpected error: " + err.Error())
	}
	if output != expected {
		t.Errorf("unexpected output of dump: %s", output)
	}
}

func TestOperatorDumpWithCond(t *testing.T) {
	a := getParser()
	empty := ""
	a.Def("", &empty).Opt()
	input := "import=[$HOMEY $NONESUCH] " +
		"cond=[if=[$HOMEY] then=[$[HOMEY]] else=[dump=[comment=[$[HOMEY] not set]]]] " +
		"dump=[$HOMEY $NONESUCH []]"
	expected := "$[HOMEY] not set\n? $HOMEY\n? $NONESUCH\n[] \n"
	output, err := captureStderr(func() error { return a.Parse(input) })
	if err != nil {
		t.Errorf("unexpected error: " + err.Error())
	}
	if output != expected {
		t.Errorf("unexpected output of dump: %s", output)
	}
}

func TestOperatorDumpWithCondRenamed(t *testing.T) {
	c := args.NewConfig()
	c.SetOpName(args.OpImport, "IMPORTIEREN")
	c.SetOpName(args.OpCond, "KONDITIONAL")
	c.SetOpName(args.OpDump, "DUMPIEREN")
	a := args.CustomParser(c)
	empty := ""
	a.Def("", &empty).Opt()
	input := "IMPORTIEREN=[$HOMEY $NONESUCH] " +
		"KONDITIONAL=[if=[$HOMEY] then=[$[HOMEY]] else=[DUMPIEREN=[comment=[$[HOMEY] not set]]]] " +
		"DUMPIEREN=[$HOMEY $NONESUCH []]"
	expected := "$[HOMEY] not set\n? $HOMEY\n? $NONESUCH\n[] \n"
	output, err := captureStderr(func() error { return a.Parse(input) })
	if err != nil {
		t.Errorf("unexpected error: " + err.Error())
	}
	if output != expected {
		t.Errorf("unexpected output of dump: %s", output)
	}
}

func TestOperatorDumpWithNoEmpty(t *testing.T) {
	a := args.NewParser()
	input := "dump=[[]]"
	expected := "? []\n"
	output, err := captureStderr(func() error { return a.Parse(input) })
	if err != nil {
		t.Errorf("unexpected error: " + err.Error())
	}
	if output != expected {
		t.Errorf("unexpected output of dump: %s", output)
	}
}

func TestOperatorImport(t *testing.T) {
	a := args.NewParser()
	err := a.Parse("import=foo")
	expected := `import: "foo": symbol prefix missing ($)`
	if err == nil {
		t.Errorf("error missing")
	} else {
		if err.Error() != expected {
			t.Errorf("unexpected error: " + err.Error())
		}
	}
}

func TestOperatorMacro(t *testing.T) {
	a := getParser()
	foo := ""
	bar := ""
	a.Def("foo", &foo)
	a.Def("bar", &bar) // to test "macro before set"
	if err := matchResult(
		a.Parse("$macro=[foo=[number $[count]]] $count=1 macro=[$macro] bar=quux"),
		func() error {
			if foo != "number 1" {
				return fmt.Errorf(`unexpected value: foo="%s"`, foo)
			}
			if bar != "quux" {
				return fmt.Errorf(`unexpected value: bar="%s"`, bar)
			}
			return nil
		}); err != nil {
		t.Error(err.Error())
	}
}

func TestOperatorMacroErrors(t *testing.T) {
	a := args.NewParser()
	if err := matchErrorMessage(
		a.Parse("macro=x"),
		`macro: "x": symbol prefix missing ($)`,
	); err != nil {
		t.Error(err.Error())
	}
	if err := matchErrorMessage(
		a.Parse("macro=$x"),
		`macro: symbol "$x" undefined`,
	); err != nil {
		t.Error(err.Error())
	}
}
