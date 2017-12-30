package args_test

import "testing"

func TestParamDuplicate(t *testing.T) {
	a := getParser()
	defer panicHandler(`parameter "a" already defined`, t)
	i := 1
	a.Def("a", &i)
	a.Def("a", &i)
}

func TestParamDuplicateAlias(t *testing.T) {
	a := getParser()
	defer panicHandler(`synonym "A" clashes with an existing parameter name or synonym`, t)
	i := 1
	s := ""
	a.Def("a", &i).Aka("A")
	a.Def("b", &s).Aka("A")
}

func TestParamDuplicateTarget(t *testing.T) {
	a := getParser()
	defer panicHandler(`target for parameter "b" is already assigned`, t)
	i := 1
	a.Def("a", &i)
	a.Def("b", &i)
}

func TestParamNotPointer(t *testing.T) {
	a := getParser()
	defer panicHandler(`target for parameter "a" is not a pointer`, t)
	i := 1
	a.Def("a", i)
}

func TestParamSplit1(t *testing.T) {
	a := getParser()
	defer panicHandler(`cannot split values of "x" (only arrays and slices parameters can be split)`, t)
	var x uint8
	a.Def("x", &x).Split("foo")
}

func TestParamSplit2(t *testing.T) {
	a := getParser()
	defer panicHandler("compilation of split expression \"***\" for parameter \"x\" failed: error parsing regexp: missing argument to repetition operator: `*`", t)
	var x []uint8
	a.Def("x", &x).Split("***")
}

func TestParamOperator1(t *testing.T) {
	a := getParser()
	defer panicHandler(`parameter name "--" is the name of an operator`, t)
	i := 1
	a.Def("--", &i)
}

func TestParamOperator2(t *testing.T) {
	a := getParser()
	defer panicHandler(`parameter name "include" is the name of an operator`, t)
	i := 1
	a.Def("include", &i)
}
