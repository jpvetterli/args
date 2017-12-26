/*

Package args is used to configure programs using command line arguments and
other sources. A program usually gets its parameters from a string array
supplied by the system. This array is named "args" in many programming
languages, or something close. By an incredible coincidence, this is precisely
the name of this package.

The package keeps simple things simple and makes complicated things possible.
A first example shows how to provide help when the user says

  help

or maybe "-help", or "--help", or "-h":

    package main

    import (
    	"os"
    	"github.com/jpvetterli/args"
    )

    func main() {
    	a := args.NewParser()
    	help := false
    	a.Def("help", &help).Aka("-help").Aka("--help").Aka("-h").Opt()
    	err := a.ParseStrings(os.Args[1:])
    	if err != nil {
    		fmt.Fprintln(os.Stderr, err.Error()+" (try help)")
    		os.Exit(1)
    	}
    	if help {
    		a.PrintDoc(os.Stdout, os.Args[0])
    	}
    	os.Exit(0)
    }

Parameters can be defined to take values for simple variables of builtin types
and for arrays and slices. If needed, custom scanners can be used, so
non-standard types are supported too. From the programmer's point of view,
configuring parameters and taking values looks like this:

  p := args.NewParser()
  things := []string{}
  p.Def("thing", &things).Aka("") // empty name alias for "thing"
  p.Parse(...)
  // at this point, the things slice has the data

From the user's point of view, here are 3 different ways of specifying 3
different things:

  thing = foo thing=bar thing = [a 3d thing with "blanks" etc.]
  foo bar [a 3d thing with "blanks" etc.]
  foo []=bar thing = [a 3d thing with "blanks" etc.]

Now, writing all this repeatedly on the command line is tedious.
Fortunately the parser can take parameters from a file, like this:

  include = parameters.txt

Assuming the 3 lines above have been written verbatim to the file, there will 9
things, not 3. But the parser is capable of ignoring parameters, so let's ignore
the first two lines. The file could look like this (white space can be inserted
freely):

  --= [
      thing = foo thing=bar thing = [a 3d thing with "blanks" etc.]
      foo bar [a 3d thing with "blanks" etc.]
  ]
  foo []=bar thing = [a 3d thing with "blanks" etc.]

Symbols make it easy to override parameters in a file. Modify the file to use a
symbol for the 3d value:

  --= [
      thing = foo thing=bar thing = [a 3d thing with "blanks" etc.]
      foo bar [a 3d thing with "blanks" etc.]
  ]
  $DEFAULT3 = [a 3d thing with "blanks" etc.]
  foo []=bar thing = $[DEFAULT3]

The value can be overriden by setting the symbol DEFAULT3 before including the
file:

  $DEFAULT3 = [something else] include = parameters.txt

This works because the first value of a symbol wins. (For a scalar parameter it
is the last one.) The syntax uses special characters, like $, [, and ] in these
examples. In some environments, the predefined special characters can be
problematic, but they can be reconfigured easily by the program, and the above
input could look like this:

  ⌘DEFAULT3: «something else» include: parameters.txt

In the args package, everything is a name-value pair: parameters, operators like
include and -- (yes, it's a name). To keep simple things simple, it is possible
to omit the name when defined as empty, the parameter is of type bool, and
the value is true. This is the case of the "help" parameter in first example.
So, to get help, the user does not need to write "help=true" but simply

  help

When no help is wanted, it is not necessary to say "help = false". The reason
is that the initial value of the parameter is false.

The programming interface is explained in the type and method documentation. The
following sections describe the package from the point of view of the user.

A Mini Language

Parameters are formulated using a mini-language where words belong either to a
name or to a value. Names and values must agree with parameter definitions made
in the program. Names are composed of letters, digits (as tested by
unicode.IsLetter and IsDigit), hyphens, or underscores.

Syntactically, names and values are recognizable by the presence of a separator
between them. The separator is one of five specially designated characters
supporting the language syntax, in addition to white space: the symbol prefix,
the name-value separator, the opening quote, the closing quote, and the escape
character. Unless configured otherwise, these characters are:
  $ = [ ] \

White space is a sequence of one or more characters for which unicode.IsSpace
returns true. The escape character suppresses the effect of white space and
special characters. When it has no effect, the escape is a normal character.

An empty name is usually omitted, and in this case the separator must also be
omitted. Parameters with an empty name are known as anonymous and their values
as standalone values. When a standalone value is the name of a parameter, it
is interpreted as that name with the value "true". This effect is usually what
is wanted, but if necessary, can be avoided by using a quoted empty string: [].

White space around separators is ignored but is significant between words, as it
separates distinct values. It can be included in values by quoting. When a
quoted string contains a nested quoted string, quotes must be balanced.
Outermost quotes are removed but nested quotes are kept. A symbol is a name
preceded by the symbol prefix. A symbol reference is a symbol prefix followed by
a name between quotes.

The following examples use the default set of special characters:

  foo = bar                (1 name with 1 value)
  foo=bar foo=baz foo=quux (1 name with 3 values, assuming foo is an array)
  quux foo=bar   baz baf   (1 name with 1 value, 3 standalone values)
  foo = [ bar baz ]        (1 name with value " bar baz ")
  foo = [bar \]baz]        (1 name with value "bar ]baz")
  foo = [bar [baz]]        (1 name with value "bar [baz]", quotes nest)
  foo \= bar               (3 standalone values)
  foo                      (foo not defined as a name: 1 standalone value)
  foo                      (foo defined as boolean: 1 name with value "true")
  [] = bar                 (1 standalone value, empty name explicit)
  $sym = bar               (definition of symbol sym with value "bar")
  $X=bar foo = [x: $[X] ]  (name foo with value "x: bar ")
  $X = bar \$$[X] \\\= \[x:\ :x\]
                           (3 standalone values: "$bar", "\=", "[x: :x]")

The order in which parameters with different names are specified is not
significant. The program will not know if parameter foo was specified before
parameter bar. Multiple values of parameters can be mixed in any order. On the
other hand, multiple parameter values are kept in specification sequence. If foo
and bar are array or slice parameters, with an input like "foo=1 bar=a foo=2
bar=b bar=c foo=3", the program will see foo=[1 2 3] and bar=[a b c] but will
not know if foo came before bar. if foo and bar are scalar parameters, the last
value wins and the program will see foo=3 and bar=c.

Symbols And Substitution

Parameter names are given by the program but symbols can be freely chosen, as
long as they follow the same syntax rules as parameter names. A symbol is
defined by prefixing it once with the symbol prefix. A reference consists of a
symbol followed by name between quotes (as in the string "my $[foo] is rich").

Parameters and symbols are in distinct name spaces, and it is possible to use
the name of a parameter for a symbol, as in "$foo=bar foo=$[foo]" which is
equivalent to "foo=bar". Symbols must be defined before use: in "foo=$[x]
$x=bar", the value of "foo" cannot be resolved. Redefining a symbol has no
effect:  the first wins (unlike parameters, where the last wins). The
specification "$x=bar $x=quux foo=$[x]" is equivalent to "foo=bar".

Omission And Repetition

When a parameter is defined, it gets a target, which is the program variable
that will take values from the parameter. The target's type determines the
number and type of values the parameter can take. A target with a simple type
can take one value. If multiple values are specified, the last value wins
(again, contrast this with symbols, where the first wins). If no value is
specified, an error occurs, unless the parameter has been defined as optional.
The initial value of the target is the default value of an optional parameter.

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

Operators

There are 7 operators built into args. Operators are built-in commands which
have an effect on the state of the parser. From a user perspective, they look
like any other parameter, with a name, a name-value separator, and a value
containing subparameters. In increasing order of sophistication, they are --
(pronounced "comment"),  dump, reset, import, macro, cond, and include. Operator
subparameters are all defined as verbatim  (see Param.Verbatim) except for two
subparameters of include.

The comment operator

The -- ("comment") operator ignores its value. The value can be anything, as
long as it can be scanned by the parser, which means that quotes must be
balanced. The operator is used to insert comments into a string of parameters.
The operator is especially useful for commenting out blocks of parameters, which
can span multiple lines and can contain nested comments.

  --=[this is a comment --=[this is nested comment]
  answer = 42
  this is $[anything] that can be scanned \]
  ]

The dump operator

The dump operator is useful for debugging a complex args specification. It takes
an optional "comment" parameter, and zero or more standalone values. dump
interprets the values as parameter names and symbols and prints them line by
line on standard error with their current values. The value of a symbol is
followed by R if resolved, else by U. A name or symbol is preceded by ? if
undefined. The empty parameter name is printed between quotes. If a comment is
specified, it is printed first. A typical use may look like (import is explained
ahead):

  slic=1 $PATH=locked import=[$PATH $GOPATH $GOBBLEDYGOOK] gopath=$[GOPATH]
  path=$[PATH] slic=0.5 slic=42
  dump = [comment=[dump demo...] slic xyzzy $GOPATH $GOBBLEDYGOOK $PATH $XYZZY]

and produce the output:

  dump demo...
  slic [1 0.5 42]
  ? xyzzy
  $GOPATH R /home/user42/go
  ? $GOBBLEDYGOOK
  $PATH R locked
  ? $XYZZY

The reset operator

The reset operator is used to remove symbols. It takes a series of values,
which it interprets as symbols and removes them from the symbol table, if
present. An error occurs if values are not symbols. It is useful
because of the "first wins" principle. Contrary to parameter values,
where the "last wins", once a symbol has been set, its value cannot be
changed. When for some reason this is annoying, the symbol can be
reset before taking a new value. For example, the specification:

  $SYM1=1 $SYM2=a reset=$SYM2 $SYM1=2 $SYM2=b dump=[$SYM1 $SYM2]

produces:

  $SYM1 U 1
  $SYM2 U b

The import operator

The import operator is used to import environment variables. It takes a series
of values, which it interprets as symbols. For each symbol a value is taken from
the environment variable with the corresponding name (after removing the symbol
prefix). The value is inserted into the symbol table unless there is already an
entry for the symbol ("first wins" principle). If the environment variable does
not exist, nothing is done. For example, the specification:

  import=[$HOME $NONESUCH] $[HOME] dump=[$HOME $NONESUCH []]

produces the output:

  $HOME R /home/user42
  ? $NONESUCH
  [/home/user42]

The macro operator

The macro operator is used to expand standalone symbol references. Without
special care, standalone symbol references are either invalid (when no anonymous
parameter was defined) or are taken up as values of anonymous parameters.  They
can be used when contained in verbatim parameters, typically intended to be
parsed by sub-commands, as shown in the example with Param.Verbatim. The macro
operator makes this pattern available at the top level without needing any
auxiliary parameters.

The operator takes a series of values, which it interprets as symbols, gets
their values from the symbol table without resolving them, and passes them
recursively to Parser.Parse. An error occurs if values are not symbols, if any
symbol is not found, or if parsing fails. As an example, after parsing the input

  $macro=[foo=[number $[count]]]
  $count=1 macro=[$macro]
  reset=$count $count=2 macro=$macro

the string slice foo has the values "number 1" and "number 2". Notice the reset
operator, necessary for the new count to take effect (because of the "first
wins" principle).

The cond operator

The cond operator allows conditional parsing. It has two mandatory parameters,
"if" and "then" and an optional one, "else", all taking one value.  The value of
"if" is interpreted as a parameter name or symbol. It  evaluates to true if the
symbol exists or if the parameter has been set at least once. If the value is an
undefined parameter, an error occurs. When the value of "if" is true the value
of "then" is parsed, else it is the value of "else" which is parsed, if
specified. For example, after parsing the input

  cond=[if=[$UNDEF] then=[foo=foo] else=[foo=bar]]

the string foo has the value "bar".

The include operator

The include operator has two different modes: a basic mode for recursively
parsing a file containing parameters and a key-selection mode for extracting
name-value pairs from a file.

In basic mode, include takes a file name as anonymous parameter. It reads the
file and parses its content recursively.  Files can be included recursively and
any cyclical dependency is detected. The anonymous parameter taking the file
name is one the two operator parameters not defined as verbatim.

In key-selection mode, include takes a file name, a "keys" parameter, and an
optional "extractor" parameter. (The extractor parameter is the second operator
parameter which is not verbatim.) The value of "keys" is interpreted as a series
of standalone keys or key-translation pairs (using the current separator
character of the parser). If there is no translation, the key translates to
itself. include  extracts name-value pairs from each line of the file, and if it
finds a name matching one of the keys, it uses the value to set a parameter or a
symbol, depending on the translated key. As always a symbol is set only if it
does not already exist ("first wins" principle).

The "extractor" parameter specifies a custom regular expression for extracting
key-value pairs. The default extractor is \s*(\S+)\s*=\s*(\S+)\s*. It is an
error to specify an extractor in basic mode (when no keys are specified).

As an example, suppose there is file /home/u649/.db.conf with data we can use.
Only the user and password information is needed.

  # this is .db.conf
  the file contains other stuff
  it also contains name-value pairs we are not interested in
  "port": "4242"
  "user": "u649"
  "password": "!=.sesam568"

Parsing the input

  include=[
    /home/u649/.db.conf
    extractor=[\s*"(\S+)"\s*:\s*"(\S+)"\s*]
    keys=[user=usr password=$PASS]
  ]
  dump=[usr $PASS]

produces this dump output:

  usr u649
  $PASS U !=.sesam568

This example will parse successfully only if the user running
the program has read access to the file.

*/
package args
