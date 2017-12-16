# args

Define and parse command line arguments.

Parameters are defined in one-liners, with synonyms, documentation, and other configuration details. Values are directly taken into simple variables, arrays or slices. Arbitrary types are easily supported like in this example:

```
a := args.NewParser(nil)
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

Complete information and examples in the package documentation.