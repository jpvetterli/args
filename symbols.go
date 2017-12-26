package args

import (
	"fmt"
)

type resolver interface {
	get(string) (*symval, error)
}

type cycleError struct {
	s string
}

func (e cycleError) Error() string {
	return fmt.Sprintf(`cyclical symbol definition detected: "%s"`, e.s)

}

// symval encapsulates a symbol table value.
// Its zero value is the initial state.
type symval struct {
	resolved bool
	s        string
}

// symtab is a lazy symbol table. Values are resolved when needed, and resolving
// a value can trigger the resolution of another one. All symbols use the same
// prefix, available in config.
type symtab struct {
	config *Config
	table  map[string]*symval
	cycle  map[string]bool
}

// newSymtab returns a new symbol table using the config.
func newSymtab(config *Config) symtab {
	return symtab{
		config: config,
		table:  make(map[string]*symval),
		cycle:  make(map[string]bool),
	}
}

// put adds an entry to the symbol table and returns true if s agrees with
// the syntax of a symbol definition. If the entry is already present it is
// left untouched. This behavior is known as "first wins". The method returns
// false if symbol does not agree with the syntax. The syntax is described in
// detail in the package documentation.
func (t *symtab) put(s, value string) bool {
	r := []rune(s)
	// symbol if 2 or more characters starting with prefix but not prefix+prefix
	prefix := t.config.GetSpecial(SpecSymbolPrefix)
	if len(r) > 1 && r[0] == prefix && r[1] != prefix {
		sym := string(r[1:])
		if _, ok := t.table[sym]; !ok {
			// initially not resolved
			t.table[sym] = &symval{s: value}
		}
		return true
	}
	return false
}

// get returns the address of the symval for a symbol in the symbol table. It
// returns nil and no error when the symbol is not in the table.  It resolves
// the symbol when not done yet. It returns nil and an error when a cyclical
// dependency is detected. The method updates the symbol table.
func (t *symtab) get(symbol string) (value *symval, err error) {
	if _, ok := t.cycle[symbol]; ok {
		return nil, cycleError{s: symbol}
	}
	t.cycle[symbol] = true
	defer func() {
		delete(t.cycle, symbol)
	}()
	sv, ok := t.table[symbol]
	if !ok {
		return nil, nil
	}
	if sv.resolved {
		return sv, nil
	}

	// not resolved, scan recursively the *quoted* value
	tkz := newTokenizer(t.config, t)
	quoted := string(t.config.GetSpecial(SpecOpenQuote)) +
		sv.s + string(t.config.GetSpecial(SpecCloseQuote))
	tkz.Reset([]byte(quoted))
	token, sv1, err := tkz.Next()

	if err != nil {
		return nil, err
	}
	if token != tokenString {
		return nil, fmt.Errorf(`recursive scan failed: %s`, quoted)
	}
	sv.resolved = sv1.resolved
	return sv1, nil
}
