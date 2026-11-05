package ch32fun

import "strings"

var validPins = map[string]struct{}{
	"PC0": {}, "PC1": {}, "PC2": {}, "PC3": {}, "PC4": {}, "PC5": {}, "PC6": {}, "PC7": {},
	"PD0": {}, "PD1": {}, "PD2": {}, "PD3": {}, "PD4": {}, "PD5": {}, "PD6": {}, "PD7": {},
	"PA1": {}, "PA2": {},
}

// NormalizePin returns the canonical pin name: trimmed and uppercased.
// All validation and generated C output must use this canonical form so that
// every accepted source pin resolves to the same runtime pin identifier.
func NormalizePin(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}

// IsValidPin reports whether the CH32V003 target supports the supplied pin name.
func IsValidPin(name string) bool {
	_, ok := validPins[NormalizePin(name)]
	return ok
}

// IsValidPinMethod reports whether a method is valid on a JikaScript pin object.
func IsValidPinMethod(method string) bool {
	switch method {
	case "output", "input", "write", "read", "toggle":
		return true
	default:
		return false
	}
}
