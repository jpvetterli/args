# CHANGELOG

### Unreleased

* INCOMPATIBLE CHANGE COMING SOON: NewParser will not take an argument any more

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
