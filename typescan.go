package args

import (
	"fmt"
	"reflect"
	"strconv"
)

// Scan converts the value to the type pointed to by the target. The target must
// be a pointer to one of the basic types supported by Parse* functions in the
// strconv package.
func Scan(value string, target interface{}) error {
	if reflect.ValueOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf(`target for value "%s" is not a pointer`, value)
	}
	var (
		b   bool
		i   int64
		u   uint64
		f   float64
		err error
	)
	v := reflValue(target)
	switch v.Kind() {
	case reflect.String:
		v.SetString(value)
	case reflect.Bool:
		if b, err = strconv.ParseBool(value); err == nil {
			v.SetBool(b)
		}
	case reflect.Int:
		if i, err = strconv.ParseInt(value, 0, 0); err == nil {
			v.SetInt(i)
		}
	case reflect.Int8:
		if i, err = strconv.ParseInt(value, 0, 8); err == nil {
			v.SetInt(i)
		}
	case reflect.Int16:
		if i, err = strconv.ParseInt(value, 0, 16); err == nil {
			v.SetInt(i)
		}
	case reflect.Int32:
		if i, err = strconv.ParseInt(value, 0, 32); err == nil {
			v.SetInt(i)
		}
	case reflect.Int64:
		if i, err = strconv.ParseInt(value, 0, 64); err == nil {
			v.SetInt(i)
		}
	case reflect.Uint:
		if u, err = strconv.ParseUint(value, 0, 0); err == nil {
			v.SetUint(u)
		}
	case reflect.Uint8:
		if u, err = strconv.ParseUint(value, 0, 8); err == nil {
			v.SetUint(u)
		}
	case reflect.Uint16:
		if u, err = strconv.ParseUint(value, 0, 16); err == nil {
			v.SetUint(u)
		}
	case reflect.Uint32:
		if u, err = strconv.ParseUint(value, 0, 32); err == nil {
			v.SetUint(u)
		}
	case reflect.Uint64:
		if u, err = strconv.ParseUint(value, 0, 64); err == nil {
			v.SetUint(u)
		}
	case reflect.Float32:
		if f, err = strconv.ParseFloat(value, 32); err == nil {
			v.SetFloat(f)
		}
	case reflect.Float64:
		if f, err = strconv.ParseFloat(value, 64); err == nil {
			v.SetFloat(f)
		}
	default:
		err = fmt.Errorf(`target for value "%s" has unsupported type %v`, value, v.Type())
	}
	return err
}
