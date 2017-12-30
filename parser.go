package args

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"
)

// Parser methods define and parse command line parameters. There are also
// methods for specifying and producing command documentation.
type Parser struct {
	config  *Config
	params  map[string]*Param
	seq     []string // names in definition sequence
	doc     []string
	targets map[interface{}]bool // duplicate detection
	symbols symtab
	cycle   map[string]bool // include cycle detector
}

// CustomParser returns a new Parser with a specific configuration. Because the
// parser keeps a copy of the configuration and not the original, changes to the
// configuration has only an effect before calling this function, but not after.
func CustomParser(configuration *Config) *Parser {
	copy := configuration.copy()
	return &Parser{
		config:  copy,
		params:  make(map[string]*Param),
		seq:     make([]string, 0),
		doc:     make([]string, 0),
		targets: make(map[interface{}]bool),
		symbols: newSymtab(copy),
		cycle:   make(map[string]bool),
	}
}

// SubParser returns a new Parser configured like the parser specified.
func SubParser(parser *Parser) *Parser {
	return CustomParser(parser.config)
}

// NewParser returns a new Parser with a default configuration.
func NewParser() *Parser {
	return CustomParser(NewConfig())
}

// Def defines a parameter with a name and a target to take one or more values.
// It returns a Param which can be used to optionally configure various details.
// This is designed to allow chaining of methods, so that a complete parameter
// definition can be made with a one-liner.
//
// Example of chaining:
//    var help bool
//    a.Def("-help", &help).Aka("-h").Opt().Doc("print a usage summary and exit")
//
// When target points to an array, the parameter takes a number of values
// exactly equal to its length. When target points to a slice, it takes a number
// of values not exceeding its capacity, unless the capacity is zero, which is
// interpreted as no limit.
//
// When target points to a map, it takes an arbitrary number of key-value pairs
// separated by the same special character used as separator between parameter
// names and values. When the parameter is anonymous, the outer brackets around
// groups of key-value pairs can be omitted. This makes key-value pairs look
// very much like parameter names and values in this case, except that keys have
// not been defined. However, defined parameters take precedence over key-value
// pairs.
//
// Def is the only Parser method which panics when it detects an error. It
// panics if the name is already used, if the name contains a character other
// than a letter, a digit, a hyphen or an underscore, if the target is not a
// pointer, or if the target is already assigned to another parameter.
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
	if err := validate(name); err != nil {
		panic(err)
	}
	if a.operator(name) != nil {
		panic(fmt.Errorf(`parameter name "%s" is the name of an operator`, name))
	}
	p := Param{parser: a, name: name, target: target}

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

// ParseBytes parses b to extract and assign values to parameter targets.  The
// result is nil unless there is an error.  The input syntax is explained in the
// package documentation.
func (a *Parser) ParseBytes(b []byte) error {
	err := a.parse(b)
	if err != nil {
		return err
	}
	return a.verify()
}

// Parse calls ParseBytes with s converted to a byte slice.
func (a *Parser) Parse(s string) error {
	return a.ParseBytes([]byte(s))
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
		case reflect.Map:
			details = ""
			if value.Len() > 0 {
				details += fmt.Sprintf(" (default: %v)", value)
			}
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
		fmt.Fprintf(w, "\nSpecial characters:\n")
		for _, s := range [5]specConstant{SpecSymbolPrefix, SpecOpenQuote, SpecCloseQuote, SpecSeparator, SpecEscape} {
			fmt.Fprintf(w, "  %c        %s\n", a.config.GetSpecial(s), specialDescription[s])
		}
		fmt.Fprintf(w, "\nBuilt-in operators:\n")

		reverse := make(map[opConstant]string, len(a.config.opDict))
		for n, v := range a.config.opDict {
			reverse[v] = n
		}
		text := make(map[opConstant]string, len(a.config.opDict))
		text[OpCond] = "conditional parsing (if, then, else)"
		text[OpDump] = "print parameters and symbols on standard error (comment)"
		text[OpImport] = "import environment variables as symbols"
		text[OpInclude] = "include a file or extract name-values (keys, extractor)"
		text[OpMacro] = "expand symbols"
		text[OpReset] = "remove symbols"
		text[OpSkip] = "do not parse the value (= comment out)"

		print := func(name, doc string) {
			if len(name) > 8 {
				fmt.Fprintf(w, "  %s\n", name)
				fmt.Fprintf(w, "  %-8s %s\n", "", doc)
			} else {
				fmt.Fprintf(w, "  %-8s %s\n", name, doc)
			}
		}
		print(reverse[OpCond], text[OpCond])
		print(reverse[OpDump], text[OpDump])
		print(reverse[OpImport], text[OpImport])
		print(reverse[OpInclude], text[OpInclude])
		print(reverse[OpMacro], text[OpMacro])
		print(reverse[OpReset], text[OpReset])
		print(reverse[OpSkip], text[OpSkip])
	}
}

// helpers

// parse parses b. It can be used recursively.
func (a *Parser) parse(b []byte) error {
	nvp := newNameValParser(a, b)
	var name, value *symval
	var err error
	for {
		name, value, err = nvp.next()

		if err != nil {
			return err
		}

		if name == nil && value == nil {
			break
		}

		if name == nil {
			// standalone name or value
			if a.isStandaloneBoolParameter(value) {
				// standalone name
				name, value = value, &symval{resolved: true, s: "true"}
			} else {
				// standalone value
				if _, ok := a.params[""]; !ok {
					return fmt.Errorf(`unexpected standalone value: "%s"`, value.s)
				}
				// the famous empty name
				name = &symval{resolved: true, s: ""}
			}
		}

		// assert name != nil && value != nil

		operator := a.operator(name.s)
		if operator != nil {
			err := operator.handle(value.s)
			if err != nil {
				return err
			}
		} else {
			err := a.setValue(name, value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Parser) isStandaloneBoolParameter(value *symval) bool {
	if value.resolved {
		if p, ok := a.params[value.s]; ok {
			return reflTakesBool(p.target)
		}
	}
	return false
}

// getAnonymousMapParameter returns *Param of anonymous map if defined, else nil
func (a *Parser) getAnonymousMapParameter() *Param {
	if p, ok := a.params[""]; ok {
		if reflValue(p.target).Kind() == reflect.Map {
			return p
		}
	}
	return nil
}

// setValue adds value to symbol table or sets parameter value
func (a *Parser) setValue(name, value *symval) error {

	if !name.resolved {
		return fmt.Errorf(`cannot resolve name in "%s %c %s"`, name.s, a.config.GetSpecial(SpecSeparator), value.s)
	}

	if !a.symbols.put(name.s, value.s) {
		if p, ok := a.params[name.s]; ok {

			if !value.resolved {
				if !p.verbatim {
					if len(name.s) == 0 {
						return fmt.Errorf(`cannot resolve standalone value "%s"`, value.s)
					}
					return fmt.Errorf(`cannot resolve value in "%s %c %s"`, name.s, a.config.GetSpecial(SpecSeparator), value.s)
				}
			}

			err := p.parseValues(p.split(value.s))

			if err != nil {
				return err
			}
		} else {
			if p := a.getAnonymousMapParameter(); p != nil {
				return convertKeyValue(name.s, value.s, p.target)
			}
			return fmt.Errorf(`parameter not defined: "%s"`, name.s)
		}
	}
	return nil
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

// name-value parsing

// name-value parser manages the tokenizer and remembers one name read to far.
type nameValParser struct {
	t    tokenizer
	name *symval // nil means next is a name
}

// newNameValParser returns a new name-value parser
func newNameValParser(p *Parser, input []byte) nameValParser {
	tkz := newTokenizer(p.config, &p.symbols)
	tkz.reset(input)
	return nameValParser{t: *tkz}
}

// next returns a name symval, a value symval, and an error. The name can be
// nil. The value cannot be nil when the name is not nil. All results nil
// indicate the end of the input. When the method returns a non nil error, name
// and value are nil.
func (nvp *nameValParser) next() (*symval, *symval, error) {

	var name *symval

	if nvp.name == nil {
		token, s, err := nvp.t.next()
		if token == tokenError {
			return nil, nil, err
		}

		switch token {
		case tokenEnd:
			return nil, nil, nil
		case tokenEqual:
			return nil, nil, fmt.Errorf(`at "%s": "%c" unexpected`, nvp.t.errorContext(), nvp.t.config.GetSpecial(SpecSeparator))
		case tokenString:
			name = s
		}
	} else {
		name, nvp.name = nvp.name, nil
	}

	// got a name so far, if next token is a string, it's in fact a standalone value
	token, s, err := nvp.t.next()
	if token == tokenError {
		return nil, nil, decorate(err, name.s)
	}

	switch token {
	case tokenEnd:
		// single string: by convention put it in the value
		return nil, name, nil
	case tokenEqual:
		// expect now a value
	case tokenString:
		// standalone value, swap and save
		nvp.name = s
		return nil, name, nil
	}

	// after name and separator, expect value (string)

	token, s, err = nvp.t.next()
	if token == tokenError {
		return nil, nil, decorate(err, name.s)
	}
	switch token {
	case tokenEnd:
		return nil, nil, fmt.Errorf(`at "%s": premature end of input`, nvp.t.errorContext())
	case tokenEqual:
		return nil, nil, fmt.Errorf(`at "%s": "%c" unexpected`, nvp.t.errorContext(), nvp.t.config.GetSpecial(SpecSeparator))
	case tokenString:
		return name, s, nil
	}
	panic("unreachable")
}

// helpers

// buildSynonyms makes synonym map for PrintDoc.
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

// validSpecial returns true iff char is valid as a special character.
// Valid special characters are graphic, not white space, not valid in a name.
func validSpecial(char rune) bool {
	return !valid(char) && unicode.IsGraphic(char) && !unicode.IsSpace(char)
}

// validate verifies a name
func validate(name string) error {
	for _, r := range []rune(name) {
		if !valid(r) {
			return fmt.Errorf(`"%s" cannot be used as a name because it includes the character '%c'`, name, r)
		}
	}
	return nil
}

// valid returns true iff char is valid in a parameter or symbol name.
// Valid characters are letters, digits, the hyphen and the underscore.
func valid(char rune) bool {
	return unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_'
}
