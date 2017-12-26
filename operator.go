package args

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

type operator interface {
	handle(value string) error
}

// operator returns the operator with the given name or nil
func (a *Parser) operator(name string) operator {
	o, ok := a.config.opDict[name]
	if ok {
		switch o {
		case OpCond:
			return &condOperator{parser: a}
		case OpInclude:
			return &includeOperator{parser: a}
		case OpDump:
			return &dumpOperator{parser: a}
		case OpImport:
			return &importOperator{parser: a}
		case OpMacro:
			return &macroOperator{parser: a}
		case OpReset:
			return &resetOperator{parser: a}
		case OpSkip:
			return &skipOperator{}
		default:
			panic(fmt.Errorf("bug: %v (%s)", o, name)) // forgot something
		}
	}
	return nil
}

// condOperator implements cond. cond has two mandatory parameters, "if" and
// "then" and an optional one, "else". All three take exactly one value, all
// verbatim. The value of "if" is interpreted as a parameter name or symbol. The
// value of "if" evaluates to true if the symbol exists or if the parameter has
// been set at least once. If the parameter has no been defined, an error
// occurs. When the value of "if" is true then the value of "then" is parsed,
// else the value of "else" is parsed, if specified.
type condOperator struct {
	parser *Parser
}

func (o *condOperator) handle(value string) error {
	local := SubParser(o.parser)
	condIf := ""
	condThen := ""
	condElse := ""
	local.Def("if", &condIf).Verbatim()
	local.Def("then", &condThen).Verbatim()
	local.Def("else", &condElse).Opt().Verbatim()
	err := local.Parse(value)
	if err != nil {
		return err
	}

	ifVal, isSymbol := symbol(condIf, o.parser)
	var cond bool

	if isSymbol {
		_, cond = o.parser.symbols.table[ifVal]
	} else {
		p, ok := o.parser.params[ifVal]
		if !ok {
			return fmt.Errorf(`cond/if: parameter "%s" not defined`, ifVal)
		}
		cond = ok && p.count > 0
	}
	if cond {
		return o.parser.Parse(condThen)
	}
	if len(condElse) > 0 {
		return o.parser.Parse(condElse)
	}
	return nil
}

// dumpOperator implements dump. dump takes an optional "comment" parameter and
// a series of anonymous values. All values are taken verbatim. dump interprets
// the values as parameter names and symbols and prints them line by line on
// standard error with their current values. The value of a symbol is preceded
// by R if resolved, else by U. A name or symbol is preceded by ? if undefined.
// If a comment is specified, it is printed first.
type dumpOperator struct {
	parser *Parser
}

func (o *dumpOperator) handle(value string) error {
	local := SubParser(o.parser)
	comment := ""
	var names []string
	local.Def("", &names).Verbatim()
	local.Def("comment", &comment).Opt().Verbatim()
	local.Parse(value)
	if len(comment) > 0 {
		fmt.Fprintln(os.Stderr, comment)
	}
	for _, n := range names {
		if s, isSymbol := symbol(n, o.parser); isSymbol {
			if v, ok := o.parser.symbols.table[s]; ok {
				r := 'U'
				if v.resolved {
					r = 'R'
				}
				fmt.Fprintf(os.Stderr, "%s %c %s\n", n, r, v.s)
			} else {
				fmt.Fprintf(os.Stderr, "? %s\n", n)
			}
		} else {
			if p, ok := o.parser.params[n]; ok {
				if len(n) == 0 {
					n = "[]"
				}
				fmt.Fprintf(os.Stderr, "%s %v\n", n, reflValue(p.target))
			} else {
				if len(n) == 0 {
					n = "[]"
				}
				fmt.Fprintf(os.Stderr, "? %s\n", n)
			}
		}
	}
	return nil
}

// importOperator implements import. import takes a series of verbatim values,
// which it interprets as symbols. For each symbol a value is taken from the
// environment variable with the corresponding name (smybol prefix removed). The
// value is inserted in the symbol table unless there is already an entry for
// the symbol ("first wins" principle). If the environment variable does not
// exist,nothing is done.
type importOperator struct {
	parser *Parser
}

func (o *importOperator) handle(value string) error {
	local := SubParser(o.parser)
	var symbols []string
	local.Def("", &symbols).Verbatim()
	local.Parse(value)
	for _, sym := range symbols {
		if k, isSymbol := symbol(sym, o.parser); isSymbol {
			if v, ok := os.LookupEnv(k); ok {
				o.parser.symbols.put(sym, v)
			}
		} else {
			return fmt.Errorf(`import: "%s": symbol prefix missing (%c)`, sym, o.parser.config.GetSpecial(SpecSymbolPrefix))
		}
	}
	return nil
}

// includeOperator implements include. include works in two different modes.
//
// In basic mode, include takes a file name from a mandatory and anonymous
// parameter. It reads the file and passes its content to Parse.
//
// In key selection mode include takes a file name as in the first mode, but
// takes also a "keys" parameter and an optional "extractor" parameter. The
// value of "keys" is interpreted as a series of standalone keys or
// key-translation pairs (using the current separator character of the parser).
// If there is no translation, the key translates to itself. include  extracts
// name-value pairs from each line of the file, and if it finds a name matching
// one of the keys, it uses the value to set a parameter or a symbol depending
// on the translated key. As always a symbol is set only if it does not already
// exist ("first wins" principle).
//
//The "extractor" parameter specifies a custom regular expression for extracting
//key-value pairs. The default extractor is \s*(\S+)\s*=\s*(\S+)\s*. It is an
//error to specify an extractor in basic mode (no keys specified).
//
// Keys are taken verbatim, but the file name and the extractor are resolved.
type includeOperator struct {
	parser *Parser
}

func (o *includeOperator) handle(value string) error {
	local := SubParser(o.parser)
	filename := ""
	keys := ""
	extractor := ""
	local.Def("", &filename)
	local.Def("keys", &keys).Opt().Verbatim()
	local.Def("extractor", &extractor).Opt()
	local.Parse(value)

	// detect cycles using canonical file name
	path, err := filepath.Abs(filename)
	if err != nil {
		return err
	}
	if _, ok := o.parser.cycle[path]; ok {
		return fmt.Errorf(`cyclical include dependency with file "%s"`, filename)
	}
	o.parser.cycle[path] = true
	defer func() {
		delete(o.parser.cycle, path)
	}()

	// standard mode: parse the file
	if len(keys) == 0 {
		if len(extractor) > 0 {
			return fmt.Errorf("include: specify extractor only with keys parameter")
		}
		data, e := ioutil.ReadFile(path)
		if e != nil {
			return fmt.Errorf("include: %v", e)
		}
		// remove byte order mark if any
		if data[0] == 0xef && data[1] == 0xbb || data[2] == 0xbf {
			data = data[3:]
		}
		return o.parser.ParseBytes(data)
	}

	// key selection mode

	if len(extractor) == 0 {
		extractor = `\s*(\S+)\s*=\s*(\S+)\s*`
	}

	re, err := regexp.Compile(extractor)
	if err != nil {
		panic(fmt.Errorf(`compilation of extractor "%s" failed: %v`, extractor, err))
	}

	kvmap := make(map[string]string)
	nvp := newNameValParser(o.parser, []byte(keys))
	for {
		n, v, e := nvp.next()
		if e != nil {
			return e
		}
		if n == nil && v == nil {
			break
		}

		if !v.resolved {
			return fmt.Errorf(`include: cannot resolve key "%s"`, v.s)
		}

		if n == nil {
			kvmap[v.s] = v.s
		} else {
			kvmap[n.s] = v.s
		}
	}

	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	r := bufio.NewReader(f)

loop:
	for {
		line, e := r.ReadString('\n')
		switch e {
		case nil:
		case io.EOF:
			if len(line) == 0 {
				break loop
			}
		default:
			return e
		}

		capture := re.FindStringSubmatch(line)
		if len(capture) == 3 {
			if name, ok := kvmap[capture[1]]; ok {
				o.parser.setValue(name, capture[2])
			}
		}
	}

	return nil
}

// macroOperator implements macro. macro takes a series of values verbatim,
// which it interprets as symbols, gets their values from the symbol table
// without resolving them, and passes them recursively to Parse. An error occurs
// if values are not symbols, if any symbol is not found, or if parsing fails.
type macroOperator struct {
	parser *Parser
}

func (o *macroOperator) handle(value string) error {
	local := SubParser(o.parser)
	var symbols []string
	local.Def("", &symbols).Verbatim()
	local.Parse(value)
	code := []string{}
	for _, s := range symbols {
		if sym, isSymbol := symbol(s, o.parser); isSymbol {
			if v, ok := o.parser.symbols.table[sym]; ok {
				code = append(code, v.s)
			} else {
				return fmt.Errorf(`macro: symbol "%s" undefined`, s)
			}
		} else {
			return fmt.Errorf(`macro: "%s": symbol prefix missing (%c)`, s, o.parser.config.GetSpecial(SpecSymbolPrefix))
		}
	}
	err := o.parser.ParseStrings(code)
	if err != nil {
		return fmt.Errorf(`macro: parsing of %v failed %v`, code, err)
	}
	return nil
}

// resetOperator implements reset. reset takes a series of values verbatim,
// which it interprets as symbols and removes them from the symbol table, if
// present. An error occurs if values are not symbols.
type resetOperator struct {
	parser *Parser
}

func (o *resetOperator) handle(value string) error {
	local := SubParser(o.parser)
	var symbols []string
	local.Def("", &symbols).Verbatim()
	local.Parse(value)
	for _, s := range symbols {
		if sym, isSymbol := symbol(s, o.parser); isSymbol {
			delete(o.parser.symbols.table, sym)
		} else {
			return fmt.Errorf(`reset: "%s": symbol prefix missing (%c)`, s, o.parser.config.GetSpecial(SpecSymbolPrefix))
		}
	}
	return nil
}

// skipOperator implements skip. skip ignores the value and can be used for
// commenting. Any quotes in the value must be balanced.
type skipOperator struct {
}

func (o *skipOperator) handle(value string) error {
	return nil
}

// symbols returns s without the symbol prefix and true if s starts with the
// symbol prefix else it returns s and false.
func symbol(s string, p *Parser) (string, bool) {
	r := []rune(s)
	if len(r) > 1 && r[0] == p.config.GetSpecial(SpecSymbolPrefix) {
		return string(r[1:]), true
	}
	return s, false
}
