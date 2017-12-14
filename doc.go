/*

Package args is used to define and parse command line arguments. A program
usually gets its parameters from a string array supplied by the system. By a
strange coincidence, this array is named "args", like this package.

Parameters can be defined to take values for simple variables of builtin types
and for arrays and slices. If needed, custom scanners can be used, so
non-standard types are supported too. From the programmer's point of view,
configuring parameters and taking values looks like this:

  p := args.NewParser(nil)
  things []string
  p.Def("thing", &things).Aka("") // Aka defines the empty name as synonym
  p.Parse(...)
  // at this point, the things slice has the data

From the user's point of view, here are 3 different ways of specifying 3
different things:

  thing = foo thing=bar thing = [a 3d thing with "blanks" etc.]
  foo bar [a 3d thing with "blanks" etc.]
  foo []=bar thing = [a 3d thing with "blanks" etc.]

The programming interface is explained in the type and method documentation.
This introduction describes the package from the point of view of the user.

A Mini Language

Parameters are formulated using a mini-language where words belong either to a
name or to a value. Names and values must agree with parameter definitions made
in the program.

Syntactically, names and values are recognizable by the presence of a separator
between them. The separator is one of five specially designated characters
supporting the language syntax, in addition to white space: the symbol prefix,
the name-value separator, the opening quote, the closing quote, and the escape
character. Unless configured otherwise, these characters are:
  $ = [ ] \

White space is a sequence of one or more characters for which unicode.IsSpace
returns true. The escape character suppresses the effect of white space and
special characters, with the exception of the symbol prefix, which cannot be
escaped. When it has no effect, the escape is a normal character.

An empty name is usually omitted, and in this case the separator must also be
omitted. Parameters with an empty name are known as anonymous and their values
as standalone values. When a standalone value is the name of a parameter, it
is interpreted as that name with the value "true". This effect is usually what
is wanted, but if necessary, can be avoided using the []= notation.

White space around separators is ignored but is significant between words, as it
separates distinct values. White space can be included in names and values by
quoting. When a quoted string contains a nested quoted string, quotes must be
balanced.  Outermost quotes are removed but nested quotes are kept.

The following examples use the default set of special characters:

  foo = bar                (1 name with 1 value)
  foo=bar foo=baz foo=quux (1 name with 3 values)
  quux foo=bar   baz baf   (1 name with 1 value, 3 standalone values)
  foo = [ bar baz ]        (1 name with value " bar baz ")
  foo = [bar \]baz]        (1 name with value "bar ]baz")
  foo = [bar [baz]]        (1 name with value "bar [baz]", quotes nest)
  foo \= bar               (3 standalone values)
  foo                      (foo not defined as a name: 1 standalone value)
  foo                      (foo defined as a name: 1 name with value "true")
  [] = bar                 (1 standalone value, empty name explicit)
  $sym = bar               (definition of symbol sym with value "bar")
  $X=bar foo = [x: $$X ]   (name foo with value "x: bar ")
  $X = bar \$$X \\\= \[x:\ :x\]
                           (3 standalone values: "\bar", "\=", "[x: :x]", notice escaping behavior)

The order in which name-value pairs are specified is not significant. The
program will not know if parameter foo was specified before parameter bar.
Multiple values of parameters can be mixed in any order. On the other hand,
multiple parameter values are kept in specification sequence. Given "foo=1 bar=a
foo=2 bar=b bar=c foo=3", the program will see foo=[1 2 3] and var=[a b c] but
will not know if foo came before bar.

Symbols And Substitution

Parameter names are given by the program but symbols can be freely chosen. A
symbol is a string of letters or digits (as tested by unicode.IsLetter and
unicode.isDigit), hyphens or undersores and cannot contain a symbol prefix
character. A symbol is defined by prefixing it once with the symbol prefix. A
reference consists consists of a symbol preceded either by 2 prefixes (as in the
string "my $$foo is rich") or preceded by 3 prefixes and followed by 1 (as in
"$$$foo$bar"). The second notation is ugly, but useful when a symbol reference
is directly followed by a valid symbol character.

Parameters and symbols are in distinct name spaces, and it is possible to use
the name of a parameter for a symbol, as in "$foo=bar foo=$$foo" which is
equivalent to "foo=bar". Symbols must be defined before use: in "foo=$$x
$x=bar", the value of "foo" cannot be resolved. Redefining a symbol has no
effect:  the first wins (with parameters, it is the last that wins). The
specification "$x=bar $x=quux foo=$$x" is equivalent to "foo=bar".

Omission And Repetition

When a parameter is defined, it gets a target, which is the program variable
that will take values from the parameter. The target's type determines the
number and type of values the parameter can take. A target with a simple type
can take one value. If multiple values are specified, the last value wins
(contrast this with symbols, where the first wins). If no value is specified, an
error occurs, unless the parameter has been defined as optional. The initial
value of the target is the default value of an optional parameter.

If the target is an array type, the number of values specified must be exactly
equal to the array length. If the target is a slice type, the number of values
can be at most equal to the slice capacity. If the slice capacity is zero, the
parameter can take any number of values. A parameter with a slice target can be
omitted or it can take a number of values less than its current length. In such
cases the initial values in the slice provide the default values of the
parameter.

Instead of requiring repeating values, a parameter can be defined
with a splitter, which is a regular expression. For example, if a splitter
is defined to split a string around a colon (with optional white space)
the specification "foo=[1:2:3] foo=[ 4 : 5]" sets 5 values.

Macros

The parser behaves in a special way when it finds a standalone value which
contains symbol references. In such a case, symbol references are resolved, if
possible,  and if this modifies the value, it is parsed again recursively. This
behavior is comparable to "macro substitution", where a piece of text is defined
once and reused many times. Unfortunately it also makes some border cases
difficult to understand. For example the specification
  $X=Y $$X\\
cannot be parsed, because the recursive step sees `Y\` which is invalid.
The workaround is to specify
  $X=Y $$X\\\\
so the recursive step sees `Y\\`. An example using a macro is provided with
the Param.Verbatim method.

*/
package args
