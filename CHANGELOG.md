# CHANGELOG

### Unreleased

* Version 1.0.0

### v0.6.2 (2017-12-26)

* Handle byte order mark gracefully when including files.

### v0.6.1 (2017-12-26)

* Rename most files and move around some functions and methods.

### v0.6.0 (2017-12-26)

* Require symbol prefix to be always escaped when followed by special character.
  This change is INCOMPATIBLE from the point of view of users.
* Documentation fixes.
* More tests.
* Refactoring.

### v0.5.0 (2017-12-24)

The input syntax for symbol references has been modified. A symbol reference is
now written $[foo]. Previously it was $$foo or $$$foo$. The release is
compatible from the point of view of programs but is INCOMPATIBLE from the point
of view of users. Symbols can be escaped. The implementation has been heavily
refactored (simplified) and the input is now interpreted in a single pass.

### v0.4.1 (2017-12-21)

New method Parser.ParseBytes. Reduces the number of conversions when including
files.

### v0.4.0 (2017-12-20)

This release includes INCOMPATIBLE changes. It supports full customization of
special characters and operator names.

* NewParser function does not take a parameter any more.
* Type Specials was removed.
* New type Config.
* New function CustomParser uses Config.
* New function SubParser.

### v0.3.1 (2017-12-19)

Documentation fixes.

### v0.3.0 (2017-12-19)

All modifications are backward compatible.

* Parser.PrintDoc can take zero or more arguments to format on line 1.
* Parser.PrintDoc provides default command doc texts.
* New Parser.PrintConfig prints the parser's configuration, which was done
  previously by PrintDoc, but is not always wanted.
* New operator "macro". There are now 7 built-in operators.
* All operators are now documented in the general package documentation.
* The README file has been updated and points to the package documentation at
  godoc.org.

### v0.2.2 (2017-12-18)

Modify operators (cond, dump, import, include, reset) to take also parameter
names, not only symbols. (jp)

### v0.2.1 (2017-12-16)

Implement key selection and custom regular expression in include operator.

### v0.2.0 (2017-12-16)

Add built-in operators:

* cond (parses on condition)
* dump (helps debugging)
* import (imports environment variables as symbols)
* include (includes a file)
* reset (resets symbols)
* -- (skips value, kind of comment)

### v0.1.2 (2017-12-15)

Bug fix. Ensure that the set of characters allowed in parameter and symbol names
and the set of characters allowed as special characters are disjoint.

### v0.1.1 (2017-12-14)

Initial release (jp)
