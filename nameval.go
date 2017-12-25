package args

import (
	"fmt"
)

// name-value parser manages the tokenizer and remembers one name read to far.
type nameValParser struct {
	t    tokenizer
	name *symval // nil means next is a name
}

// newNameValParser returns a new name-value parser
func newNameValParser(p *Parser, input []byte) nameValParser {
	tkz := newTokenizer(p.config, &p.symbols)
	tkz.Reset(input)
	return nameValParser{t: *tkz}
}

// next returns a name symval, a value symval, and an error. The name can be
// nil. All nil indicate the end of the input. When the method returns a non nil
// error, name and value are nil.
func (nvp *nameValParser) next() (*symval, *symval, error) {

	var name *symval

	if nvp.name == nil {
		token, s, err := nvp.t.Next()
		if token == tokenError {
			return nil, nil, err
		}

		switch token {
		case tokenEnd:
			return nil, nil, nil
		case tokenEqual:
			return nil, nil, fmt.Errorf(`at "%s": "%c" unexpected`, nvp.t.ErrorContext(), nvp.t.config.GetSpecial(SpecSeparator))
		case tokenString:
			name = s
		}
	} else {
		name, nvp.name = nvp.name, nil
	}

	// got a name so far, if next token is a string, it's in fact a standalone value
	token, s, err := nvp.t.Next()
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

	token, s, err = nvp.t.Next()
	if token == tokenError {
		return nil, nil, decorate(err, name.s)
	}
	switch token {
	case tokenEnd:
		return nil, nil, fmt.Errorf(`at "%s": premature end of input`, nvp.t.ErrorContext())
	case tokenEqual:
		return nil, nil, fmt.Errorf(`at "%s": "%c" unexpected`, nvp.t.ErrorContext(), nvp.t.config.GetSpecial(SpecSeparator))
	case tokenString:
		return name, s, nil
	}
	panic("unreachable")
}
