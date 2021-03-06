package args

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

type testResolver struct {
}

// NOTE: cannot test unresolved values at this level
func (tr *testResolver) get(symbol string) (*symval, error) {
	if symbol == "ERROR" {
		return nil, fmt.Errorf("(simulated error)")
	}
	if strings.HasPrefix(symbol, "_") {
		return &symval{resolved: true, s: "value of " + symbol}, nil
	}
	return nil, nil
}

var symResolver = &testResolver{}

var tokTestData = []struct {
	input  string
	expect []interface{}
}{
	{`[\ ]`, []interface{}{" "}},
	{`[\=]`, []interface{}{"="}},
	{`[\\]`, []interface{}{`\`}},
	{`\[`, []interface{}{`[`}},
	{`\]`, []interface{}{`]`}},
	{`\$`, []interface{}{`$`}},

	{"", []interface{}{tokenEnd}},
	{`\x`, []interface{}{`\x`}},
	{`\\`, []interface{}{"\\"}},
	{`\`, []interface{}{errors.New(`at "\": premature end of input`)}},
	{"foo", []interface{}{"foo"}},
	{`[foo]`, []interface{}{"foo"}},
	{`[[foo]]`, []interface{}{"[foo]"}},
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
	{"foo = bar]quux", []interface{}{"foo", tokenEqual, errors.New(`at "foo = bar]": premature ']'`)}},
	{"foo = bar[quux", []interface{}{"foo", tokenEqual, errors.New(`at "foo = bar[quux": premature end of input`)}},
	{"foo = bar[qu[ux", []interface{}{"foo", tokenEqual, errors.New(`at "foo = bar[qu[ux": premature end of input`)}},
	{"foo = bar[qu[ux]", []interface{}{"foo", tokenEqual, errors.New(`at "...oo = bar[qu[ux]": premature end of input`)}},
	{"]", []interface{}{errors.New(`at "]": premature ']'`)}},
	{"[", []interface{}{errors.New(`at "[": premature end of input`)}},
	{"\ufefffoo", []interface{}{errors.New("at \"\ufffd\": byte order mark character not supported")}},
	{"fo\ufeffo", []interface{}{errors.New("at \"fo\ufffd\": byte order mark character not supported")}}, // \ufffd is �
	{"fo\ufffdo", []interface{}{errors.New("at \"fo\ufffd\": invalid character")}},
	{`foo \= \[x: \$$X\\`, []interface{}{"foo", "=", "[x:", `$$X\`}},
	{`foo \= \[x: \$$X\]`, []interface{}{"foo", "=", "[x:", `$$X]`}},

	{`$[]`, []interface{}{"$[]"}},
	{`$[x]`, []interface{}{"$[x]"}},
	{`$[symbol]`, []interface{}{"$[symbol]"}},
	{`foo = $[symbol]`, []interface{}{"foo", tokenEqual, "$[symbol]"}},
	{`foo = [$[symbol]]`, []interface{}{"foo", tokenEqual, "$[symbol]"}},
	{`foo = [[$[symbol]]]`, []interface{}{"foo", tokenEqual, "[$[symbol]]"}},
	{`foo = [[$[symbol]]]`, []interface{}{"foo", tokenEqual, "[$[symbol]]"}},
	{`foo = abc$[symbol]cba etc.`, []interface{}{"foo", tokenEqual, "abc$[symbol]cba", "etc."}},
	{`foo = \ $[symbol]\]`, []interface{}{"foo", tokenEqual, " $[symbol]]"}},
	{`foo = \$[symbol]`, []interface{}{"foo", tokenEqual, "$symbol"}},
	{`foo = $[s mbol]`, []interface{}{"foo", tokenEqual, errors.New(`at "foo = $[s ": character invalid in symbol: ' '`)}},
	{`foo = $[s[mbol]`, []interface{}{"foo", tokenEqual, errors.New(`at "foo = $[s[": character invalid in symbol: '['`)}},
	{`foo = $[s=mbol]`, []interface{}{"foo", tokenEqual, errors.New(`at "foo = $[s=": character invalid in symbol: '='`)}},
	{`foo = $[s\mbol]`, []interface{}{"foo", tokenEqual, errors.New(`at "foo = $[s\": character invalid in symbol: '\'`)}},
	{`foo = $[s$mbol]`, []interface{}{"foo", tokenEqual, errors.New(`at "foo = $[s$": character invalid in symbol: '$'`)}},
	{`$foo = bar`, []interface{}{"$foo", tokenEqual, "bar"}},
	{`$foo = $[bar] etc.`, []interface{}{"$foo", tokenEqual, "$[bar]", "etc."}},
	{`$foo = $[s\mbol]`, []interface{}{"$foo", tokenEqual, errors.New(`at "$foo = $[s\": character invalid in symbol: '\'`)}},
	{`$foo = $[s+mbol]`, []interface{}{"$foo", tokenEqual, errors.New(`at "$foo = $[s+": character invalid in symbol: '+'`)}},

	{`$[_]`, []interface{}{"value of _"}},
	{`$[_x]`, []interface{}{"value of _x"}},
	{`$[_symbol]`, []interface{}{"value of _symbol"}},
	{`foo = $[_symbol]`, []interface{}{"foo", tokenEqual, "value of _symbol"}},
	{`foo = [$[_symbol]]`, []interface{}{"foo", tokenEqual, "value of _symbol"}},
	{`foo = [[$[_symbol]]]`, []interface{}{"foo", tokenEqual, "[value of _symbol]"}},
	{`foo = [[$[_symbol]]]`, []interface{}{"foo", tokenEqual, "[value of _symbol]"}},
	{`foo = abc$[_symbol]cba etc.`, []interface{}{"foo", tokenEqual, "abcvalue of _symbolcba", "etc."}},
	{`foo = \ $[_symbol]\]`, []interface{}{"foo", tokenEqual, " value of _symbol]"}},
	{`$foo = $[_bar] etc.`, []interface{}{"$foo", tokenEqual, "value of _bar", "etc."}},

	{`$[ERROR]`, []interface{}{errors.New(`at "$[ERROR]": error resolving "ERROR": (simulated error)`)}},

	{`\$a`, []interface{}{`$a`}},
	{`\$\$`, []interface{}{`$$`}},
	{`\$\.`, []interface{}{`$\.`}},
	{`\$\a`, []interface{}{`$\a`}},
	{`\$\[`, []interface{}{`$[`}},
	{`\$\]`, []interface{}{`$]`}},
	{` x `, []interface{}{`x`}},
	{` \$ `, []interface{}{`$`}},
	{` \$=`, []interface{}{`$`, tokenEqual}},

	{`$$`, []interface{}{errors.New(`at "$$": character invalid in symbol: '$'`)}},
	{`$\.`, []interface{}{errors.New(`at "$\": character invalid in symbol: '\'`)}},
	{`$\a`, []interface{}{errors.New(`at "$\": character invalid in symbol: '\'`)}},
	{`$\[`, []interface{}{errors.New(`at "$\": character invalid in symbol: '\'`)}},
	{`$]`, []interface{}{errors.New(`at "$]": character invalid in symbol: ']'`)}},
	{` $ `, []interface{}{errors.New(`at " $ ": character invalid in symbol: ' '`)}},
	{` $=`, []interface{}{errors.New(`at " $=": character invalid in symbol: '='`)}},

	{`foo=\$]`, []interface{}{`foo`, tokenEqual, errors.New(`at "foo=\$]": premature ']'`)}},
	{`foo=$[bar\$]`, []interface{}{`foo`, tokenEqual, errors.New(`at "foo=$[bar\": character invalid in symbol: '\'`)}},
	{`foo=[bar\$]`, []interface{}{`foo`, tokenEqual, "bar$"}},
	{`foo=x[bar\$]y`, []interface{}{`foo`, tokenEqual, "xbar$y"}},
	{`foo=[ [bar\$] ]`, []interface{}{`foo`, tokenEqual, " [bar$] "}},

	{`\$[ a \]b] = x`, []interface{}{`$ a ]b`, tokenEqual, "x"}},
	{`foo= [b$ c]`, []interface{}{`foo`, tokenEqual, errors.New(`at "foo= [b$ ": character invalid in symbol: ' '`)}},
}

func TestTokenizerOnGenericData(t *testing.T) {
	tkz := newTokenizer(NewConfig(), symResolver)
	for _, data := range tokTestData {
		tkz.reset([]byte(data.input))
		for i, exp := range data.expect {
			switch exp.(type) {
			case string:
				tkz.expectString(data.input, i, exp.(string), t)
			case scanToken:
				tkz.expectToken(data.input, i, exp.(scanToken), t)
			case error:
				tkz.expectError(data.input, i, exp.(error).Error(), t)
			default:
				panic(exp)
			}
		}
	}
}

func TestTokenizer(t *testing.T) {
	tokenizer := newTokenizer(NewConfig(), symResolver)
	// no reset() so must get tokEnd
	tok, s, err := tokenizer.next()
	if err != nil {
		t.Errorf("error!")
	}
	if tok != tokenEnd || s != nil {
		t.Errorf("tok, s == %v, \"%s\" expect: %v, %s", tok, s.s, tokenEnd, "\"\"")
	}
}

func TestTokenizerCallAfterError(t *testing.T) {
	tkz := newTokenizer(NewConfig(), symResolver)
	tkz.reset([]byte("]foo"))
	tkz.expectError("]foo", 0, `at "]": premature ']'`, t)
	defer panicHandler("Next() called after an error, context: ]", t)
	tkz.expectString("]foo", 1, "foo", t)
}

func (tkz *tokenizer) expectError(input string, pos int, expectedMsg string, t *testing.T) {
	tok, _, err := tkz.next()
	if err == nil || tok != tokenError {
		t.Errorf("E \"%s\"[%d]: error is nil or wrong token: %v (expected: %v)", input, pos, tok, tokenNone)
		return
	}
	errorString := err.Error()
	if errorString != expectedMsg {
		t.Errorf("E \"%s\"[%d]: error message: %s, expected: %s", input, pos, errorString, expectedMsg)
	}
}

func (tkz *tokenizer) expectToken(input string, pos int, expectedToken scanToken, t *testing.T) {
	tok, s, err := tkz.next()
	if err != nil {
		t.Errorf("T \"%s\"[%d]: unexpected error: %s", input, pos, err.Error())
		return
	}
	if tok != expectedToken {
		t.Errorf("T \"%s\"[%d]: token: %v, expected: %v", input, pos, tok, expectedToken)
	}
	if s != nil {
		t.Errorf("T \"%s\"[%d]: unexpected token string: %s", input, pos, s.s)
	}
}

func (tkz *tokenizer) expectString(input string, pos int, expectedString string, t *testing.T) {
	tok, s, err := tkz.next()
	if err != nil {
		t.Errorf("S \"%s\"[%d]: unexpected error: %s", input, pos, err.Error())
		return
	}
	if tok != tokenString {
		t.Errorf("S \"%s\"[%d]: token: %v, expected: %v", input, pos, tok, tokenString)
	}
	if s == nil {
		t.Errorf("S \"%s\"[%d]: no token string, expected %s", input, pos, expectedString)
	}
	if s.s != expectedString {
		t.Errorf("S \"%s\"[%d]: token string: %s, expected %s", input, pos, s.s, expectedString)
	}
}

func TestStack1(t *testing.T) {
	var st stack
	st.push(tsString)
	st.push(tsBracket)
	st.push(tsBracket)
	if st.top() != tsBracket {
		t.Errorf("top should be tsBracket")
	}
	if st.pop() != tsBracket {
		t.Errorf("pop 1 should be tsBracket")
	}
	if st.pop() != tsBracket {
		t.Errorf("pop 2 should be tsBracket")
	}
	if st.pop() != tsString {
		t.Errorf("pop 3 should be tsString")
	}
	defer panicHandler("bug: stack empty", t)
	st.pop()
}

func TestStack(t *testing.T) {
	var st stack
	defer panicHandler("bug: push tsInit not allowed", t)
	st.push(tsInit)
}

func TestStack3(t *testing.T) {
	var st stack
	st.pushIfEmpty(tsString)
	if len(st) != 1 {
		t.Errorf("size not 1: %d", len(st))
	}
	st.push(tsBracket)
	st.push(tsBracket)
	if len(st) != 3 {
		t.Errorf("size not 3: %d", len(st))
	}
	st.pushIfEmpty(tsString)
	if len(st) != 3 {
		t.Errorf("size not 3: %d", len(st))
	}
}

// panicHandler triggers a testing error if panic message differs from expected
// (copied from args_test.go, not same package)
func panicHandler(expected string, t *testing.T) {
	err := recover()
	if err == nil {
		if len(expected) > 0 {
			t.Errorf(`(recovery) no error caught, expected: "%s"`, expected)
		}
	} else {
		if e, ok := err.(error); !ok {
			t.Errorf("(recovery) unexpected error: %v", err)
		} else {
			if e.Error() != expected {
				t.Errorf(`(recovery) unexpected error message: "%s" expected: "%s"`, err, expected)
			}
		}
	}
}
