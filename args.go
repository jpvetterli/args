package args

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
)

// Parser methods define and parse command line parameters. There are also
// methods for specifying and producing command documentation.
type Parser struct {
	custom  *Specials
	params  map[string]*Param
	seq     []string // names in definition sequence
	doc     []string
	targets map[interface{}]bool // duplicate detection
	symbols symtab
	cycle   map[string]bool // include cycle detector
}

// NewParser returns a Parser with a configuration of special characters.
// A default configuration is used if it is nil.
func NewParser(configuration *Specials) *Parser {
	if configuration == nil {
		configuration = NewSpecials("")
	}
	return &Parser{
		custom:  configuration,
		params:  make(map[string]*Param),
		seq:     make([]string, 0),
		doc:     make([]string, 0),
		targets: make(map[interface{}]bool),
		symbols: newSymtab(configuration.SymbolPrefix()),
		cycle:   make(map[string]bool),
	}
}

// Def defines a parameter with a name and a target to take one or more values.
// It returns a Param which can be used to optionally configure various details.
// This is designed to allow chaining of methods, so that a complete parameter
// definition can be made with a one-liner.
//
// Example of chaining:
//    var help bool
//    a.Def("-help", &help).Aka("-h").Aka("?").Opt().Doc("print a usage summary and exit")
//
// When target points to an array, the parameter takes a number of values
// exactly equal to its length. When target points to a slice, it takes a number
// of values not exceeding its capacity, unless the capacity is zero, which is
// interpreted as no limit. It is the only Parser method which can officially
// panic. It panics if the name is already used, if the name contains a
// character other than a letter, a digit, a hyphen or an underscore, if the
// target is not a pointer, or if the target is already assigned to another
// parameter.
func (a *Parser) Def(name string, target interface{}) *Param {

	// many functions rely on target being a pointer (see refl*)

	if reflect.ValueOf(target).Kind() != reflect.Ptr {
		panic(fmt.Errorf(`target for parameter "%s" is not a pointer`, name))
	}
	if _, ok := a.params[name]; ok {
		panic(fmt.Errorf(`parameter "%s" already defined`, name))
	}
	if _, ok := a.targets[target]; ok {
		panic(fmt.Errorf(`target for parameter "%s" is already assigned`, name))
	}
	if err := a.validate(name); err != nil {
		panic(err)
	}
	if a.operator(name) != nil {
		panic(fmt.Errorf(`parameter name "%s" is the name of an operator`, name))
	}
	p := Param{dict: a, name: name, target: target}

	v := reflValue(target)
	switch v.Kind() {
	case reflect.Array:
		p.limit = v.Len()
	case reflect.Slice:
		p.limit = v.Cap()
	default:
		p.limit = 1
	}

	a.params[name] = &p
	a.targets[target] = true
	a.seq = append(a.seq, name)
	return &p
}

// Parse parses s to extract and assign values to parameter targets.  The result
// is nil unless there is an error.  The input syntax is explained in the
// package documentation.
func (a *Parser) Parse(s string) error {
	err := a.parse(s)
	if err != nil {
		return err
	}
	return a.verify()
}

// parse parses s. It can be used recursively.
func (a *Parser) parse(s string) error {

	// build list of name-value pairs
	namevals, e := pairs(a.custom, []byte(s))
	if e != nil {
		return e
	}

loop:
	for _, nv := range namevals {

		if len(nv.Name) == 0 {
			recursive, err := a.parseSingleton(nv)
			if err != nil {
				return err
			}
			if recursive {
				err = a.parse(nv.Value)
				if err != nil {
					return err
				}
				continue loop
			}
		} else {
			err := a.parsePair(nv)
			if err != nil {
				return err
			}
		}

		operator := a.operator(nv.Name)
		if operator != nil {
			err := operator.handle(nv.Value)
			if err != nil {
				return err
			}
			continue loop
		}

		err := a.setValue(nv.Name, nv.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// setValue adds value to symbol table or set parameter value
func (a *Parser) setValue(name, value string) error {
	if !a.symbols.put(name, value) {
		if p, ok := a.params[name]; ok {
			err := p.parseValues(p.split(value))
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf(`parameter not defined: "%s"`, name)
		}
	}
	return nil
}

// parseSingleton parses nv with an empty name and returns true to request
// recursive parsing. It updates nv.
func (a *Parser) parseSingleton(nv *nameValue) (bool, error) {
	symv, _, err := substitute(nv.Value, &a.symbols)
	if err != nil {
		return false, err
	}

	isBoolParamWithStandaloneName := func() bool {
		if p, ok := a.params[symv.s]; ok {
			return reflTakesBool(p.target)
		}
		return false
	}

	switch {

	// standalone name
	case symv.resolved && isBoolParamWithStandaloneName():
		nv.Name, nv.Value = symv.s, "true"

	// actual standalone value
	default:
		p, ok := a.params[""]
		if !ok {
			return false, fmt.Errorf(`unexpected standalone value: "%s"`, nv.Value)
		}
		if !p.verbatim {
			if !symv.resolved {
				return false, fmt.Errorf(`cannot resolve standalone value "%s"`, nv.Value)
			}
			nv.Value = symv.s
		} // else leave verbatim value

	}
	return false, nil
}

// parsePair parses nv with a non-empty name.
func (a *Parser) parsePair(nv *nameValue) error {
	symv, _, err := substitute(nv.Name, &a.symbols)
	if err != nil {
		return err
	}
	if !symv.resolved {
		return fmt.Errorf(`cannot resolve name in "%s %c %s"`, nv.Name, a.custom.Separator(), nv.Value)
	}
	nv.Name = symv.s

	p, ok := a.params[nv.Name]
	if ok {
		symv, _, err = substitute(nv.Value, &a.symbols)
		if err != nil {
			return decorate(err, nv.Name)
		}
		if !symv.resolved {
			if !p.verbatim {
				return fmt.Errorf(`cannot resolve value in "%s %c %s"`, nv.Name, a.custom.Separator(), nv.Value)
			}
		}
		nv.Value = symv.s
	}
	// else: do not resolve value if not a parameter (either a symbol or a wrong name)

	return nil
}

// ParseStrings calls Parse with all arguments joined with a blank.
func (a *Parser) ParseStrings(s []string) error {
	return a.Parse(strings.Join(s, " "))
}

// Doc sets lines of help text for the command as a whole.
func (a *Parser) Doc(s ...string) {
	a.doc = s
}

// PrintDoc uses a Writer to print the command help text, followed by the help
// text of each parameter in definition sequence. Any relevant information about
// parameters is included.
//
// If any s is specified, the first line of command help text is assumed
// to contain formatting verbs and is printed with Fprintf, else it is
// printed with Fprintln.
//
// If no help text was supplied with Doc, a default help text is provided
// which depends on the length of s and on whether any parameter was
// defined:
//
// No s, no parameters:
//	the command takes no parameter\n
// No s, parameters specified:
//	the command takes these parameters:\n
// s specified, no parameters:
//	Usage: %v\n
// s specified, parameters specified:
//	Usage: %v parameters...\n
//
//	Parameters:
// If a single s is specified the single value is selected,
// else all values are taken, which will probably look a bit
// strange in the output.
func (a *Parser) PrintDoc(w io.Writer, s ...interface{}) {
	switch {
	case len(a.doc) > 0:
		for i, line := range a.doc {
			if i == 0 {
				if len(s) > 0 {
					fmt.Fprintf(w, line, s...)
				} else {
					fmt.Fprintln(w, line)
				}
			} else {
				fmt.Fprintln(w, line)
			}
		}
	case len(s) == 0 && len(a.seq) == 0:
		fmt.Fprintln(w, "the command takes no parameter")
	case len(s) == 0 && len(a.seq) > 0:
		fmt.Fprintln(w, "the command takes these parameters:")
	case len(s) == 1 && len(a.seq) == 0:
		fmt.Fprintf(w, "Usage: %v\n", s[0])
	case len(s) == 1 && len(a.seq) > 0:
		fmt.Fprintf(w, "Usage: %v parameters...\n\nParameters:\n", s[0])
	case len(s) > 0 && len(a.seq) == 0:
		fmt.Fprintf(w, "Usage: %v\n", s)
	case len(s) > 0 && len(a.seq) > 0:
		fmt.Fprintf(w, "Usage: %v parameters...\n\nParameters:\n", s)
	}

	syn := buildSynonyms(a)
	for _, n := range a.seq {
		p := a.params[n]
		value := reflValue(p.target)
		details := ""
		typ := value.Type()
		switch value.Kind() {
		case reflect.Slice:
			typ = typ.Elem()
			details = ""
			if p.splitter != nil {
				details += fmt.Sprintf(", split: %v", p.splitter)
			}
			if p.limit > 0 {
				details += fmt.Sprintf(", 0-%d value%s", p.limit, plural(p.limit))
			} else {
				details += ", any number of values"
			}
			if value.Len() > 0 {
				details += fmt.Sprintf(" (default: %v)", value)
			}
		case reflect.Array:
			typ = typ.Elem()
			details = ""
			if p.splitter != nil {
				details += fmt.Sprintf(", split: %v", p.splitter)
			}
			details += fmt.Sprintf(", exactly %d value%s", p.limit, plural(p.limit))
		default:
			// scalar
			if p.limit == 0 {
				details = fmt.Sprintf(", optional (default: %v)", value)
			}
		}
		if n == p.name {
			info := fmt.Sprintf("type: %s%s", typ, details)
			n = syn[n]
			next := -1
			if len(n) > 8 {
				fmt.Fprintf(w, "  %s\n", n)
				next = 0
			} else {
				if len(p.doc) > 0 {
					fmt.Fprintf(w, "  %-8s %s\n", n, p.doc[0])
					next = 1
				} else {
					fmt.Fprintf(w, "  %-8s %s\n", n, info)
				}
			}
			if next >= 0 {
				for _, s := range p.doc[next:] {
					fmt.Fprintf(w, "  %-8s %s\n", "", s)
				}
				fmt.Fprintf(w, "  %-8s %s\n", "", info)
			}
		}
	}
}

// PrintConfig uses a Writer to print the parser configuration. This consists of
// the special characters configured in the parser. Nothing is printed when no
// parameter is defined.
func (a *Parser) PrintConfig(w io.Writer) {
	if len(a.seq) > 0 {
		fmt.Fprintf(w, "Special characters:\n")
		fmt.Fprintf(w, "  %c        %s\n", a.custom.SymbolPrefix(), "symbol prefix")
		fmt.Fprintf(w, "  %c        %s\n", a.custom.Separator(), "name-value separator")
		fmt.Fprintf(w, "  %c        %s\n", a.custom.LeftQuote(), "opening quote")
		fmt.Fprintf(w, "  %c        %s\n", a.custom.RightQuote(), "closing quote")
		fmt.Fprintf(w, "  %c        %s\n", a.custom.Escape(), "escape")
	}
}

// Param methods specify optional details of parameter definitions. A Param is
// created by Parser.Def, which gives the parameter a name and sets the target
// that will take values. Param methods are designed to support chaining. Any
// error detected by a Param function results in a panic (as is also the case
// for Parser.Def). This is natural since a definition error is a bug in the
// program, which cannot continue safely. On the other hand,  errors originating
// from user input don't cause panics.  Panics are documented in the relevant
// functions.
type Param struct {
	dict     *Parser
	name     string // the canonical name
	limit    int    // limit for number of values (array: exact, slice: max unless 0, scalar: 0 for opt)
	count    int    // actual number of values seen
	verbatim bool
	target   interface{}
	scan     func(value string, target interface{}) error
	splitter *regexp.Regexp
	doc      []string
}

// Aka sets alias as a synonym for the parameter name.  Panics if alias is
// already used as a name or synonym for any parameter.
func (p *Param) Aka(alias string) *Param {
	if _, ok := p.dict.params[alias]; ok {
		panic(fmt.Errorf(`synonym "%s" clashes with an existing parameter name or synonym`, alias))
	}
	if err := p.dict.validate(alias); err != nil {
		panic(err)
	}
	p.dict.params[alias] = p
	p.dict.seq = append(p.dict.seq, alias)
	return p
}

// Opt indicates that the parameter is optional. Only single-value parameters
// can be specified as optional. To make multi-value parameters optional use a
// slice. Panics if the target is an array or a slice.
func (p *Param) Opt() *Param {
	if reflLen(p.target) >= 0 {
		panic(fmt.Errorf(`parameter "%s" is multi-valued and cannot be optional (hint: use a slice with length 0 instead)`, p.name))
	}
	p.limit = 0
	return p
}

// Verbatim indicates that the parameter value can contain unresolved symbol
// references. Only parameters with a target taking strings can be specified as
// verbatim. Panics if the target points to a non-string.
func (p *Param) Verbatim() *Param {
	ok := true

	v := reflValue(p.target)

	switch v.Kind() {
	case reflect.String:
	case reflect.Array:
		ok = v.Type().Elem().Kind() == reflect.String
	case reflect.Slice:
		ok = v.Type().Elem().Kind() == reflect.String
	default:
		ok = false
	}
	if !ok {
		what := "anonymous parameter"
		if len(p.name) > 0 {
			what = `parameter "` + p.name + `"`
		}
		panic(fmt.Errorf(`%s cannot be verbatim because its target of type %v cannot take a string`, what, reflect.TypeOf(p.target)))
	}
	p.verbatim = true
	return p
}

// Doc sets lines of help text for the parameter.
func (p *Param) Doc(s ...string) *Param {
	p.doc = s
	return p
}

// Scan sets a function to scan one parameter value into the target. When no
// function is provided, values are scanned with a builtin function, which
// supports all the same basic types as the Parse* functions of the strconv
// package. When a target is an array or slice, each value is scanned separately
// into corresponding elements of the target. When a custom scanner function is
// configured, any unset initial value is scanned to ensure agreement.
func (p *Param) Scan(f func(string, interface{}) error) *Param {
	p.scan = f
	return p
}

// Split sets a regular expression for splitting values. The expression is
// compiled with regexp.Compile. Panics if the target is neither an array nor a
// slice or if the regular expression is invalid.
func (p *Param) Split(regex string) *Param {
	k := reflValue(p.target).Kind()
	if k != reflect.Array && k != reflect.Slice {
		panic(fmt.Errorf(`cannot split values of parameter "%s" which is not multi-valued`, p.name))
	}
	var err error
	p.splitter, err = regexp.Compile(regex)
	if err != nil {
		panic(fmt.Errorf(`compilation of split expression "%s" for parameter "%s" failed: %v`, regex, p.name, err))
	}
	return p
}

// helpers

// validate verifies a name (no symbol prefix allowed)
func (a *Parser) validate(name string) error {
	for _, r := range []rune(name) {
		if !valid(r) {
			return fmt.Errorf(`"%s" cannot be used as parameter name or alias because it includes the character '%c'`, name, r)
		}
	}
	return nil
}

// parseValues converts values and assigns them to targets
func (p *Param) parseValues(values []string) error {
	var err error
	count := len(values)
	v := reflValue(p.target)
	switch v.Kind() {
	case reflect.Array:
		total := count + p.count
		if total > p.limit {
			err = fmt.Errorf("too many values specified, expected %d", p.limit)
		} else {
			// scan all values
			for i, value := range values {
				if err = p.assign(value, reflElementAddr(p.count+i, v)); err != nil {
					break
				}
			}
			p.count = total
		}
	case reflect.Slice:
		total := count + p.count
		switch {
		case p.limit == 0:
			// any number of values is okay
		case total > p.limit:
			err = fmt.Errorf("%d value%s specified, at most %d expected", total, plural(total), p.limit)
		}
		if err == nil {
			if total > v.Len() {
				// grow the slice
				s := reflect.MakeSlice(v.Type(), total, total)
				// no need to copy since no way to skip over existing values
				// do it anyway in just in case (e.g. extract values by splitting and skipping)
				reflect.Copy(s, v)
				v.Set(s)
			}
			// scan all values
			for i, value := range values {
				if err = p.assign(value, reflElementAddr(p.count+i, v)); err != nil {
					break
				}
			}
			p.count = total
		}
	default:
		// if too many values specified, the last wins
		err = p.assign(values[len(values)-1], p.target)
		p.count++
	}
	if err != nil {
		err = decorate(err, p.name)
	}
	return err
}

// split splits value around a splitter regular expression. It returns the input
// if the parameter has no splitter.
func (p *Param) split(value string) []string {
	if p.splitter == nil {
		return []string{value}
	}
	return p.splitter.Split(value, -1)
}

// assign converts value and assigns it to target. It does it with a custom
// scanner if defined for the parameter.
func (p *Param) assign(value string, target interface{}) error {
	if p.scan != nil {
		return p.scan(value, target)
	}
	return typescan(value, target)
}

// verify verifies that omitted parameters can be omitted and that default
// values of omitted parameters are valid.
func (a *Parser) verify() error {
	for n, p := range a.params {
		if n == p.name {
			value := reflValue(p.target)
			switch value.Kind() {
			case reflect.Slice:
				for i := p.count; i < reflLen(p.target); i++ {
					if p.scan != nil {
						// scan remaining initial values to ensure they are okay
						e := p.scan(fmt.Sprint(reflElement(i, value)), reflCopy(reflElementAddr(i, value)))
						if e != nil {
							return decorate(fmt.Errorf("invalid default value at offset %d: %v", i, e), n)
						}
					}
				}
			case reflect.Array:
				if p.count != p.limit {
					return decorate(fmt.Errorf("%d value%s specified but exactly %d expected", p.count, plural(p.count), p.limit), n)
				}
			default:
				// single-valued parameter
				if p.count < 1 {
					if p.limit != 0 {
						return decorate(fmt.Errorf("mandatory parameter not set"), n)
					}
					// scan initial value (into a copy) to ensure it's okay
					if p.scan != nil {
						e := p.scan(fmt.Sprint(value), reflCopy(p.target))
						if e != nil {
							return decorate(fmt.Errorf("invalid default value: %v", e), n)
						}
					}
				}
			}
		}
	}
	return nil
}

// buildSynonyms is a helper for PrintDoc.
func buildSynonyms(a *Parser) map[string]string {
	synonyms := make(map[string]string)
	for _, n := range a.seq {
		p := a.params[n]
		if n == p.name {
			if len(n) == 0 {
				synonyms[n] = "(nameless)"
			} else {
				synonyms[n] = n
			}
		} else {
			synonyms[p.name] += ", " + n
		}
	}
	return synonyms
}

// decorate adds name information to error messages.
func decorate(err error, name string) error {
	if len(name) == 0 {
		name = "anonymous parameter"
	}
	return fmt.Errorf(`Parse error on %s: %v`, name, err)
}

// plural returns "" if n == 1 else "s"
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// helpers (reflection)

// reflLen returns length of array or slice or -1 using reflection
func reflLen(target interface{}) int {
	v := reflect.Indirect(reflect.ValueOf(target))
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		return v.Len()
	}
	return -1
}

// reflValue returns the value of target using reflection
func reflValue(target interface{}) reflect.Value {
	return reflect.Indirect(reflect.ValueOf(target))
}

// reflCopy returns a new copy of target using reflection
func reflCopy(target interface{}) interface{} {
	return reflect.New(reflect.TypeOf(target).Elem()).Interface()
}

// reflElementAddr returns the address of the i-th element of target using
// reflection
func reflElementAddr(i int, v reflect.Value) interface{} {
	return v.Index(i).Addr().Interface()
}

// reflElement returns the i-th element of target using reflection
func reflElement(i int, v reflect.Value) interface{} {
	return v.Index(i).Interface()
}

// reflTakesBool returns true if the target takes a bool.
// It can be a simple variable, an array or a slice.
func reflTakesBool(target interface{}) bool {
	val := reflValue(target)
	switch val.Kind() {
	case reflect.Bool:
		return true
	case reflect.Array, reflect.Slice:
		return val.Type().Elem().Kind() == reflect.Bool
	default:
		return false
	}
}
