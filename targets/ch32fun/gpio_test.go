package ch32fun

import "testing"

func TestIsValidPin(t *testing.T) {
	if !IsValidPin("PC1") {
		t.Fatal("expected PC1 to be valid")
	}
	if IsValidPin("PX1") {
		t.Fatal("expected PX1 to be invalid")
	}
}

func TestIsValidPinMethod(t *testing.T) {
	if !IsValidPinMethod("toggle") {
		t.Fatal("expected toggle to be valid pin method")
	}
	if IsValidPinMethod("pulse") {
		t.Fatal("expected pulse to be invalid pin method")
	}
}
