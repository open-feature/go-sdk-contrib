package regex

// Hex returns a Validator that validates that a flag result is a hex color
func Hex() (Validator, error) {
	return NewValidator("^#(?:[0-9a-fA-F]{3}){1,2}$")
}
