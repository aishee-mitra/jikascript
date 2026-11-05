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
