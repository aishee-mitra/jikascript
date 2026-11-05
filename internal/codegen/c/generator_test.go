package cgen

import (
	"strings"
	"testing"

	"microjs/internal/parser"
)

func TestGenerateContainsMainAndTask(t *testing.T) {
	program, err := parser.ParseProgram(`task("blink", 1000, () => {
    led.toggle();
});
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Generate(program)
	if !strings.Contains(out, "periodic_init") {
		t.Fatal("expected periodic_init in generated C")
	}
	if !strings.Contains(out, "blink") {
		t.Fatal("expected task name in generated C")
	}
}
