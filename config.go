package args

import "fmt"

type specConstant uint8

// Special character constants for Config methods.
const (
	SpecSymbolPrefix specConstant = iota
	SpecOpenQuote
	SpecCloseQuote
	SpecSeparator
	SpecEscape
)

type opConstant uint8

// Operator constants for Config methods.
const (
	OpCond opConstant = iota
	OpDump
	OpImport
	OpInclude
	OpMacro
	OpReset
	OpSkip
)

// Config holds configurable special characters and operator names.
type Config struct {
	specList [5]rune
	opDict   map[string]opConstant
}

var specialDescription = [5]string{
	"symbol prefix",
	"open quote",
	"close quote",
	"separator",
	"escape",
}

// NewConfig returns the address of a new default Config.
func NewConfig() *Config {
	return &Config{
		specList: [5]rune{'$', '[', ']', '=', '\\'},
		opDict: map[string]opConstant{
			"macro":   OpMacro,
			"cond":    OpCond,
			"dump":    OpDump,
			"import":  OpImport,
			"include": OpInclude,
			"reset":   OpReset,
			"--":      OpSkip,
		},
	}
}

func (c *Config) copy() *Config {
	var sc [5]rune
	for i, r := range c.specList {
		sc[i] = r
	}
	oc := make(map[string]opConstant, len(c.opDict))
	for n, v := range c.opDict {
		oc[n] = v
	}
	return &Config{specList: sc, opDict: oc}
}

// GetSpecial returns the character currently corresponding to a special
// character identified by its constant.
func (c *Config) GetSpecial(which specConstant) rune {
	switch which {
	case SpecSymbolPrefix, SpecOpenQuote, SpecCloseQuote, SpecSeparator, SpecEscape:
		return c.specList[which]
	}
	panic(fmt.Errorf(`unknown special: %v`, which))
}

// SetSpecial changes a special character identified by a constant. Panics if
// ch is invalid, or is already used, or if spec is unknown.
func (c *Config) SetSpecial(spec specConstant, ch rune) {
	switch spec {
	case SpecSymbolPrefix:
	case SpecOpenQuote:
	case SpecCloseQuote:
	case SpecSeparator:
	case SpecEscape:
	default:
		panic(fmt.Errorf(`unknown special: %v`, spec))
	}
	if !validSpecial(ch) {
		panic(fmt.Errorf("cannot use '%c' as %s: not a valid special character", ch, specialDescription[spec]))
	}
	if c.isDuplicate(spec, ch) {
		panic(fmt.Errorf("cannot use '%c' as %s: already used", ch, specialDescription[spec]))
	}
	c.specList[spec] = ch

}

func (c *Config) isDuplicate(i specConstant, ch rune) bool {
	switch i {
	case 0:
		return ch == c.specList[1] || ch == c.specList[2] || ch == c.specList[3] || ch == c.specList[4]
	case 1:
		return ch == c.specList[0] || ch == c.specList[2] || ch == c.specList[3] || ch == c.specList[4]
	case 2:
		return ch == c.specList[0] || ch == c.specList[1] || ch == c.specList[3] || ch == c.specList[4]
	case 3:
		return ch == c.specList[0] || ch == c.specList[1] || ch == c.specList[2] || ch == c.specList[4]
	case 4:
		return ch == c.specList[0] || ch == c.specList[1] || ch == c.specList[2] || ch == c.specList[3]
	}
	panic(fmt.Errorf("bug found: %d", i))
}

// GetOpName returns the name of operator op. Panics if op is unknown.
func (c *Config) GetOpName(op opConstant) string {
	for n, o := range c.opDict {
		if o == op {
			return n
		}
	}
	panic(fmt.Errorf(`unknown operator: %v`, op))
}

// SetOpName changes the name of an operator identified by a constant. Panics if
// name is invalid, or is already used, or if op is unknown.
func (c *Config) SetOpName(op opConstant, name string) {
	if err := validate(name); err != nil {
		panic(err)
	}
	if _, ok := c.opDict[name]; ok {
		panic(fmt.Errorf(`cannot set name of %v to "%s": name already used`, op, name))
	}
	done := false
	for n, o := range c.opDict {
		if o == op {
			delete(c.opDict, n)
			c.opDict[name] = op
			done = true
			break
		}
	}
	if !done {
		panic(fmt.Errorf(`cannot set name of %v to "%s": no such operator`, op, name))
	}
}
