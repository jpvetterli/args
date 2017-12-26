package args

import (
	"fmt"
	"reflect"
	"strconv"
)

// typescan converts the value to the type pointed to by the target. The target must
// be a pointer to one of the basic types supported by Parse* functions in the
// strconv package.
func typescan(value string, target interface{}) error {
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
