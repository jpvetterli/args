package args

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
)

// TODO: review all doc
// TODO: remove all the TODOs in all files
// TODO: skip BOM if 1st character of an input file

// Parser methods are used to define, parse, and document command line
// parameters.
type Parser struct {
	custom  *Specials // nil for standard marks
	params  map[string]*Param
	seq     []string // names in definition sequence
	doc     []string
	targets map[interface{}]bool // duplicate detection
	symbols map[string]string
	subst   substituter
}

// NewParser returns a Parser with a configuration of special characters.
func NewParser(configuration *Specials) *Parser {
	return &Parser{
		custom:  configuration,
		params:  make(map[string]*Param),
		seq:     make([]string, 0),
		doc:     make([]string, 0),
		targets: make(map[interface{}]bool),
		symbols: make(map[string]string),
		subst:   *newSubstituter(configuration.SymbolPrefix()),
	}
}

// Def defines a parameter with a name and a target to take one or more values.
// It returns a Param to allow chaining with Param functions.
//
// Example of chaining:
//  func setup(a Parser) (err error) {
//    defer func() {
//      err = fmt.Errorf("%v", recover())
//    }()
//    var help bool
//    a.Def("-help", &help).Aka("-h").Aka("?").Opt().Doc("print a usage summary and exit")
//    return
//  }
//
// When target points to an array, the parameter takes a number of values
// exaclty equal to the length. When target points to a slice, it takes a number
// of values not exceeding its capacity, unless the capacity is zero, which is
// interpreted as no limit. Panics if name is already used. Panics if target is
// not a pointer. It is the only Parser function which can panic (except for
// bugs).
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

	p := Param{dict: a, name: name, target: target}
	a.params[name] = &p
	a.targets[target] = true
	a.seq = append(a.seq, name)
	return &p
}

// Parse extracts parameter values from the input. The result is nil unless
// there is an error. Values are scanned and the targets passed to Def are set.
// The input syntax is explained in the package documentation.
func (a *Parser) Parse(s string) error {

	seen := make(map[string]int)
	for n, v := range a.params {
		if n == v.name {
			seen[n] = 0
		}
	}

	namevals, err := a.extractAndResolveNameValues([]byte(s))
	if err != nil {
		return err
	}

	_, list, err := normalize(namevals, a.synonyms())
	if err != nil {
		return err
	}

	for _, nv := range list {
		name := nv[0]
		switch len(nv) {
		case 1:
			seen[name] = 1
			err = parseStandaloneName(a.params[name])
		default:
			seen[name] += len(nv) - 1
			p := a.params[name]
			err = parseValues(p, splitValues(p, nv[1:]))
		}
		if err != nil {
			return err
		}
	}

	return a.verifyNotSeen(seen)
}

// ParseStrings is a wrapper for Parse, which is passed all arguments joined
// with a blank.
func (a *Parser) ParseStrings(s []string) error {
	return a.Parse(strings.Join(s, " "))
}

// Doc sets lines of help text for the command as a whole.
func (a *Parser) Doc(s ...string) {
	a.doc = s
}

// PrintDoc uses the Writer to print the command help text, followed by the help
// text of each parameter in definition sequence. Any relevant information about
// parameters is included.
func (a *Parser) PrintDoc(w io.Writer) {
	for _, s := range a.doc {
		fmt.Fprintln(w, s)
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
			if c := value.Cap(); c > 0 {
				details = fmt.Sprintf(", 0-%d value%s", c, plural(c))
			} else {
				details = ", any number of values"
			}
			if value.Len() > 0 {
				details += fmt.Sprintf(" (default: %v)", value)
			}
		case reflect.Array:
			typ = typ.Elem()
			details = fmt.Sprintf(", exactly %d value%s", value.Len(), plural(value.Len()))
		default:
			// scalar
			if p.opt {
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

// Param methods are used to specify details of parameter definitions. A Param
// is created by Parser.Def, which names a parameter and sets the target that
// will take one or more values. The functions are designed to support chaining.
// Any error detected by a Param function results in a panic (as is also the
// case for Parser.Def). This is natural since a program cannot continue safely
// after a definition error. Panics occur as documented in the functions.
type Param struct {
	dict     *Parser
	name     string // the canonical name
	opt      bool
	target   interface{}
	scan     func(value string, target interface{}) error
	splitter *regexp.Regexp
	doc      []string
}

// Aka sets name as a synonym for the parameter. Panics if the name is in use.
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
// slice with length zero as target. Panics if the target is an array or a
// slice.
func (p *Param) Opt() *Param {
	if reflLen(p.target) >= 0 {
		panic(fmt.Errorf(`parameter "%s" is multi-valued and cannot be optional (hint: use a slice with length 0 instead)`, p.name))
	}
	p.opt = true
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
// into corresponding elements of the target.
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
		panic(fmt.Errorf(`compilation of split expression "%s" for parameter "%s" failed (%s)`, regex, p.name, err.Error()))
	}
	return p
}

// helpers

// extractAndResolveNameValues does like its name says
func (a *Parser) extractAndResolveNameValues(input []byte) ([]*nameValue, error) {

	namevals, err := pairs(a.custom, input)
	if err != nil {
		return nil, err
	}

	namevals, err = a.resolve(namevals)
	if err != nil {
		return nil, err
	}
	return namevals, nil
}

// resolve substitutes symbol references in names and values.  If a name is a
// symbol, it is entered into the variables map of the Parser.
//
// There is special handling if a standalone values is modified by resolution:
// the result is interpreted again, recursively, since it could contain one or
// more name-value pairs. This allows to use symbols as macros.
// This behavior is limited to standalone values.
//
// An important detail is that unresolved symbol references are not considered
// an error at this step; they are simply passed along verbatim to the output.
func (a *Parser) resolve(namevals []*nameValue) ([]*nameValue, error) {
	result := make([]*nameValue, 0, cap(namevals))
	for _, nv := range namevals {
		if len(nv.Name) > 0 {

			name, _, _, err := a.subst.Substitute([]byte(nv.Name), &a.symbols)
			if err != nil {
				return nil, err
			}

			value, _, _, err := a.subst.Substitute([]byte(nv.Value), &a.symbols)
			if err != nil {
				return nil, err
			}

			sym := a.symbol(string(name))
			if len(sym) > 0 {
				// enter symbol into table only if absent ("first wins")
				if _, ok := a.symbols[sym]; !ok {
					a.symbols[sym] = string(value)
				}
			} else {
				result = append(result, &nameValue{Name: string(name), Value: string(value)})
			}
		} else {
			// resolve the standalone value, maybe recursively
			value, rcount, _, err := a.subst.Substitute([]byte(nv.Value), &a.symbols)
			if err != nil {
				return nil, err
			}
			if rcount > 0 {
				// something changed, this could be a "macro", recurse
				namevals1, err := a.extractAndResolveNameValues(value)
				if err != nil {
					return nil, err
				}
				for _, nv1 := range namevals1 {
					result = append(result, nv1)
				}
			} else {
				result = append(result, nv)
			}
		}
	}

	return result, nil
}

// symbol returns a symbol if the input corresponds to a symbol, else it returns
// an empty string. The input must start with exactly one symbol prefix and must
// be longer than 1. It must also be a valid name.
//
// Examples (with the symbol prefix $):
// 	for "$foo" returns "foo"
//	for "$$foo" returns ""
//	for "$"	returns "" (there is no empty symbol)
func (a *Parser) symbol(s string) string {
	if len(s) > 1 && strings.IndexRune(s, a.custom.SymbolPrefix()) == 0 {
		if sym := s[1:]; strings.IndexRune(sym, a.custom.SymbolPrefix()) != 0 {
			return sym
		}
	}
	return ""
}

// validate verifies a name (no symbol prefix allowed)
func (a *Parser) validate(name string) error {
	if strings.IndexRune(name, a.custom.SymbolPrefix()) == 0 {
		return fmt.Errorf(`"%s" cannot be used as parameter name or alias because it starts with the symbol prefix`, name)
	}
	return nil
}

// synonyms is a helper for Parse.
func (a *Parser) synonyms() map[string]string {
	s := make(map[string]string)
	for n, p := range a.params {
		s[n] = p.name
	}
	return s
}

// parseStandaloneName is a helper for Parse.
func parseStandaloneName(p *Param) error {
	// target must be a bool
	if v := reflValue(p.target); v.Kind() == reflect.Bool {
		v.SetBool(true)
		return nil
	}
	return fmt.Errorf(`Parse error on %s: not bool`, p.name)
}

// parseValues is a helper for Parse.
func parseValues(p *Param, values []string) error {
	var err error
	count := len(values)
	scanfunc := scanFunc(*p)
	v := reflValue(p.target)
	switch v.Kind() {
	case reflect.Array:
		// check: number of values must agree with array len
		if count != v.Len() {
			err = fmt.Errorf("%d value%s specified, but exactly %d expected", count, plural(count), v.Len())
		} else {
			// scan all values
			for i, value := range values {
				if err = scanfunc(value, reflElementAddr(i, v)); err != nil {
					break
				}
			}
		}
	case reflect.Slice:
		// check: number of values must agree with slice len and cap unless both 0
		switch {
		case v.Len() == 0 && v.Cap() == 0:
			// any number of values is okay
		case count > v.Cap():
			err = fmt.Errorf("%d value%s specified, at most %d expected", count, plural(count), v.Cap())
		}
		if err == nil {
			if count > v.Len() {
				// grow the slice
				s := reflect.MakeSlice(v.Type(), count, count)
				// no need to copy since no way to skip over existing values
				// do it anyway in just in case (e.g. extract values by splitting and skipping)
				reflect.Copy(s, v)
				v.Set(s)
			}
			// scan all values
			for i, value := range values {
				if err = scanfunc(value, reflElementAddr(i, v)); err != nil {
					break
				}
			}
		}
	default:
		// if too many values specified, the last wins
		err = scanfunc(values[len(values)-1], p.target)
	}
	if err != nil {
		err = decorate(err, p.name)
	}
	return err
}

// splitValues splits values around a splitter regular expression.
// It returns the input if the parameter has no splitter.
// When there is no splitter, multiple values can only be appended
// and there is no way to remove values already specified.
// When there is a splitter, the last value wins.
//
// Example
// 	a=1 a=2 a=3	a has 3 values: [1 2 3] (no splitter)
// 	a=1:2:3		a has 3 values: [1 2 3] (splits on :)
//	a=1:2:3 a=4:5	a has 2 values: [4 5] (splits on :)
func splitValues(p *Param, values []string) []string {
	if p.splitter == nil {
		return values
	}
	// repeated values are appended but with a splitter the last wins
	var splitted []string
	for _, value := range values {
		splitted = p.splitter.Split(value, -1)
	}
	return splitted
}

// scanFunc returns the scan function configured in Param if any else the
// builtin Scan function.
func scanFunc(p Param) func(string, interface{}) error {
	if p.scan != nil {
		return p.scan
	}
	return Scan
}

// verifyNotSeen is a help for Parse.
// It verifies that omitted parameters can be omitted and that
// default values of omitted parameters are valid.
func (a *Parser) verifyNotSeen(seen map[string]int) error {
	for n, count := range seen {
		p := a.params[n]
		value := reflValue(p.target)
		switch value.Kind() {
		case reflect.Slice:
			for i := count; i < reflLen(p.target); i++ {
				// scan remaining initial values to ensure they are okay
				if e := scanFunc(*p)(fmt.Sprint(reflElement(i, value)), reflCopy(reflElementAddr(i, value))); e != nil {
					return decorate(fmt.Errorf("invalid default value at offset %d: %v", i, e), n)
				}
			}
		case reflect.Array:
			if count != reflLen(p.target) {
				return decorate(fmt.Errorf("%d value%s expected but only %d specified", reflLen(p.target), plural(reflLen(p.target)), count), n)
			}
		default:
			// single-valued parameter
			if count < 1 {
				if !p.opt {
					return decorate(fmt.Errorf("mandatory parameter not set"), n)
				}
				// scan initial value (into a copy) to ensure it's okay
				if e := scanFunc(*p)(fmt.Sprint(value), reflCopy(p.target)); e != nil {
					return decorate(fmt.Errorf("invalid default value: %v", e), n)
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

// decorate improves error messages.
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

// reflValue returns value using reflection
func reflValue(target interface{}) reflect.Value {
	return reflect.Indirect(reflect.ValueOf(target))
}

// reflCopy returns new copy using reflection
func reflCopy(target interface{}) interface{} {
	return reflect.New(reflect.TypeOf(target).Elem()).Interface()
}

// reflElementAddr returns address of i-th element using reflection
func reflElementAddr(i int, v reflect.Value) interface{} {
	return v.Index(i).Addr().Interface()
}

// reflElement returns i-th element using reflection
func reflElement(i int, v reflect.Value) interface{} {
	return v.Index(i).Interface()
}
