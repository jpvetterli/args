package args

import (
	"errors"
	"testing"
)

var tokTestData = []struct {
	input  string
	expect []interface{}
}{
	{"foo", []interface{}{"foo"}},
	{" foo bar", []interface{}{"foo", "bar"}},
	{"foo [bar]", []interface{}{"foo", "bar"}},
	{"foo =   [nihongo <日本語>] ", []interface{}{"foo", tokenEqual, "nihongo <日本語>"}},
	{"foo = [bar [quux] ]", []interface{}{"foo", tokenEqual, "bar [quux] "}},
	{"foo= bar", []interface{}{"foo", tokenEqual, "bar"}},
	{`foo \= bar`, []interface{}{"foo", "=", "bar"}},
	{`foo \ bar`, []interface{}{"foo", " bar"}},
	{`foo \  bar`, []interface{}{"foo", " ", "bar"}},
	{`foo \=\[bar\]`, []interface{}{"foo", "=[bar]"}},
	{`foo \\=\[bar\]`, []interface{}{"foo", `\`, tokenEqual, "[bar]"}},
	{`foo = [bar\]]`, []interface{}{"foo", tokenEqual, "bar]"}},
	{`foo = [ba\r]`, []interface{}{"foo", tokenEqual, `ba\r`}},
	{"foo = [bar = [quux] ]", []interface{}{"foo", tokenEqual, "bar = [quux] "}},
	{"foo = X[bar = [quux] ]Y", []interface{}{"foo", tokenEqual, "Xbar = [quux] Y"}},
	{"=foo == bar", []interface{}{tokenEqual, "foo", tokenEqual, tokenEqual, "bar"}},
	{"foo = bar]quux", []interface{}{"foo", tokenEqual, errors.New(`at "foo = bar]": premature ]`)}},
	{"foo = bar[quux", []interface{}{"foo", tokenEqual, errors.New(`at "foo = bar[quux": premature end of input`)}},
	{"foo = bar[qu[ux", []interface{}{"foo", tokenEqual, errors.New(`at "foo = bar[qu[ux": premature end of input`)}},
	{"foo = bar[qu[ux]", []interface{}{"foo", tokenEqual, errors.New(`at "...oo = bar[qu[ux]": premature end of input`)}},
	{"]", []interface{}{errors.New(`at "]": premature ]`)}},
	{"[", []interface{}{errors.New(`at "[": premature end of input`)}},
	{"\ufefffoo", []interface{}{errors.New("at \"\ufffd\": byte order mark character not supported")}},
	{"fo\ufeffo", []interface{}{errors.New("at \"fo\ufffd\": byte order mark character not supported")}}, // \ufffd is �
	{"fo\ufffdo", []interface{}{errors.New("at \"fo\ufffd\": invalid character")}},
	{`foo \= \[x: \$$X\\`, []interface{}{"foo", "=", "[x:", `\$$X\`}},
	{`foo \= \[x: \$$X\]`, []interface{}{"foo", "=", "[x:", `\$$X]`}},
}

func TestTokenizerOnGenericData(t *testing.T) {
	tkz := newTokenizer(NewSpecials(""))
	for _, data := range tokTestData {
		tkz.Reset([]byte(data.input))
		for i, exp := range data.expect {
			switch exp.(type) {
			case string:
				tkz.expectString(data.input, i, exp.(string), t)
			case token:
				tkz.expectToken(data.input, i, exp.(token), t)
			case error:
				tkz.expectError(data.input, i, exp.(error).Error(), t)
			default:
				panic(exp)
			}
		}
	}
}

func TestTokenizer(t *testing.T) {
	tokenizer := newTokenizer(NewSpecials(""))
	// no reset() so must get tokEnd
	tok, s, err := tokenizer.Next()
	if err != nil {
		t.Errorf("error!")
	}
	if tok != tokenEnd || len(s) > 0 {
		t.Errorf("tok, s == %v, \"%s\" expect: %v, %s", tok, s, tokenEnd, "\"\"")
	}
}

func TestTokenizerCallAfterError(t *testing.T) {
	tkz := newTokenizer(NewSpecials(""))
	tkz.Reset([]byte("]foo"))
	tkz.expectError("]foo", 0, `at "]": premature ]`, t)
	defer func() {
		if s := recover(); s != nil {
			expected := "tokenizer.next() called after an error"
			if s != expected {
				t.Errorf(`recovery failure: error message: "%s" expected: "%s"`, s, expected)
			}
		}
	}()
	tkz.expectString("]foo", 1, "foo", t)
}

func (tkz *tokenizer) expectError(input string, pos int, expectedMsg string, t *testing.T) {
	tok, _, err := tkz.Next()
	if err == nil || tok != tokenError {
		t.Errorf("E \"%s\"[%d]: error is nil or wrong token: %v (expected: %v)", input, pos, tok, tokenNone)
		return
	}
	errorString := err.Error()
	if errorString != expectedMsg {
		t.Errorf("E \"%s\"[%d]: error message: %s, expected: %s", input, pos, errorString, expectedMsg)
	}
}

func (tkz *tokenizer) expectToken(input string, pos int, expectedToken token, t *testing.T) {
	tok, s, err := tkz.Next()
	if err != nil {
		t.Errorf("T \"%s\"[%d]: unexpected error: %s", input, pos, err.Error())
		return
	}
	if tok != expectedToken {
		t.Errorf("T \"%s\"[%d]: token: %v, expected: %v", input, pos, tok, expectedToken)
	}
	if len(s) > 0 {
		t.Errorf("T \"%s\"[%d]: unexpected token string: %s", input, pos, s)
	}
}

func (tkz *tokenizer) expectString(input string, pos int, expectedString string, t *testing.T) {
	tok, s, err := tkz.Next()
	if err != nil {
		t.Errorf("S \"%s\"[%d]: unexpected error: %s", input, pos, err.Error())
		return
	}
	if tok != tokenString {
		t.Errorf("S \"%s\"[%d]: token: %v, expected: %v", input, pos, tok, tokenString)
	}
	if s != expectedString {
		t.Errorf("S \"%s\"[%d]: token string: %s, expected %s", input, pos, s, expectedString)
	}
}
