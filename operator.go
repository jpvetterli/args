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

type op uint8

const (
	opcond op = iota
	opDump
	opImport
	opInclude
	opReset
	opSkip
)

var dispatch = map[string]op{
	"cond":    opcond,
	"dump":    opDump,
	"import":  opImport,
	"include": opInclude,
	"reset":   opReset,
	"--":      opSkip,
}

// operator returns the operator with the given name or nil
func (a *Parser) operator(name string) operator {
	o, ok := dispatch[name]
	if ok {
		switch o {
		case opcond:
			return &condOperator{parser: a}
		case opInclude:
			return &includeOperator{parser: a}
		case opDump:
			return &dumpOperator{parser: a}
		case opImport:
			return &importOperator{parser: a}
		case opReset:
			return &resetOperator{parser: a}
		case opSkip:
			return &skipOperator{}
		default:
			panic(fmt.Errorf("bug: %v (%s)", o, name)) // forgot something
		}
	}
	return nil
}

// condOperator implements cond. cond has two mandatory parameters, "if" and
// "then" and an optional one, "else". All three take exactly one value. The
// value of "if" is interpreted as a symbol name (without the symbol prefix). If
// the symbol exists, and its resolved value is not empty, then the value of
// "then" is parsed, else the value of "else" is parsed, if it is specified.
// Note that if the value of "if" cannot be resolved, it is not empty, and the
// value of "then" is parsed.
type condOperator struct {
	parser *Parser
}

func (o *condOperator) handle(value string) error {
	local := NewParser(nil)
	condIf := ""
	condThen := ""
	condElse := ""
	local.Def("if", &condIf)
	local.Def("then", &condThen)
	local.Def("else", &condElse).Opt()
	err := local.Parse(value)
	if err != nil {
		return err
	}
	v, err := o.parser.symbols.get(condIf)
	if err != nil {
		return err
	}
	if v != nil && len(v.s) > 0 {
		return o.parser.Parse(condThen)
	}
	if len(condElse) > 0 {
		return o.parser.Parse(condElse)
	}
	return nil
}

// dumpOperator implements dump. dump takes an optional "comment" parameter and
// a series of anonymous values. It interprets the values as symbols (without
// the symbol prefix) and prints their names and values line by line on standard
// error. The value is preceded by R if resolved, else by U. If a comment is
// specified it is printed first.
type dumpOperator struct {
	parser *Parser
}

func (o *dumpOperator) handle(value string) error {
	local := NewParser(nil)
	comment := ""
	var symbols []string
	local.Def("", &symbols)
	local.Def("comment", &comment).Opt()
	local.Parse(value)
	if len(comment) > 0 {
		fmt.Fprintln(os.Stderr, comment)
	}
	for _, s := range symbols {
		if v, ok := o.parser.symbols.table[s]; ok {
			r := 'U'
			if v.resolved {
				r = 'R'
			}
			fmt.Fprintf(os.Stderr, "%s (%c) %s\n", s, r, v.s)
		}
	}
	return nil
}

// importOperator implements import. import takes a series of values, which it
// interprets as keys of environment variables. It gets the corresponding values
// from the environment and puts them in the symbol table unless there is
// already an entry with the same name. If they are not in the environment empty
// values are used.
type importOperator struct {
	parser *Parser
}

func (o *importOperator) handle(value string) error {
	local := NewParser(nil)
	var keys []string
	local.Def("", &keys)
	local.Parse(value)
	for _, k := range keys {
		v := os.Getenv(k)
		if _, exists := o.parser.symbols.table[k]; !exists {
			o.parser.symbols.table[k] = &symval{s: v}
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
// include tries to extract a key-value pair from each line of the file, and if
// a value is found with a matching key, it is inserted into the symbol table
// using the key as symbol or using the translation, if specified. The
// "extractor" parameter specifies a custom regular expression for extracting
// key-value pairs. The default extractor is \s*(\S+)\s*=\s*(\S+)\s*. It is an
// error to specify an extractor in basic mode (no keys specified).
type includeOperator struct {
	parser *Parser
}

func (o *includeOperator) handle(value string) error {
	local := NewParser(nil)
	filename := ""
	keys := ""
	extractor := ""
	local.Def("", &filename)
	local.Def("keys", &keys).Opt()
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
		return o.parser.Parse(string(data))
	}

	// key selection mode

	if len(extractor) == 0 {
		extractor = `\s*(\S+)\s*=\s*(\S+)\s*`
	}

	re, err := regexp.Compile(extractor)
	if err != nil {
		panic(fmt.Errorf(`compilation of extractor "%s" failed: %v`, extractor, err))
	}

	kvs, err := pairs(o.parser.custom, []byte(keys))
	if err != nil {
		return err
	}

	kvmap := make(map[string]string)
	for _, kv := range kvs {
		if len(kv.Name) == 0 {
			kvmap[kv.Value] = kv.Value
		} else {
			kvmap[kv.Name] = kv.Value
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
				o.parser.symbols.put(string(o.parser.custom.SymbolPrefix())+name, capture[2])
			}
		}
	}

	return nil
}

// resetOperator implements reset. reset takes a series of values, which it
// interprets as symbols (without the symbol prefix) and removes them from the
// symbol table, if present.
type resetOperator struct {
	parser *Parser
}

func (o *resetOperator) handle(value string) error {
	local := NewParser(nil)
	var symbols []string
	local.Def("", &symbols)
	local.Parse(value)
	for _, s := range symbols {
		delete(o.parser.symbols.table, s)
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
