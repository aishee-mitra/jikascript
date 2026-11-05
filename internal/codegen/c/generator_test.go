package cgen

import (
	"strings"
	"testing"

	"jikascript/internal/parser"
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
	if !strings.Contains(out, "static periodic_task_t jikascript_task_blink_task;") {
		t.Fatal("expected static task registration variable")
	}
}

func TestGenerateMapsGPIOAndSkipsSchedulerStart(t *testing.T) {
	program, err := parser.ParseProgram(`const led = pin("PC1");
led.output();
task("blink", 1000, () => {
    led.toggle();
});
scheduler.start();
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Generate(program)
	if !strings.Contains(out, "gpio_output(js_led);") {
		t.Fatal("expected led.output() to map to gpio_output")
	}
	if !strings.Contains(out, "gpio_toggle(js_led);") {
		t.Fatal("expected led.toggle() to map to gpio_toggle")
	}
	if strings.Contains(out, "scheduler.start();") {
		t.Fatal("scheduler.start() should not be emitted as C selector syntax")
	}
	if strings.Contains(out, "static const int led = pin(\"PC1\");") {
		t.Fatal("pin() top-level const init should not be emitted as non-C99 static initializer")
	}
	if !strings.Contains(out, "static int js_led;") {
		t.Fatal("expected deferred top-level declaration for runtime pin initialization")
	}
	if !strings.Contains(out, "js_led = pin(\"PC1\");") {
		t.Fatal("expected runtime initialization assignment for pin binding")
	}
}

func TestGenerateSwitchAndStrictEquality(t *testing.T) {
	program, err := parser.ParseProgram(`switch (mode) { case 1: break; default: break; } if (a === b) { flag = 1; }`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Generate(program)
	if !strings.Contains(out, "switch (mode)") {
		t.Fatal("expected switch statement in generated C")
	}
	if !strings.Contains(out, "case 1:") || !strings.Contains(out, "default:") {
		t.Fatal("expected switch cases in generated C")
	}
	if strings.Contains(out, "===") {
		t.Fatal("strict equality should be emitted as C equality operator")
	}
}

func TestGenerateBoundedArrayHelpers(t *testing.T) {
	program, err := parser.ParseProgram(`let readings: int32[4] = [10, 20];
readings.push(30);
const latest = readings.pop();
readings[0] = latest;
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Generate(program)
	if !strings.Contains(out, "js_readings_push") || !strings.Contains(out, "js_readings_pop") {
		t.Fatal("expected array helper functions for push/pop")
	}
	if !strings.Contains(out, ".length") || !strings.Contains(out, ".capacity") {
		t.Fatal("expected logical length and fixed capacity fields")
	}
	if !strings.Contains(out, "js_readings_set") || !strings.Contains(out, "js_readings_get") {
		t.Fatal("expected indexed read/write helpers")
	}
	if !strings.Contains(out, "jikascript_array_fail") {
		t.Fatal("expected shared array failure handler")
	}
	if !strings.Contains(out, "index < 0") || !strings.Contains(out, ">= js_readings.length") {
		t.Fatal("expected runtime bounds checks against logical length")
	}
	if !strings.Contains(out, "js_readings.length >= js_readings.capacity") {
		t.Fatal("expected push capacity guard")
	}
	pushPos := strings.Index(out, "js_readings_push(30);")
	popPos := strings.Index(out, "js_latest = js_readings_pop();")
	if pushPos == -1 || popPos == -1 || pushPos > popPos {
		t.Fatal("expected top-level array operations to preserve source order")
	}
}

func TestGenerateObjectRecordsAndPropertyAccess(t *testing.T) {
	program, err := parser.ParseProgram(`const device = { name: "sensor-1", pin: 3, enabled: true };
device.pin = 4;
device.enabled = false;
`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Generate(program)
	if !strings.Contains(out, "typedef struct {") || !strings.Contains(out, "const char * name;") || !strings.Contains(out, "int32_t pin;") || !strings.Contains(out, "bool enabled;") {
		t.Fatalf("expected C record definition, got:\n%s", out)
	}
	if !strings.Contains(out, ".name = \"sensor-1\"") || !strings.Contains(out, "js_device.pin = 4;") || !strings.Contains(out, "js_device.enabled = false;") {
		t.Fatalf("expected object initializer and direct property writes, got:\n%s", out)
	}
	if strings.Contains(out, "const jikascript_object") {
		t.Fatal("const object binding must retain mutable C fields")
	}
}

func TestGenerateObjectFunctionArgumentAndReturn(t *testing.T) {
	program, err := parser.ParseProgram(`function setPin(device) { device.pin = 4; return { pin: 4 }; }
const device = { pin: 3 };
const updated = setPin(device);`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Generate(program)
	if !strings.Contains(out, "static jikascript_object_1_t js_setPin(jikascript_object_1_t js_device)") {
		t.Fatalf("expected object parameter and return type, got:\n%s", out)
	}
	if !strings.Contains(out, "js_updated = js_setPin(js_device);") {
		t.Fatalf("expected object-return assignment, got:\n%s", out)
	}
}

func TestGenerateInferredArrayOfObjects(t *testing.T) {
	program, err := parser.ParseProgram(`let devices = [{ pin: 1 }, { pin: 2 }];`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Generate(program)
	if !strings.Contains(out, "jikascript_object_1_t data[2]") || !strings.Contains(out, "js_devices.data[0] = (jikascript_object_1_t){ .pin = 1 };") {
		t.Fatalf("expected statically allocated array of record values, got:\n%s", out)
	}
}

func TestGenerateManglesMainBinding(t *testing.T) {
	program, err := parser.ParseProgram(`let main = 1;`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Generate(program)
	if strings.Contains(out, "int main = 1;") {
		t.Fatalf("user binding should not collide with C main symbol, got:\n%s", out)
	}
	if !strings.Contains(out, "static int js_main = 1;") {
		t.Fatalf("expected mangled user binding name, got:\n%s", out)
	}
}

func TestGenerateMapsHighLowConstants(t *testing.T) {
	program, err := parser.ParseProgram(`const led = pin("PC1"); led.write(HIGH); led.write(LOW);`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	out := Generate(program)
	if !strings.Contains(out, "gpio_write(js_led, 1);") || !strings.Contains(out, "gpio_write(js_led, 0);") {
		t.Fatalf("expected HIGH/LOW constants to map to C integer values, got:\n%s", out)
	}
}
