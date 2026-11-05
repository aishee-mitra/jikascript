package checker

import (
	"strings"
	"testing"

	"jikascript/internal/parser"
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

func TestCheckRejectsDuplicateTaskNames(t *testing.T) {
	program, err := parser.ParseProgram(`task("blink", 1000, () => { }); task("blink", 500, () => { });`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "duplicate task name") {
		t.Fatalf("expected duplicate task error, got %v", err)
	}
}

func TestCheckRejectsBreakOutsideLoopOrSwitch(t *testing.T) {
	program, err := parser.ParseProgram(`break;`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "break is only valid") {
		t.Fatalf("expected invalid break error, got %v", err)
	}
}

func TestCheckRejectsContinueOutsideLoop(t *testing.T) {
	program, err := parser.ParseProgram(`continue;`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "continue is only valid") {
		t.Fatalf("expected invalid continue error, got %v", err)
	}
}

func TestCheckAllowsSwitchAndLoopFlowControl(t *testing.T) {
	program, err := parser.ParseProgram(`let mode = 1; let x = 0; switch (mode) { case 1: break; default: while (x < 2) { continue; } }`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err != nil {
		t.Fatalf("unexpected checker error: %v", err)
	}
}

func TestCheckAllowsBoundedArrayOps(t *testing.T) {
	program, err := parser.ParseProgram(`let readings: int32[6] = [1, 2, 3];
readings.push(4);
const latest = readings.pop();
readings[0] = latest;
const count = readings.length;
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err != nil {
		t.Fatalf("unexpected checker error: %v", err)
	}
}

func TestCheckRejectsArrayCapacityOverflow(t *testing.T) {
	program, err := parser.ParseProgram(`let readings: int32[2] = [1, 2, 3];`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "exceeds declared capacity") {
		t.Fatalf("expected capacity overflow error, got %v", err)
	}
}

func TestCheckRejectsConstArrayReassignmentButAllowsMutation(t *testing.T) {
	program, err := parser.ParseProgram(`const readings: int32[4] = [1, 2];
readings.push(3);
readings = [4, 5];
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "cannot assign to constant variable") {
		t.Fatalf("expected const reassignment error, got %v", err)
	}
}

func TestCheckRejectsArrayPushTypeMismatch(t *testing.T) {
	program, err := parser.ParseProgram(`let readings: int32[4] = [1, 2];
readings.push("oops");
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "push() type mismatch") {
		t.Fatalf("expected push type mismatch error, got %v", err)
	}
}

func TestCheckAllowsConstObjectPropertyMutation(t *testing.T) {
	program, err := parser.ParseProgram(`const device = { name: "sensor", pin: 3, enabled: true };
device.pin = 4;
device.enabled = false;
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err != nil {
		t.Fatalf("unexpected checker error: %v", err)
	}
}

func TestCheckRejectsInvalidObjectPropertyAndType(t *testing.T) {
	for _, source := range []string{
		`let device = { pin: 3 }; device.temperature;`,
		`let device = { pin: 3 }; device.pin = "wrong";`,
		`let device = { pin: 3, pin: 4 };`,
	} {
		program, err := parser.ParseProgram(source)
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := Check(program); err == nil {
			t.Fatalf("expected checker error for %q", source)
		}
	}
}

func TestCheckObjectFunctionReturn(t *testing.T) {
	program, err := parser.ParseProgram(`function createDevice() { return { pin: 3, enabled: true }; }
const device = createDevice();
device.pin = 4;`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err != nil {
		t.Fatalf("unexpected checker error: %v", err)
	}
}

func TestCheckAllowsInferredArrayOfObjects(t *testing.T) {
	program, err := parser.ParseProgram(`let devices = [{ pin: 1 }, { pin: 2 }];`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err != nil {
		t.Fatalf("unexpected checker error: %v", err)
	}
}
