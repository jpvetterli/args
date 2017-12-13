/*
Package args defines parameters and parses strings and configuration files.
It provides a simple language and a small set of built-in operators. Here is a
short example:

 $greeting = hello $subject = world
 config = [
   greet=$$greeting
 ]
 exec = [hello.say=[$$subject]]

Substitution


Variables are substituted.
Symbols are found using a marker. This document assumes the marker is the $
sign.

A symbol is a string of letters or digits (as tested by unicode.IsLetter and
unicode.isDigit), hyphens or undersores and cannot contain a marker.

A reference consists consists of a symbol preceded either by 2 markers (as in
the string "my $$foo is rich") or preceded by 3 markers and followed by 1 (as
in "$$$foo$bar"). The second notation is useful when a symbol reference  is
directly followed by a valid symbol character.

Substitution stops on the first invalid UTF8 character, with the
exception of a byte order mark character in the first position, which
is discarded.


FROM-NAME-VALUE-SCANNER


The following syntax is enforced:

There following characters are special: the equal sign (=), the left and
right square brackets ([]), and the backslash (\), and all space characters,
as tested by unicode.IsSpace. Any special character can be escaped by
preceding it with a backslash. Such escaped special characters are handled
like usual characters. Escaping a usual character has no effect, the
backslash is kept. For example \\ becomes \ while \a remains \a.

A name or a value is a raw string not containing any special character or a
quoted string which can contain any character. A quoted string is enclosed in
square brackets, like [ this ] (the blanks belong to the quoted string). If a
quoted string contains brackets they must be balanced. Outermost brackets are
removed but nested brackets are kept.

A name-value pair is separated by an equal sign. Strings without an equal
sign between them are standalone values.

Examples

	a = b		(name-value pair "a", "b")
	a \= b		(value "a", value "=", value "b")
	a = b c		(name-value pair "a", "b", value "c")
	a = [b c]	(name-value pair "a", "b c")
	[a b] = [c [d e]] [f=g]
			(name-value pair "a b", "c [d e]", value "f=g")



*/
package args
