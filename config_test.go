package args_test

import (
	"fmt"
	"testing"

	"github.com/jpvetterli/args"
)

// test immutability

func TestConfigImmutability(t *testing.T) {

	c := args.NewConfig()
	// changing symbol prefix has an effect until parser created
	c.SetSpecial(args.SpecSymbolPrefix, '&')

	a := args.CustomParser(c)
	foo := false
	a.Def("foo", &foo)
	err := a.Parse("&FOO=true foo=&[FOO]")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if !foo {
		t.Errorf("foo should be true")
	}

	// changing ymbol prefix should have no effect
	c.SetSpecial(args.SpecSymbolPrefix, '@')
	err = a.Parse("reset=@FOO @FOO=false foo=@@FOO")
	expected := `reset: "@FOO": symbol prefix missing (&)`
	if err == nil {
		t.Errorf(`error missing, expected: "%s"`, expected)
	} else if err.Error() != expected {
		t.Errorf(`unexpected error: "%v", expected: "%s"`, err, expected)
	}

}

// test a few configurations

func TestNewConfig1(t *testing.T) {
	c := args.NewConfig()
	expect([5]rune{'$', '[', ']', '=', '\\'}, c, t)
}

func TestNewConfig2(t *testing.T) {
	c := args.NewConfig()
	c.SetSpecial(args.SpecSymbolPrefix, '@')
	c.SetSpecial(args.SpecOpenQuote, '<')
	c.SetSpecial(args.SpecCloseQuote, '>')
	c.SetSpecial(args.SpecSeparator, ':')
	expect([5]rune{'@', '<', '>', ':', '\\'}, c, t)
}

func expect(chars [5]rune, c *args.Config, t *testing.T) {
	s := c.GetSpecial(args.SpecSymbolPrefix)
	ch := chars[0]
	test := func() {
		if s != ch {
			t.Errorf("unexpected special character '%c', expected '%c'", s, ch)
		}
	}
	test()
	s = c.GetSpecial(args.SpecOpenQuote)
	ch = chars[1]
	test()

	s = c.GetSpecial(args.SpecCloseQuote)
	ch = chars[2]
	test()

	s = c.GetSpecial(args.SpecSeparator)
	ch = chars[3]
	test()

	s = c.GetSpecial(args.SpecEscape)
	ch = chars[4]
	test()
}

func TestOpName(t *testing.T) {
	c := args.NewConfig()
	name := "zurücksetzen"
	c.SetOpName(args.OpReset, name)
	if c.GetOpName(args.OpReset) != name {
		t.Errorf("unexpected name: %s", c.GetOpName(args.OpReset))
	}
}

// test config panics

func TestConfigPanic1(t *testing.T) {
	defer panicHandler(`cannot use ' ' as separator: not a valid special character`, t)
	testConfigString([]rune(`$[] \`))
}

func TestConfigPanic2(t *testing.T) {
	defer panicHandler(`cannot use 'E' as escape: not a valid special character`, t)
	testConfigString([]rune(`$[]=E`))
}

func TestConfigPanic3(t *testing.T) {
	defer panicHandler(`cannot use '"' as close quote: already used`, t)
	testConfigString([]rune(`@"":\`))
}

func testConfigString(s []rune) {
	if len(s) != 5 {
		panic(fmt.Errorf(`length of "%v" not 5`, s))
	}
	config := args.NewConfig()
	config.SetSpecial(args.SpecSymbolPrefix, s[0])
	config.SetSpecial(args.SpecOpenQuote, s[1])
	config.SetSpecial(args.SpecCloseQuote, s[2])
	config.SetSpecial(args.SpecSeparator, s[3])
	config.SetSpecial(args.SpecEscape, s[4])
}

func TestConfigPanic4(t *testing.T) {
	defer panicHandler(`cannot set name of 0 to "FOO": name already used`, t)
	c := args.NewConfig()
	c.SetOpName(args.OpImport, "FOO")
	c.SetOpName(args.OpCond, "FOO")
}

func TestConfigPanic5(t *testing.T) {
	defer panicHandler(`cannot use '$' as escape: already used`, t)
	c := args.NewConfig()
	c.SetSpecial(args.SpecEscape, '$')
}
