package args

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"
)

// scannerState keeps track of work in progress
type scannerState struct {
	gigo       bool // garbage in garbage out (accept invalid Unicode)
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

func newScannerState(input []byte, prefix rune) scannerState {
	return scannerState{
		marker:    prefix,
		markerLen: len(string(prefix)),
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

// substitute replaces all symbol references in s with  their values from
// symbols. It returns a symval, a flag to indicate if anything was modified,
// and an error. The symval pointer is nil iff the error is non-nil. Any
// unresolved symbol references are copied untouched to the output. The syntax
// is explained in details in the package documentation.
func substitute(s string, symbols *symtab) (*symval, bool, error) {
	input := []byte(s)
	state := newScannerState(input, symbols.prefix)
	var resolved bytes.Buffer

	modified := false
	complete := true
	end := false
	for !end {
		status, err := state.scan()
		if err != nil {
			return nil, false, err
		}
		switch status {
		case 0:
		case 1: // $$symbol
			// append prefix (length can be 0) (and no need to test err which is always nil)
			resolved.Write(input[state.beforePos : state.symbolPos-2*state.markerLen])
			// append resolved symbol or restore original
			sym := input[state.symbolPos:nextPos(state.input)]
			symv, err := symbols.get(string(sym))

			if err != nil {
				return nil, false, err
			}

			if symv != nil {
				complete = complete && symv.resolved
				modified = true
				resolved.WriteString(symv.s)
			} else {
				// restore
				complete = false
				resolved.WriteRune(state.marker)
				resolved.WriteRune(state.marker)
				resolved.Write(sym)
			}
			state.beforePos = nextPos(state.input)
			state.symbolPos = -1
		case 2: // $$$symbol$
			// append prefix (length can be 0)
			resolved.Write(input[state.beforePos : state.symbolPos-3*state.markerLen])
			// append resolved symbol or restore original
			sym := input[state.symbolPos : nextPos(state.input)-1*state.markerLen]
			symv, err := symbols.get(string(sym))

			if err != nil {
				return nil, false, err
			}

			if symv != nil {
				modified = true
				complete = complete && symv.resolved
				resolved.WriteString(symv.s)
			} else {
				// restore
				complete = false
				resolved.WriteRune(state.marker)
				resolved.WriteRune(state.marker)
				resolved.WriteRune(state.marker)
				resolved.Write(sym)
				resolved.WriteRune(state.marker)
			}

			// discard closing marker
			state.input.ReadRune()
			state.beforePos = nextPos(state.input) - 1*state.markerLen
			state.symbolPos = -1
		case -1:
			if state.beforePos < len(input) {
				resolved.Write(input[state.beforePos:])
			}
			end = true
		default:
			// (no test coverage)
			panic("bug found: unexpected value returned by scan()")
		}
	}
	return &symval{s: resolved.String(), resolved: complete}, modified, nil
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
