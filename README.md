# args

Define and parse command line arguments.

[![GoDoc](https://godoc.org/github.com/jpvetterli/args?status.svg)](https://godoc.org/github.com/jpvetterli/args)
[![Build Status](https://travis-ci.org/jpvetterli/args.svg?branch=master)](https://travis-ci.org/jpvetterli/args)
[![Coverage Status](https://coveralls.io/repos/github/jpvetterli/args/badge.svg)](https://coveralls.io/github/jpvetterli/args)

The package keeps simple things simple and makes complicated things possible.
For example, this is a simple program argument:
```
  help
```
and this is something more complicated:
```
  include=[
    /home/u649/.db.conf
    extractor=[\s*"(\S+)"\s*:\s*"(\S+)"\s*]
    keys=[user=usr password=$PASS]
  ]
```
With package args, both examples are handled with minimal application logic.

Parameters are defined in one-liners, with synonyms, documentation, and other
configuration details. Values are directly taken into simple variables, arrays
or slices. Arbitrary types are easily supported like in this example:

```
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

// define parameter "time", use custom scanner
a.Def("time", &target).Scan(scanner).Aka("datetime").Doc(`specify time in the format "yyyy-mm-dd hh:mm:ss"`)

// parse an input
a.Parse("time=[2015-06-30 15:32:00]")

// voilà! the value is in the variable
fmt.Println(target.String()) // prints: 2015-06-30 15:32:00 +0000 UTC
```

More information and examples are available from the
[package documentation](https://godoc.org/github.com/jpvetterli/args).
