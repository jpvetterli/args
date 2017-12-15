package args_test

import (
	"testing"

	"github.com/jpvetterli/args"
)

func TestNewSpecialsPanic1(t *testing.T) {
	defer panicHandler(`cannot use 'E' as a special character`, t)
	args.NewSpecials(`$=[]E`)
}

func TestNewSpecialPanic2(t *testing.T) {
	defer panicHandler(`the special characters in @"":\ are not all distinct`, t)
	args.NewSpecials(`@"":\`)
}

func TestNewSpecialsPanic3(t *testing.T) {
	defer panicHandler(`exactly 5 special characters are required (@:\)`, t)
	args.NewSpecials(`@:\`)
}

func TestNewSpecialsPanic4(t *testing.T) {
	defer panicHandler(`exactly 5 special characters are required (++++++)`, t)
	args.NewSpecials(`++++++`)
}

func TestNewSpecialsEmpty(t *testing.T) {
	s := args.NewSpecials("")
	if s.SymbolPrefix() != '$' ||
		s.LeftQuote() != '[' ||
		s.RightQuote() != ']' ||
		s.Separator() != '=' ||
		s.Escape() != '\\' {
		t.Errorf("unexpected special characters: " + s.String())
	}
}

func TestNewSpecialsCustom(t *testing.T) {
	s := args.NewSpecials("@<>:\\")
	if s.SymbolPrefix() != '@' ||
		s.LeftQuote() != '<' ||
		s.RightQuote() != '>' ||
		s.Separator() != ':' ||
		s.Escape() != '\\' {
		t.Errorf("unexpected special characters: " + s.String())
	}
}

func TestRawSpecials(t *testing.T) {
	s := args.Specials{}
	if s.SymbolPrefix() != '$' ||
		s.LeftQuote() != '[' ||
		s.RightQuote() != ']' ||
		s.Separator() != '=' ||
		s.Escape() != '\\' {
		t.Errorf("unexpected special characters: " + s.String())
	}
}
