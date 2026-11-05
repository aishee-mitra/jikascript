package ch32fun

import "strings"

var validPins = map[string]struct{}{
	"PC0": {}, "PC1": {}, "PC2": {}, "PC3": {}, "PC4": {}, "PC5": {}, "PC6": {}, "PC7": {},
	"PD0": {}, "PD1": {}, "PD2": {}, "PD3": {}, "PD4": {}, "PD5": {}, "PD6": {}, "PD7": {},
	"PA1": {}, "PA2": {},
}

// IsValidPin reports whether the CH32V003 target supports the supplied pin name.
func IsValidPin(name string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(name))
	_, ok := validPins[normalized]
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
