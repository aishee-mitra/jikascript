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
}
