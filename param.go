package args

import (
	"fmt"
	"reflect"
	"regexp"
)

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
	if err := validate(alias); err != nil {
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
