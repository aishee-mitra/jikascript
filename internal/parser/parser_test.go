package parser

import "testing"

func TestParseBlinkProgram(t *testing.T) {
	program, err := ParseProgram(`const led = pin("PC1");
led.output();
task("blink", 1000, () => {
    led.toggle();
});
scheduler.start();
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(program.Statements) != 4 {
		t.Fatalf("expected 4 statements, got %d", len(program.Statements))
	}
}

func TestParseFunctionAndIf(t *testing.T) {
	program, err := ParseProgram(`function add(a, b) { return a + b; }
if (x > 0) { y = 1; } else { y = 2; }
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}
}

func TestParseLoopStatements(t *testing.T) {
	program, err := ParseProgram(`while (count < 3) { count = count + 1; }
for (let i = 0; i < 3; i = i + 1) { count = count + 1; }
do { count = count - 1; } while (count > 0);
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(program.Statements) != 3 {
		t.Fatalf("expected 3 loop statements, got %d", len(program.Statements))
	}
}
