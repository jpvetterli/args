package args

import (
	"fmt"
	"testing"
)

func TestBadTarget(t *testing.T) {
	i := 42
	err := convertValue("43", i)
	expected := `target for value "43" is not a pointer`
	if err == nil {
		t.Errorf("error missing")
	} else if err.Error() != expected {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestType(t *testing.T) {
	var i int
	var i8 int8
	var i16 int16
	var i32 int32
	var i64 int64
	var ui uint
	var ui8 uint8
	var ui16 uint16
	var ui32 uint32
	var ui64 uint64
	var f32 float32
	var f64 float64

	count := 0

	test := func(input string, i interface{}) {
		count++
		err := convertValue(input, i)
		if err != nil {
			t.Errorf("unexpected error in test %d: %v", count, err)
		}
		s := fmt.Sprintf("%v", reflValue(i))
		if s != input {
			t.Errorf(`difference in test %d: %s != %s`, count, input, s)
		}
	}
	test("-1", &i)
	test("-1", &i8)
	test("-1", &i16)
	test("-1", &i32)
	test("-1", &i64)
	test("1", &ui)
	test("1", &ui8)
	test("1", &ui16)
	test("1", &ui32)
	test("1", &ui64)
	test("1.5", &f32)
	test("1.5", &f64)
}

// math.MaxInt64 is 9223372036854775807
// math.MaxFloat64 is 1.7976931348623157e+308

func TestTypeError(t *testing.T) {
	var i int
	var i8 int8
	var i16 int16
	var i32 int32
	var i64 int64
	var ui uint
	var ui8 uint8
	var ui16 uint16
	var ui32 uint32
	var ui64 uint64
	var f32 float32
	var f64 float64

	count := 0

	test := func(input string, i interface{}) {
		count++
		err := convertValue(input, i)
		if err == nil {
			t.Errorf("error missing in test %d", count)
		}
	}
	test("abc", &i)
	test("9223372036854775807", &i8)
	test("9223372036854775807", &i16)
	test("9223372036854775807", &i32)
	test("abc", &i64)
	test("-1", &ui)
	test("-1", &ui8)
	test("-1", &ui16)
	test("-1", &ui32)
	test("-1", &ui64)
	test("1.7976931348623157e+308", &f32)
	test("true", &f64)
}
