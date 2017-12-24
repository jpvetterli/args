package args_test

import (
	"fmt"
	"os"
	"time"

	"github.com/jpvetterli/args"
)

func ExampleParser_PrintDoc() {
	a := args.NewParser()
	a.Doc(
		"Usage: %s parameter...\n",
		"The command does nothing but is has many parameters.",
		"",
		"Parameters:")

	var files []string
	var help bool
	var short string
	long := []string{"a1", "b2"}
	sl := make([]int, 0)
	var ar [4]float64

	a.Def("", &files).Aka("file").Doc("foo takes any number of file names")
	a.Def("help", &help).Aka("-h").Doc("provide help").Opt()
	a.Def("short", &short).Opt().Doc("short is a parameter with a short name")
	a.Def("long-name", &long).Doc(
		"long-name is a parameter with a name longer than 8",
		"It also has a long explanation.")
	a.Def("slice", &sl).Doc("slice is a parameter taking any number of values")
	a.Def("array", &ar).Doc("array is a parameter taking 4 values").Split(`\s*:\s*`)

	a.PrintDoc(os.Stdout, "foo")
	a.PrintConfig(os.Stdout)

	// output:
	// Usage: foo parameter...
	// The command does nothing but is has many parameters.
	//
	// Parameters:
	//   (nameless), file
	//            foo takes any number of file names
	//            type: string, any number of values
	//   help, -h provide help
	//            type: bool, optional (default: false)
	//   short    short is a parameter with a short name
	//            type: string, optional (default: )
	//   long-name
	//            long-name is a parameter with a name longer than 8
	//            It also has a long explanation.
	//            type: string, 0-2 values (default: [a1 b2])
	//   slice    slice is a parameter taking any number of values
	//            type: int, any number of values
	//   array    array is a parameter taking 4 values
	//            type: float64, split: \s*:\s*, exactly 4 values
	//
	// Special characters:
	//   $        symbol prefix
	//   [        open quote
	//   ]        close quote
	//   =        separator
	//   \        escape
	//
	// Built-in operators:
	//   cond     conditional parsing (if, then, else)
	//   dump     print parameters and symbols on standard error (comment)
	//   import   import environment variables as symbols
	//   include  include a file or extract name-values (keys, extractor)
	//   macro    expand symbols
	//   reset    remove symbols
	//   --       do not parse the value (= comment out)
}

func ExampleParam_Scan() {
	a := args.NewParser()
	var target time.Time

	scanner := func(value string, target interface{}) error {
		if s, ok := target.(*time.Time); ok {
			if t, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
				*s = t
			} else {
				return err
			}
			return nil
		}
		return fmt.Errorf(`time scanner error: "%s", *time.Time target required, not %T`, value, target)
	}

	a.Def("time", &target).Scan(scanner).Aka("datetime").Doc(`specify time in the format "yyyy-mm-dd hh:mm:ss"`)

	a.Parse("time=[2015-06-30 15:32:00]")

	fmt.Println(target.String())
	// output:
	// 2015-06-30 15:32:00 +0000 UTC
}

func ExampleParser_Parse() {
	a := args.NewParser()
	var s string
	var f [3]float64

	a.Def("foo", &s)
	a.Def("bar", &f).Split(":")

	err := a.Parse("foo=bar bar=1:2:3:4")
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("Oops... let's try again")
		err = a.Parse("foo=quux bar=1:2:3")
		if err != nil {
			fmt.Println(err.Error())
		}

		fmt.Println(s)
		fmt.Println(f)

	}

	// output:
	// Parse error on bar: too many values specified, expected 3
	// Oops... let's try again
	// quux
	// [1 2 3]
}

func ExampleParam_Verbatim() {
	a := args.NewParser()
	cmd1 := ""
	cmd2 := ""
	a.Def("cmd1", &cmd1).Verbatim() // IMPORTANT: verbatim
	a.Def("cmd2", &cmd2).Verbatim()
	a.Parse("$MACRO = [arg1=$[ARG1] arg2=$[ARG2]] " +
		"cmd1=[$ARG1=x $ARG2=y $[MACRO]] " +
		"cmd2=[$ARG1=a $ARG2=b $[MACRO]]")
	for _, s := range []string{cmd1, cmd2} {
		a = args.NewParser() // IMPORTANT: get a new parser
		arg1 := ""
		arg2 := ""
		a.Def("arg1", &arg1)
		a.Def("arg2", &arg2)
		a.Parse(s)
		fmt.Println(arg1, arg2)
	}
	// output:
	// x y
	// a b
}
