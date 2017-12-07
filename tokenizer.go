package args

import (
	"bytes"
	"fmt"
	"unicode"
	"unicode/utf8"
)

// tokenizer methods help to parse an input as a series of name-value pairs.
type tokenizer struct {
	config    *Specials
	input     []byte
	reader    *bytes.Reader
	state     tokenizerScanState
	escState  tokenizerScanState
	stringBuf bytes.Buffer
	depth     int // nested brackets
}

func newTokenizer(config *Specials) *tokenizer {
	return &tokenizer{config: config, reader: bytes.NewReader(nil)}
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

type tokenizerScanState uint8

const (
	tsInit tokenizerScanState = iota
	tsEnd
	tsString
	tsBracket
	tsEscape
	tsError
)

// Reset makes the tokenizer ready to process a new input.
func (t *tokenizer) Reset(input []byte) {
	t.input = input
	t.reader.Reset(input)
	t.state = tsInit
	t.escState = tsInit
	t.stringBuf.Reset()
	t.depth = 0
}

// Next finds the next token in the input. It returns a token, a slice and an
// error. If and only if the token is tokenError, error is not nil. If and only
// if the token is tokenString, the string is not empty.
func (t *tokenizer) Next() (token, string, error) {
	if t.state == tsError {
		panic("tokenizer.next() called after an error")
	}
	t.state = tsInit
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
	t.state = tsError
	return tokenError, nil, fmt.Errorf(`at "%s": %s`, t.ErrorContext(), msg)
}

func (t *tokenizer) invalidCharacterError(msg string) (token, []byte, error) {
	t.state = tsError
	return tokenError, nil, fmt.Errorf(`at "%s%c": %s`, t.ErrorContext(), utf8.RuneError, msg)
}

func (t *tokenizer) panic(r rune) {
	panic(fmt.Sprintf("bug in scan() character: %c state: %d", r, t.state))
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

	// "default return" at the end: "return ttNone, nil, nil"

	switch {

	case r == 0:
		switch t.state {
		case tsInit, tsEnd:
			return tokenEnd, nil, nil
		case tsString:
			return tokenString, t.stringBuf.Bytes(), nil
		case tsBracket, tsEscape:
			return t.genericError("premature end of input")
		default:
			t.panic(r)
		}

	case unicode.IsSpace(r):
		switch t.state {
		case tsInit:
		case tsString:
			return tokenString, t.stringBuf.Bytes(), nil
		case tsBracket:
			t.stringBuf.WriteRune(r)
		case tsEscape:
			t.state = t.escState
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	case r == t.config.Separator():
		switch t.state {
		case tsInit:
			return tokenEqual, nil, nil
		case tsString:
			t.reader.UnreadRune()
			return tokenString, t.stringBuf.Bytes(), nil
		case tsBracket:
			t.stringBuf.WriteRune(r)
		case tsEscape:
			t.state = t.escState
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	case r == t.config.Escape():
		switch t.state {
		case tsInit, tsString:
			t.escState = tsString
			t.state = tsEscape
		case tsBracket:
			t.escState = tsBracket
			t.state = tsEscape
		case tsEscape:
			t.state = t.escState // restore state
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	case r == t.config.LeftQuote():
		switch t.state {
		case tsInit, tsString:
			t.depth = 1
			t.state = tsBracket
		case tsBracket:
			t.depth++
			t.stringBuf.WriteRune(r)
		case tsEscape:
			t.state = t.escState
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	case r == t.config.RightQuote():
		switch t.state {
		case tsInit, tsString:
			return t.genericError("premature " + string(r))
		case tsBracket:
			t.depth--
			if t.depth == 0 {
				t.state = tsString
				// do not print outermost bracket
			} else {
				t.stringBuf.WriteRune(r)
			}
		case tsEscape:
			t.state = t.escState
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}

	default:
		switch t.state {
		case tsInit:
			t.state = tsString
			t.stringBuf.WriteRune(r)
		case tsString:
			t.stringBuf.WriteRune(r)
		case tsBracket:
			t.stringBuf.WriteRune(r)
		case tsEscape:
			t.state = t.escState
			t.stringBuf.WriteRune(t.config.Escape())
			t.stringBuf.WriteRune(r)
		default: // tsEnd
			t.panic(r)
		}
	}

	return tokenNone, nil, nil
}
