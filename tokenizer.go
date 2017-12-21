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
	stringBuf bytes.Buffer
	stack     stack
}

func newTokenizer(configuration *Config) *tokenizer {
	return &tokenizer{config: configuration, reader: bytes.NewReader(nil)}
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
	tsEnd
	tsString // push tsString does nothing if top already tsString
	tsBracket
	tsEscape
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

func (s *stack) push(state scanState) {
	if state == tsInit {
		panic(fmt.Errorf("bug: push tsInit not allowed"))
	}
	if state == tsString && s.top() == tsString {

	} else {
		*s = append(*s, state)
	}
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
	t.stack = t.stack[:0]
}

func (t *tokenizer) ORIGReset(input []byte) {
	// t.input = input
	// t.reader.Reset(input)
	// t.state = tsInit
	// t.escState = tsInit
	// t.stringBuf.Reset()
	// t.depth = 0
}

// Next finds the next token in the input. It returns a token, a slice and an
// error. If and only if the token is tokenError, error is not nil. If and only
// if the token is tokenString, the string is not empty.
func (t *tokenizer) Next() (token, string, error) {
	if len(t.stack) != 0 {
		if t.stack.top() == tsError {
			panic(fmt.Errorf("Next() called after an error"))
		} else {
			panic(fmt.Errorf("Next() called on non-empty stack (size=%d top=%v)", len(t.stack), t.stack.top()))
		}
	}
	t.stringBuf.Reset()
	for {
		tokType, tok, err := t.scan()
		if tokType != tokenNone {
			return tokType, string(tok), err
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

func (t *tokenizer) genericError(msg string) (token, []byte, error) {
	// t.state = tsError
	t.stack.push(tsError)
	return tokenError, nil, fmt.Errorf(`at "%s": %s`, t.ErrorContext(), msg)
}

func (t *tokenizer) invalidCharacterError(msg string) (token, []byte, error) {
	// t.state = tsError
	t.stack.push(tsError)
	return tokenError, nil, fmt.Errorf(`at "%s%c": %s`, t.ErrorContext(), utf8.RuneError, msg)
}

func (t *tokenizer) panic(r rune) {
	// panic(fmt.Sprintf("bug in scan() character: %c state: %d", r, t.state))
	panic(fmt.Sprintf("bug in scan() character: %c state: %d", r, t.stack.top()))
}

func (t *tokenizer) scan() (token, []byte, error) {

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

	// "default return" at the end: "return tokenNone, nil, nil"

	switch {

	case r == 0:
		switch t.stack.top() {
		case tsInit:
			return tokenEnd, nil, nil
		case tsEnd:
			t.stack.pop()
			return tokenEnd, nil, nil
		case tsString:
			t.stack.pop()
			return tokenString, t.stringBuf.Bytes(), nil
		case tsBracket, tsEscape:
			return t.genericError("premature end of input")
		default:
			t.panic(r)
		}

	case unicode.IsSpace(r):
		switch t.stack.top() {
		case tsInit:
		case tsString:
			t.stack.pop()
			return tokenString, t.stringBuf.Bytes(), nil
		case tsBracket:
			t.stringBuf.WriteRune(r)
		case tsEscape:
			t.stack.pop()
			t.stack.push(tsString)
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
			return tokenString, t.stringBuf.Bytes(), nil
		case tsBracket:
			t.stringBuf.WriteRune(r)
		case tsEscape:
			t.stack.pop()
			t.stack.push(tsString)
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	case r == t.config.GetSpecial(SpecEscape):
		switch t.stack.top() {
		case tsInit, tsString:
			t.stack.push(tsEscape)
		case tsBracket:
			t.stack.push(tsEscape)
		case tsEscape:
			t.stack.pop()
			t.stack.push(tsString)
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
		case tsEscape:
			t.stack.pop()
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	case r == t.config.GetSpecial(SpecCloseQuote):
		switch t.stack.top() {
		case tsInit, tsString:
			return t.genericError("premature " + string(r))
		case tsBracket:
			t.stack.pop()
			if t.stack.top() == tsBracket {
				// do not print outermost bracket
				t.stringBuf.WriteRune(r)
			} else {
				t.stack.push(tsString)
			}
		case tsEscape:
			t.stack.pop()
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
		case tsEscape:
			t.stack.pop()
			t.stringBuf.WriteRune(t.config.GetSpecial(SpecEscape))
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}
	}

	return tokenNone, nil, nil
}
