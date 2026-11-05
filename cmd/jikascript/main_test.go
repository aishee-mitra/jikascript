package main

import (
	"strings"
	"testing"
)

func TestParseOptionsDefaults(t *testing.T) {
	opts, err := parseOptions([]string{"examples/blink.jika"})
	if err != nil {
		t.Fatalf("parseOptions error: %v", err)
	}
	if opts.input != "examples/blink.jika" {
		t.Fatalf("unexpected input: %s", opts.input)
	}
	if opts.target != "ch32v003" {
		t.Fatalf("unexpected target: %s", opts.target)
	}
	if opts.optimize != "size" {
		t.Fatalf("unexpected optimize mode: %s", opts.optimize)
	}
}

func TestParseOptionsWithFlags(t *testing.T) {
	opts, err := parseOptions([]string{"examples/blink.jika", "-o", "build/blink.c", "--target", "ch32v003", "--optimize", "speed"})
	if err != nil {
		t.Fatalf("parseOptions error: %v", err)
	}
	if opts.output != "build/blink.c" {
		t.Fatalf("unexpected output: %s", opts.output)
	}
	if opts.optimize != "speed" {
		t.Fatalf("unexpected optimize mode: %s", opts.optimize)
	}
}

func TestParseOptionsRejectsUnknownOption(t *testing.T) {
	_, err := parseOptions([]string{"examples/blink.jika", "--unknown"})
	if err == nil {
		t.Fatal("expected error for unknown option")
	}
}

func TestParseOptionsRejectsUnknownOptimizeMode(t *testing.T) {
	_, err := parseOptions([]string{"examples/blink.jika", "--optimize", "balanced"})
	if err == nil || !strings.Contains(err.Error(), "unsupported optimize mode") {
		t.Fatalf("expected optimize validation error, got %v", err)
	}
}

func TestRunCheckRejectsNonMicInput(t *testing.T) {
	err := runCheck([]string{"example.txt"})
	if err == nil || !strings.Contains(err.Error(), ".jika source file") {
		t.Fatalf("expected .jika extension error, got %v", err)
	}
}

func TestValidateTargetRejectsUnknown(t *testing.T) {
	err := validateTarget("invalid")
	if err == nil || !strings.Contains(err.Error(), "unsupported target") {
		t.Fatalf("expected unsupported target error, got %v", err)
	}
}

func TestResolveBuildOutputDefault(t *testing.T) {
	dir, base := resolveBuildOutput("examples/blink.jika", "")
	if dir != "build" || base != "blink" {
		t.Fatalf("unexpected default build output: dir=%q base=%q", dir, base)
	}
}

func TestResolveBuildOutputWithCustomPath(t *testing.T) {
	dir, base := resolveBuildOutput("examples/blink.jika", "dist/fw/blink.elf")
	if dir != "dist/fw" || base != "blink" {
		t.Fatalf("unexpected custom build output: dir=%q base=%q", dir, base)
	}
}
