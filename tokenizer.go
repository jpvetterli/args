package args

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"
)

// tokenizer methods help to parse an input as a series of name-value pairs.
type tokenizer struct {
	config    *Config
	input     []byte
	reader    *bytes.Reader
	resolver  resolver
	resolved  bool
	stringBuf bytes.Buffer
	symBuf    bytes.Buffer
	stack     stack
}

func (t *tokenizer) symval() *symval {
	return &symval{
		resolved: t.resolved,
		s:        t.stringBuf.String(),
	}
}

func newTokenizer(configuration *Config, resolver resolver) *tokenizer {
	return &tokenizer{
		config:   configuration,
		reader:   bytes.NewReader(nil),
		resolver: resolver,
	}
}

var errorContextLength = 15

type token uint8

const (
	tokenNone token = iota
	tokenEnd
	tokenEqual
	tokenString
	tokenError
)

type scanState uint8

const (
	tsInit scanState = iota // means "stack empty"
	tsString
	tsBracket
	tsSymbol
	tsPrefix
	tsEscape
	tsEnd
	tsError // meant to be left on the stack
)

type stack []scanState

func (s *stack) top() scanState {
	l := len(*s)
	if l == 0 {
		return tsInit
	}
	return (*s)[l-1]
}

func (s *stack) pushIfEmpty(state scanState) {
	if len(*s) == 0 {
		s.push(state)
	}
}

func (s *stack) push(state scanState) {
	if state == tsInit {
		panic(fmt.Errorf("bug: push tsInit not allowed"))
	}
	*s = append(*s, state)
}

func (s *stack) pop() scanState {
	l := len(*s)
	if l == 0 {
		panic(fmt.Errorf("bug: stack empty"))
	}
	defer func() { *s = (*s)[:l-1] }()
	return (*s)[l-1]
}

// Reset makes the tokenizer ready to process a new input.
func (t *tokenizer) Reset(input []byte) {
	t.input = input
	t.reader.Reset(input)
	t.stringBuf.Reset()
	t.resolved = true
	t.symBuf.Reset()
	t.stack = t.stack[:0]
}

// Next finds the next token in the input. It returns a token, a *symval and an
// error. If and only if the token is tokenError, error is not nil. If and only
// if the token is tokenString, the symval is not nil
func (t *tokenizer) Next() (token, *symval, error) {
	if len(t.stack) != 0 {
		if t.stack.top() == tsError {
			panic(fmt.Errorf("Next() called after an error, context: %s", string(t.ErrorContext())))
		} else {
			panic(fmt.Errorf("Next() called on non-empty stack (size=%d stack=%v)", len(t.stack), t.stack))
		}
	}
	t.stringBuf.Reset()
	t.resolved = true
	for {
		tokType, tok, err := t.scan()
		if tokType != tokenNone {
			return tokType, tok, err
		}
	}
}

// ErrorContext return a piece of input preceding the position where an error
// was detected.
func (t *tokenizer) ErrorContext() []byte {
	n := nextPos(t.reader)
	if n > errorContextLength {
		return append([]byte("..."), t.input[n-errorContextLength:n]...)
	}
	return t.input[:n]
}

func (t *tokenizer) symbolCharacterError(c rune) (token, *symval, error) {
	t.stack.push(tsError)
	return tokenError, nil, fmt.Errorf(`at "%s": character invalid in symbol: '%c'`, t.ErrorContext(), c)
}

func (t *tokenizer) genericError(msg string) (token, *symval, error) {
	t.stack.push(tsError)
	return tokenError, nil, fmt.Errorf(`at "%s": %s`, t.ErrorContext(), msg)
}

func (t *tokenizer) invalidCharacterError(msg string) (token, *symval, error) {
	t.stack.push(tsError)
	return tokenError, nil, fmt.Errorf(`at "%s%c": %s`, t.ErrorContext(), utf8.RuneError, msg)
}

func (t *tokenizer) panic(r rune) {
	panic(fmt.Sprintf("bug in scan() character: %c state: %d", r, t.stack.top()))
}

func nextPos(r *bytes.Reader) int {
	return int(r.Size()) - r.Len()
}

func (t *tokenizer) scan() (token, *symval, error) {

	r, _, err := t.reader.ReadRune()
	// byte order mark (\ufeff) not supported --
	// it is user's responsibility to skip BOM if 1st chararcter of file
	if r == utf8.RuneError {
		t.reader.UnreadRune()
		return t.invalidCharacterError("invalid character")
	}
	if r == '\ufeff' {
		t.reader.UnreadRune()
		return t.invalidCharacterError("byte order mark character not supported")
	}
	if err != nil {
		// not an error but end of input
		r = 0
	}

	// notes:
	// 1. "default return" is at the end (return tokenNone, nil, nil)
	// 2. code assumes without testing with valid() that space, $, =, [, \ are
	//    invalid in symbols (i.e., not $, but SpecSymbolPrefix, etc.)

	switch {

	case r == 0:
		switch t.stack.top() {
		case tsInit:
			return tokenEnd, nil, nil
		case tsString:
			t.stack.pop()
			return tokenString, t.symval(), nil
		case tsBracket, tsSymbol, tsPrefix, tsEscape:
			return t.genericError("premature end of input")
		case tsEnd:
			t.stack.pop()
			return tokenEnd, nil, nil
		default:
			t.panic(r)
		}

	case unicode.IsSpace(r):
		switch t.stack.top() {
		case tsInit:
		case tsString:
			t.stack.pop()
			return tokenString, t.symval(), nil
		case tsBracket:
			t.stringBuf.WriteRune(r)
		case tsSymbol, tsPrefix:
			return t.symbolCharacterError(r)
		case tsEscape:
			t.stack.pop()
			t.stack.pushIfEmpty(tsString)
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	case r == t.config.GetSpecial(SpecSeparator):
		switch t.stack.top() {
		case tsInit:
			return tokenEqual, nil, nil
		case tsString:
			t.reader.UnreadRune()
			t.stack.pop()
			return tokenString, t.symval(), nil
		case tsBracket:
			t.stringBuf.WriteRune(r)
		case tsSymbol, tsPrefix:
			return t.symbolCharacterError(r)
		case tsEscape:
			t.stack.pop()
			t.stack.pushIfEmpty(tsString)
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	case r == t.config.GetSpecial(SpecEscape):
		switch t.stack.top() {
		case tsInit, tsString, tsBracket:
			t.stack.push(tsEscape)
		case tsSymbol, tsPrefix:
			return t.symbolCharacterError(r)
		case tsEscape:
			t.stack.pop()
			t.stack.pushIfEmpty(tsString)
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	case r == t.config.GetSpecial(SpecOpenQuote):
		switch t.stack.top() {
		case tsInit, tsString:
			t.stack.push(tsBracket)
		case tsBracket:
			t.stack.push(tsBracket)
			t.stringBuf.WriteRune(r)
		case tsSymbol:
			return t.symbolCharacterError(r)
		case tsPrefix:
			t.stack.pop()
			t.stack.push(tsSymbol)
		case tsEscape:
			t.stack.pop()
			t.stack.pushIfEmpty(tsString)
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	case r == t.config.GetSpecial(SpecCloseQuote):
		switch t.stack.top() {
		case tsInit, tsString:
			return t.genericError(fmt.Sprintf("premature '%c'", r))
		case tsBracket:
			t.stack.pop()
			if t.stack.top() == tsBracket {
				// do not print outermost bracket
				t.stringBuf.WriteRune(r)
			} else {
				t.stack.pushIfEmpty(tsString)
			}
		case tsSymbol:
			t.stack.pop()
			t.stack.pushIfEmpty(tsString)
			symbol := t.symBuf.String()
			t.symBuf.Reset()
			symval, err := t.resolver.get(symbol)
			if symval == nil || !symval.resolved {
				t.resolved = false // toggle!
			}
			if err != nil {
				t.stack.push(tsError)
				if _, ok := err.(cycleError); ok {
					return tokenError, nil, err
				}
				return t.genericError(fmt.Sprintf(`error resolving "%s": %v`, symbol, err))
			}
			if symval != nil {
				t.stringBuf.WriteString(symval.s)
			} else {
				t.stringBuf.WriteRune(t.config.GetSpecial(SpecSymbolPrefix))
				t.stringBuf.WriteRune(t.config.GetSpecial(SpecOpenQuote))
				t.stringBuf.WriteString(symbol)
				t.stringBuf.WriteRune(t.config.GetSpecial(SpecCloseQuote))
			}
		case tsPrefix:
			return t.symbolCharacterError(r)
		case tsEscape:
			t.stack.pop()
			t.stack.pushIfEmpty(tsString)
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	case r == t.config.GetSpecial(SpecSymbolPrefix):
		switch t.stack.top() {
		case tsInit, tsString:
			t.stack.push(tsPrefix)
		case tsBracket:
			t.stack.push(tsPrefix)
		case tsSymbol, tsPrefix:
			return t.symbolCharacterError(r)
		case tsEscape:
			t.stack.pop()
			t.stack.pushIfEmpty(tsString)
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	default:
		switch t.stack.top() {
		case tsInit:
			t.stack.push(tsString)
			t.stringBuf.WriteRune(r)
		case tsString:
			t.stringBuf.WriteRune(r)
		case tsBracket:
			t.stringBuf.WriteRune(r)
		case tsSymbol:
			if valid(r) {
				t.symBuf.WriteRune(r)
			} else {
				return t.symbolCharacterError(r)
			}
		case tsPrefix:
			// NOTE: symbol definition
			t.stack.pop()
			t.stack.pushIfEmpty(tsString)
			t.stringBuf.WriteRune(t.config.GetSpecial(SpecSymbolPrefix))
			t.stringBuf.WriteRune(r)
		case tsEscape:
			t.stack.pop()
			t.stack.pushIfEmpty(tsString)
			t.stringBuf.WriteRune(t.config.GetSpecial(SpecEscape))
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}
	}

	return tokenNone, nil, nil
}
