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

// regroup takes a list of name-value pairs (value singular) and returns a map
// and a list of name-values (plural).
//
// Each map and list value is a list of strings with at least 1 element, the
// canonical name of the parameter, followed by all values for this parameter,
// in specification order.  The list is ordered by first occurence of the name
// in the input.
//
// The map is keyed by canonical name. Canonical names are determined from the
// synonyms map. All names must be present in the map, possibly with a value
// equal to the key. A name missing from the map produces an error.
//
// A standalone value is found when a name-value pair has an empty name (an
// empty value is a value just like any other). Such values are assigned to the
// empty name, unless the value exists as a key in synonyms. In this case, the
// standalone value is interpreted as a standalone name. There can be only at
// most one occurence of a given standalone name. Otherwise an error ocurs.
// (Standalone names are provided to support implicitly true boolean
// parameters, also known as keywords.)
//
// When the function returns a non-nil error, the two other results are nil.
func regroup(pairs []*nameValue, synonyms map[string]string) (map[string][]string, [][]string, error) {
	// NOTE: length of map value/list element ([]string) is 1 <==> name is standalone

	m := make(map[string][]string)
	list := make([][]string, 0)
	for _, nv := range pairs {
		isStandalone, canonical := standaloneName(nv, synonyms)
		if isStandalone {
			// at most one standalone name allowed for a given name
			if v, ok := m[canonical]; ok {
				if len(v) == 1 {
					return nil, nil, fmt.Errorf(`standalone name %s cannot be repeated`, reportName(nv.Value, canonical))
				}
				return nil, nil, fmt.Errorf(`name %s can only be repeated with values, but not standalone`, reportName(nv.Value, canonical))
			}
			v := make([]string, 1)
			v[0] = canonical
			m[canonical] = v
			list = append(list, v)
		} else {
			// not a standalone name
			canonical, ok := synonyms[nv.Name]
			if !ok {
				if len(nv.Name) == 0 {
					return nil, nil, fmt.Errorf(`standalone value %q rejected (empty name not defined)`, nv.Value)
				}
				return nil, nil, fmt.Errorf(`name "%s" not defined`, nv.Name)
			}
			v, ok := m[canonical]
			if !ok {
				// name not seen yet
				v = make([]string, 2)
				v[0] = canonical
				v[1] = nv.Value
				m[canonical] = v
				list = append(list, v)
			} else {
				// name repeated, make sure it is not a standalone name
				if len(v) == 1 {
					return nil, nil, fmt.Errorf(`cannot add value "%s" to standalone name %s`, nv.Value, reportName(nv.Name, canonical))
				}
				m[canonical] = append(v, nv.Value)
			}
		}
	}
	// list values are the canonical only, update with final values from map
	for i, n := range list {
		list[i] = m[n[0]]
	}
	return m, list, nil
}

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
				// assume new token is a expectName
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

// standaloneName returns true and the canonical name if nv contains
// a standalone name
func standaloneName(nv *nameValue, synonyms map[string]string) (bool, string) {
	if len(nv.Name) == 0 {
		if canonical, ok := synonyms[nv.Value]; ok {
			return ok, canonical
		}
		return false, ""
	}
	return false, ""
}

// reportName generates string to report name clearly in error messages
func reportName(specified, canonical string) string {
	if specified == canonical {
		return "\"" + specified + "\""
	}
	return fmt.Sprintf(`"%s" (synonym of "%s")`, specified, canonical)
}
