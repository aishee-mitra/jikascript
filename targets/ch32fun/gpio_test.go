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

func TestNormalizePin(t *testing.T) {
	cases := map[string]string{
		"PC1":    "PC1",
		"pc1":    "PC1",
		" pc1 ":  "PC1",
		"\tpd0\n": "PD0",
	}
	for input, want := range cases {
		if got := NormalizePin(input); got != want {
			t.Fatalf("NormalizePin(%q) = %q, want %q", input, got, want)
		}
		if !IsValidPin(input) {
			t.Fatalf("expected IsValidPin(%q) to be true", input)
		}
	}
	if IsValidPin("  px1  ") {
		t.Fatal("expected whitespace-padded invalid pin to remain invalid")
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
