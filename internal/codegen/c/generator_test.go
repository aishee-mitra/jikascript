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
	if !strings.Contains(out, "gpio_output(led);") {
		t.Fatal("expected led.output() to map to gpio_output")
	}
	if !strings.Contains(out, "gpio_toggle(led);") {
		t.Fatal("expected led.toggle() to map to gpio_toggle")
	}
	if strings.Contains(out, "scheduler.start();") {
		t.Fatal("scheduler.start() should not be emitted as C selector syntax")
	}
	if strings.Contains(out, "static const int led = pin(\"PC1\");") {
		t.Fatal("pin() top-level const init should not be emitted as non-C99 static initializer")
	}
	if !strings.Contains(out, "static int led;") {
		t.Fatal("expected deferred top-level declaration for runtime pin initialization")
	}
	if !strings.Contains(out, "led = pin(\"PC1\");") {
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
