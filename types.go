package args

import (
	"fmt"
	"reflect"
	"strconv"
)

// convertValue converts value to the type at target and assigns the converted
// value to the variable at target.
func convertValue(value string, target interface{}) error {
	if reflect.ValueOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf(`target for value "%s" is not a pointer`, value)
	}
	v := reflValue(target)
	parsed, err := convert(value, v.Type())
	if err != nil {
		return err
	}
	v.Set(reflect.ValueOf(parsed))
	return nil
}

// convertKeyValue converts and sets key and value of a map target.
func convertKeyValue(key, value string, target interface{}) error {
	if reflect.ValueOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf(`target for key "%s" and value "%s" is not a pointer`, key, value)
	}
	targetValue := reflValue(target)
	t := targetValue.Type()
	keyType := t.Key()
	valType := t.Elem()
	k, err := convert(key, keyType)
	if err != nil {
		return fmt.Errorf(`key cannot be converted: %v`, err)
	}
	v, err := convert(value, valType)
	if err != nil {
		return fmt.Errorf(`value for key "%s" cannot be converted: %v`, key, err)
	}
	targetValue.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
	return nil
}

// convert converts value to the type of target and returns the converted value
// as an empty interface. The type of the target must one of the basic types
// supported by Parse* functions in the strconv package.
func convert(value string, typ reflect.Type) (interface{}, error) {
	var err error
	var parsed interface{}
	switch typ.Kind() {
	case reflect.String:
		parsed = value
	case reflect.Bool:
		parsed, err = strconv.ParseBool(value)
	case reflect.Int:
		parsed, err = strconv.ParseInt(value, 0, 0)
		parsed = int(parsed.(int64))
	case reflect.Int8:
		parsed, err = strconv.ParseInt(value, 0, 8)
		parsed = int8(parsed.(int64))
	case reflect.Int16:
		parsed, err = strconv.ParseInt(value, 0, 16)
		parsed = int16(parsed.(int64))
	case reflect.Int32:
		parsed, err = strconv.ParseInt(value, 0, 32)
		parsed = int32(parsed.(int64))
	case reflect.Int64:
		parsed, err = strconv.ParseInt(value, 0, 64)
	case reflect.Uint:
		parsed, err = strconv.ParseUint(value, 0, 0)
		parsed = uint(parsed.(uint64))
	case reflect.Uint8:
		parsed, err = strconv.ParseUint(value, 0, 8)
		parsed = uint8(parsed.(uint64))
	case reflect.Uint16:
		parsed, err = strconv.ParseUint(value, 0, 16)
		parsed = uint16(parsed.(uint64))
	case reflect.Uint32:
		parsed, err = strconv.ParseUint(value, 0, 32)
		parsed = uint32(parsed.(uint64))
	case reflect.Uint64:
		parsed, err = strconv.ParseUint(value, 0, 64)
	case reflect.Float32:
		parsed, err = strconv.ParseFloat(value, 32)
		parsed = float32(parsed.(float64))
	case reflect.Float64:
		parsed, err = strconv.ParseFloat(value, 64)
	default:
		parsed = nil
		err = fmt.Errorf(`type %v requested for value "%s" is not supported`, typ, value)
	}
	return parsed, err
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
