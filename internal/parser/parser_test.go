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

func TestParseSwitchBreakContinueAndBitwise(t *testing.T) {
	program, err := ParseProgram(`switch (mode) {
case 1:
  flags = flags | 1;
  break;
default:
  while (x < 10) { x += 1; continue; }
}
if ((a << 1) === (b ^ 2)) { c &= 3; }
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(program.Statements))
	}
}

func TestParseBoundedArraySyntax(t *testing.T) {
	program, err := ParseProgram(`let readings: int32[8] = [10, 20, 30];
readings.push(40);
const latest = readings.pop();
const count = readings.length;
readings[0] = latest;
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(program.Statements) != 5 {
		t.Fatalf("expected 5 statements, got %d", len(program.Statements))
	}
}

func TestParseEmptyBoundedArrayLiteral(t *testing.T) {
	program, err := ParseProgram(`let queue: uint16[8] = [];`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}
}
