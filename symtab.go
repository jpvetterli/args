package args

import (
	"fmt"
)

// symval encapsulates a symbol table value.
// Its zero value is the initial state.
type symval struct {
	resolved bool
	s        string
}

// symtab is a lazy symbol table. Values are resolved when needed, and resolving
// a value can trigger the resolution of another one. All  symbols use the same
// prefix.
type symtab struct {
	table  map[string]*symval
	prefix rune
	cycle  map[string]bool
}

// newSymtab returns a new symbol table with a substituter using the given
// symbol prefix.
func newSymtab(prefix rune) symtab {
	return symtab{
		table:  make(map[string]*symval),
		prefix: prefix,
		cycle:  make(map[string]bool),
	}
}

// put adds an entry to the symbol table and returns true if s agrees with
// the syntax of a symbol definition.  If the entry is already present it is
// left untouched. This behavior is known as "first wins". The method returns
// false if symbol does not agree with the syntax. The syntax is described in
// detail in the package documentation.
func (t *symtab) put(s, value string) bool {
	r := []rune(s)
	// symbol if 2 or more characters starting with prefix but not prefix+prefix
	if len(r) > 1 && r[0] == t.prefix && r[1] != t.prefix {
		sym := string(r[1:])
		if _, ok := t.table[sym]; !ok {
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
		return nil, fmt.Errorf(`cyclical symbol definition detected: "%s"`, symbol)
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

	// next stmt can result in recursive call
	symv, modified, err := substitute(sv.s, t)
	if err != nil {
		return nil, err
	}
	if modified {
		sv.s = symv.s
	}
	sv.resolved = symv.resolved
	return sv, nil
}
