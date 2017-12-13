package args

import (
	"fmt"
)

// nameValue holds either a name-value pair or a standalone value.
// For a standalone value, Name is an empty string. An empty
// Value string is a value like any other.
type nameValue struct {
	Name  string
	Value string
}

type expectState uint8

const (
	expectName  expectState = iota // expect a name or a value
	expectEqual                    // after seeing a name or a value
	expectValue                    // after seeing an equal
)

// pairs returns a list of name-value pairs and standalone values found in the
// input, using  the given configuration of special characters.
func pairs(config *Specials, input []byte) ([]*nameValue, error) {
	result := make([]*nameValue, 0, 20)
	t := newTokenizer(config)
	t.Reset(input)
	state := expectName
	var p *nameValue
	for {
		token, s, err := t.Next()
		if token == tokenError {
			return nil, err
		}
		switch state {

		case expectName:
			switch token {
			case tokenEnd:
				return result, nil
			case tokenEqual:
				return nil, fmt.Errorf(`at "%s": "%c" unexpected`, t.ErrorContext(), config.Separator())
			case tokenString:
				// assume new token is a name
				p = new(nameValue)
				result = append(result, p)
				p.Name = s // so far, could be name-value
				state = expectEqual
			}

		case expectEqual:
			switch token {
			case tokenEnd:
				p.Name, p.Value = p.Value, p.Name // p.value was nil
				return result, nil
			case tokenEqual:
				state = expectValue
			case tokenString:
				// so, the current name is a value, swap
				p.Name, p.Value = p.Value, p.Name
				// and assume new token is a name
				p = new(nameValue)
				result = append(result, p)
				p.Name = s
				state = expectEqual
			}

		case expectValue:
			switch token {
			case tokenEnd:
				return nil, fmt.Errorf(`at "%s": premature end of input`, t.ErrorContext())
			case tokenEqual:
				return nil, fmt.Errorf(`at "%s": "%c" unexpected`, t.ErrorContext(), config.Separator())
			case tokenString:
				p.Value = s
				state = expectName
			}
		}
	}
}

// values returns a list of standalone values, using the given configuration of
// special characters. An error is returned if the input contains any name-value
// pair.
func values(config *Specials, input []byte) ([]string, error) {
	result := make([]string, 0, 20)
	t := newTokenizer(config)
	t.Reset(input)
	for {
		token, s, err := t.Next()
		if token == tokenError {
			return nil, err
		}
		switch token {
		case tokenString:
			result = append(result, s)
		case tokenEnd:
			return result, nil
		default:
			return nil, fmt.Errorf(`at "%s": the input must contain only values`, t.ErrorContext())
		}
	}
}
