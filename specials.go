package args

import "errors"

// Specials is a set of 5 characters playing a special role when parsing command
// line arguments. These characters are explained in detail in the package
// documentation.
type Specials struct {
	r [5]rune
}

// NewSpecials returns a set of special characters specified with a string of
// length 5. The special characters are the symbol prefix, the left quote, the
// right quote, the name-value separator, and the escape character, in that
// order. If the string is empty, default special characters are used: $, [, ], =,
// and \.
func NewSpecials(s string) *Specials {
	if len(s) == 0 {
		return &Specials{r: [5]rune{'$', '[', ']', '=', '\\'}}
	}
	r := []rune(s)
	if len(r) != 5 ||
		r[0] == r[1] || r[0] == r[2] || r[0] == r[3] || r[0] == r[4] ||
		r[1] == r[2] || r[1] == r[3] || r[1] == r[4] ||
		r[2] == r[3] || r[2] == r[4] ||
		r[3] == r[4] {
		panic(errors.New("expected 5 distinct special characters and not: " + s))
	}
	var r1 [5]rune
	copy(r1[:], r)
	return &Specials{r: r1}
}

func (s *Specials) check() {
	if s.r[0] == 0 {
		s.r = [5]rune{'$', '[', ']', '=', '\\'}
	}
}

func (s *Specials) String() string {
	s.check()
	return string(s.r[:])
}

// SymbolPrefix returns the symbol prefix.
func (s *Specials) SymbolPrefix() rune {
	s.check()
	return s.r[0]
}

// LeftQuote returns the left quote.
func (s *Specials) LeftQuote() rune {
	s.check()
	return s.r[1]
}

// RightQuote returns the right quote.
func (s *Specials) RightQuote() rune {
	s.check()
	return s.r[2]
}

// Separator returns the name-value separator.
func (s *Specials) Separator() rune {
	s.check()
	return s.r[3]
}

// Escape returns the escape character.
func (s *Specials) Escape() rune {
	s.check()
	return s.r[4]
}
