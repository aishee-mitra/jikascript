package checker

import (
	"strings"
	"testing"

	"jikascript/internal/ast"
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

func TestCheckAcceptsPinNameVariants(t *testing.T) {
	for _, source := range []string{
		`const led = pin("PC1");`,
		`const led = pin("pc1");`,
		`const led = pin(" pc1 ");`,
	} {
		program, err := parser.ParseProgram(source)
		if err != nil {
			t.Fatalf("parse error for %q: %v", source, err)
		}
		if err := Check(program); err != nil {
			t.Fatalf("expected %q to pass check, got %v", source, err)
		}
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

func TestCheckRejectsNarrowingNumericDeclaration(t *testing.T) {
	program, err := parser.ParseProgram(`let small: uint8 = 255;`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "type mismatch") {
		t.Fatalf("expected declaration type mismatch error, got %v", err)
	}
}

func TestCheckAllowsNumericWideningDeclaration(t *testing.T) {
	program, err := parser.ParseProgram(`let count: int32 = 42;
let reading: float32 = count;`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err != nil {
		t.Fatalf("unexpected checker error: %v", err)
	}
}

func TestCheckRejectsIncompatibleScalarAssignment(t *testing.T) {
	program, err := parser.ParseProgram(`let enabled: boolean = true;
enabled = 1;`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "type mismatch assigning 'enabled'") {
		t.Fatalf("expected assignment type mismatch error, got %v", err)
	}
}

func TestCheckRejectsIncompatibleEqualityComparison(t *testing.T) {
	program, err := parser.ParseProgram(`let count = 1;
let label = "x";
if (count == label) { count = 2; }`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "cannot compare") {
		t.Fatalf("expected incompatible equality error, got %v", err)
	}
}

func TestCheckRejectsLocalArrayDeclaration(t *testing.T) {
	program, err := parser.ParseProgram(`function init() {
  let values: int32[4] = [1, 2];
  return 0;
}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "array declarations are only supported at top-level scope") {
		t.Fatalf("expected local array declaration error, got %v", err)
	}
}

func TestCheckRejectsArrayLiteralReassignment(t *testing.T) {
	program, err := parser.ParseProgram(`let values: int32[4] = [1, 2];
values = [3, 4];`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "array literals are only supported in top-level declarations") {
		t.Fatalf("expected array reassignment rejection, got %v", err)
	}
}

func TestCheckRejectsMixedBareAndValueReturns(t *testing.T) {
	program, err := parser.ParseProgram(`function test(flag) {
  if (flag) { return 1; }
  return;
}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "mixes bare and value returns") {
		t.Fatalf("expected mixed return contract error, got %v", err)
	}
}

func TestCheckRejectsInconsistentReturnTypes(t *testing.T) {
	program, err := parser.ParseProgram(`function test(flag) {
  if (flag) { return 1; }
  return "oops";
}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "inconsistent return type") {
		t.Fatalf("expected inconsistent return type error, got %v", err)
	}
}

func TestCheckAllowsCompatibleNumericReturns(t *testing.T) {
	program, err := parser.ParseProgram(`function test(flag) {
  if (flag) { return 1; }
  return 2.5;
}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err != nil {
		t.Fatalf("unexpected checker error: %v", err)
	}
}

func TestCheckResolvesForwardFunctionCallTypes(t *testing.T) {
	program, err := parser.ParseProgram(`const result = add(1, 2);
function add(a, b) { return a + b; }`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err != nil {
		t.Fatalf("unexpected checker error: %v", err)
	}
	fn := program.Statements[1].(*ast.FunctionDecl)
	if fn.ReturnType != "int32" || len(fn.ParamTypes) != 2 || fn.ParamTypes[0] != "int32" || fn.ParamTypes[1] != "int32" {
		t.Fatalf("unexpected resolved function types: params=%v return=%q", fn.ParamTypes, fn.ReturnType)
	}
}

func TestCheckRejectsIncompatibleFunctionArgument(t *testing.T) {
	program, err := parser.ParseProgram(`function add(a, b) { return a + b; }
const result = add(1, "wrong");`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Check(program); err == nil || !strings.Contains(err.Error(), "numeric operands") {
		t.Fatalf("expected incompatible function argument error, got %v", err)
	}
}
