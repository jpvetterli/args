package args

import (
	"bytes"
	"fmt"
	"unicode"
)

func nextPos(r *bytes.Reader) int {
	return int(r.Size()) - r.Len()
}

// validSpecial returns true iff char is valid as a special character.
// Valid special characters are graphic, not white space, not valid in a name.
func validSpecial(char rune) bool {
	return !valid(char) && unicode.IsGraphic(char) && !unicode.IsSpace(char)
}

// validate verifies a name
func validate(name string) error {
	for _, r := range []rune(name) {
		if !valid(r) {
			return fmt.Errorf(`"%s" cannot be used as a name because it includes the character '%c'`, name, r)
		}
	}
	return nil
}

// valid returns true iff char is valid in a parameter or symbol name.
// Valid characters are letters, digits, the hyphen and the underscore.
func valid(char rune) bool {
	return unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_'
}
