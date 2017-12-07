package args

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"
)

// newSubstituter returns a substituter using the given symbol prefix.
func newSubstituter(symPrefix rune) *substituter {
	return &substituter{marker: symPrefix}
}

// newLooseSubstituter returns a substituter using the given symbol prefix and
// running in loose mode. In loose mode invalid characters are passed through
// untouched.
func newLooseSubstituter(symPrefix rune) *substituter {
	return &substituter{marker: symPrefix, loose: true}
}

// substituter provides a method for substituting variables. Variables are
// marked by a special prefix character. Refer to the package documentation for
// details and examples.
type substituter struct {
	marker rune // symbol marker, typically $
	loose  bool // ignore character validity
}

// scannerState keeps track of work in progress
type scannerState struct {
	gigo       bool // garbage in garbage out
	marker     rune
	markerLen  int
	input      *bytes.Reader
	state      stateMachineState
	beforePos, // position of string before symbol, ending in $$ or $$$
	symbolPos int // position of symbol ($$symbol or $$$symbol$)
}

func nextPos(r *bytes.Reader) int {
	return int(r.Size()) - r.Len()
}

// scannerState constructor
func newScannerState(input []byte, s *substituter) scannerState {
	return scannerState{
		gigo:      s.loose,
		marker:    s.marker,
		markerLen: len(string(s.marker)),
		input:     bytes.NewReader(input),
	}
}

// stateMachineState keeps track of the low-level scanner state
type stateMachineState uint8

// state machine constants
const (
	smsInit stateMachineState = iota
	smsDollar1
	smsDollar2
	smsDollar3
	smsSymbol
	smsDollarSymbol
	smsEnd
)

// Substitute returns the input with all occurences of symbols found in the
// symbols map replaced with their values. Symbols not found in the map are
// ignored. The syntax is documented in the NewSubstituter function. When the
// method returns a non-nil error, the output reflects what was done up to the
// point of error. The input is never modified.
func (subst *substituter) Substitute(input []byte, symbols *map[string]string) ([]byte, error) {
	s := newScannerState(input, subst)
	var resolved bytes.Buffer
	end := false
	for !end {
		status, err := s.scan()
		if err != nil {
			if s.beforePos < nextPos(s.input) {
				resolved.Write(input[s.beforePos:nextPos(s.input)])
			}
			return resolved.Bytes(), err
		}
		switch status {
		case 0:
		case 1: // $$symbol
			// append prefix (length can be 0) (and no need to test err which is always nil)
			resolved.Write(input[s.beforePos : s.symbolPos-2*s.markerLen])
			// append resolved symbol or restore original
			sym := input[s.symbolPos:nextPos(s.input)]
			if rsym, ok := (*symbols)[string(sym)]; ok {
				resolved.WriteString(rsym)
			} else {
				// restore
				resolved.WriteRune(s.marker)
				resolved.WriteRune(s.marker)
				resolved.Write(sym)
			}
			s.beforePos = nextPos(s.input)
			s.symbolPos = -1
		case 2: // $$$symbol$
			// append prefix (length can be 0)
			resolved.Write(input[s.beforePos : s.symbolPos-3*s.markerLen])
			// append resolved symbol or restore original
			sym := input[s.symbolPos : nextPos(s.input)-1*s.markerLen]
			if rsym, ok := (*symbols)[string(sym)]; ok {
				resolved.WriteString(rsym)
			} else {
				// restore
				resolved.WriteRune(s.marker)
				resolved.WriteRune(s.marker)
				resolved.WriteRune(s.marker)
				resolved.Write(sym)
				resolved.WriteRune(s.marker)
			}

			// discard closing marker
			s.input.ReadRune()
			s.beforePos = nextPos(s.input) - 1*s.markerLen
			s.symbolPos = -1
		case -1:
			if s.beforePos < len(input) {
				resolved.Write(input[s.beforePos:])
			}
			end = true
		default:
			// (no test coverage)
			panic("bug found: unexpected value returned by scan()")
		}
	}
	return resolved.Bytes(), nil
}

// scan the input for symbols. Return 0 to continue, 1 when when
// a $$symbol found, 2 when a $$$symbol$ found, -1 at end of input.
// An non-nil error is returned when an invalid rune is read.
func (s *scannerState) scan() (int, error) {
	r, w, err := s.input.ReadRune()
	if !s.gigo {
		if r == 0xFEFF {
			// the commented out if would skip BOM if very first character of input
			// 	if nextPos(s.input) == w {
			// 		s.beforePos = w
			// 		r, w, err = s.input.ReadRune()
			// 	} else {
			r = utf8.RuneError
			// 	}
		}
		if r == utf8.RuneError {
			s.input.UnreadRune()
			return 0, fmt.Errorf("Invalid character at offset %d", nextPos(s.input))
		}
	}
	if err != nil {
		// not an error but end of input
		r, w = 0, 0
	}

	switch s.state {

	case smsEnd:

	case smsInit:
		switch r {
		case s.marker:
			s.state = smsDollar1
		case 0:
			s.state = smsEnd
		}

	case smsDollar1:
		switch r {
		case s.marker:
			s.state = smsDollar2
		case 0:
			s.state = smsEnd
		default:
			s.state = smsInit
		}

	case smsDollar2:
		if s.valid(r) {
			// possible $$symbol
			s.symbolPos = nextPos(s.input) - w
			s.state = smsSymbol
		} else {
			switch r {
			case s.marker:
				s.state = smsDollar3
			case 0:
				s.state = smsEnd
			default:
				s.state = smsInit
			}
		}

	case smsDollar3:
		if s.valid(r) {
			// possible $$$symbol$
			s.symbolPos = nextPos(s.input) - w
			s.state = smsDollarSymbol
		} else {
			switch r {
			case s.marker:
				// same state
			case 0:
				s.state = smsEnd
			default:
				s.state = smsInit
			}
		}

	case smsSymbol:
		if !s.valid(r) {
			s.input.UnreadRune()
			if r == 0 {
				s.state = smsEnd
			} else {
				s.state = smsInit
			}
			return 1, nil
		}

	case smsDollarSymbol:
		if r == s.marker {
			// no backtracking
			s.state = smsInit
			return 2, nil
		} else if !s.valid(r) {
			// it's a $$symbol with a $ in front
			s.input.UnreadRune()
			s.state = smsInit
			return 1, nil
		}
	}
	if r == 0 {
		return -1, nil
	}
	return 0, nil
}

// valid returns true if r is valid symbol character.
func (s *scannerState) valid(r rune) bool {
	return r != 0 && r != s.marker && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_')
}
