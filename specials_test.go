package args

import (
	"testing"
)

func TestNewSpecialsPanic1(t *testing.T) {
	defer panicHandler(`expected 5 distinct special characters and not: this is too long`, t)
	NewSpecials(`this is too long`)
}

func TestNewSpecialsPanic2(t *testing.T) {
	defer panicHandler(`expected 5 distinct special characters and not: @"":\`, t)
	NewSpecials(`@"":\`)
}

func TestNewSpecialsPanic3(t *testing.T) {
	defer panicHandler(`expected 5 distinct special characters and not: @:\`, t)
	NewSpecials(`@:\`)
}

func TestNewSpecialsEmpty(t *testing.T) {
	s := NewSpecials("")
	if s.SymbolPrefix() != '$' ||
		s.LeftQuote() != '[' ||
		s.RightQuote() != ']' ||
		s.Separator() != '=' ||
		s.Escape() != '\\' {
		t.Errorf("unexpected special characters: " + s.String())
	}
}

func TestNewSpecialsCustom(t *testing.T) {
	s := NewSpecials("@<>:\\")
	if s.SymbolPrefix() != '@' ||
		s.LeftQuote() != '<' ||
		s.RightQuote() != '>' ||
		s.Separator() != ':' ||
		s.Escape() != '\\' {
		t.Errorf("unexpected special characters: " + s.String())
	}
}

func TestRawSpecials(t *testing.T) {
	s := Specials{}
	if s.SymbolPrefix() != '$' ||
		s.LeftQuote() != '[' ||
		s.RightQuote() != ']' ||
		s.Separator() != '=' ||
		s.Escape() != '\\' {
		t.Errorf("unexpected special characters: " + s.String())
	}
}
