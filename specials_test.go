package args_test

import (
	"testing"

	"github.com/jpvetterli/args"
)

func TestNewSpecialsPanic1(t *testing.T) {
	defer panicHandler(`expected 5 distinct special characters and not: this is too long`, t)
	args.NewSpecials(`this is too long`)
}

func TestNewSpecialPanic2(t *testing.T) {
	defer panicHandler(`expected 5 distinct special characters and not: @"":\`, t)
	args.NewSpecials(`@"":\`)
}

func TestNewSpecialsPanic3(t *testing.T) {
	defer panicHandler(`expected 5 distinct special characters and not: @:\`, t)
	args.NewSpecials(`@:\`)
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
