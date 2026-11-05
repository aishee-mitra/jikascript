package checker

import (
	"strings"
	"testing"

	"microjs/internal/parser"
)

func TestCheckAllowsLoopingCode(t *testing.T) {
	program, err := parser.ParseProgram(`let count = 0;
while (count < 2) {
  count = count + 1;
}
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err != nil {
		t.Fatalf("unexpected checker error: %v", err)
	}
}

func TestCheckRejectsConstMutation(t *testing.T) {
	program, err := parser.ParseProgram(`const limit = 3;
limit = 5;
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "constant") {
		t.Fatalf("expected const mutation error, got %v", err)
	}
}

func TestCheckRejectsInvalidPin(t *testing.T) {
	program, err := parser.ParseProgram(`const led = pin("PX99");`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "unknown hardware pin") {
		t.Fatalf("expected invalid pin error, got %v", err)
	}
}

func TestCheckRejectsUnsupportedPinMethod(t *testing.T) {
	program, err := parser.ParseProgram(`const led = pin("PC1"); led.pulse();`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "unsupported pin method") {
		t.Fatalf("expected unsupported pin method error, got %v", err)
	}
}
