package main

import (
	"os"
	"path/filepath"
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

func TestPrepareBuildInputAcceptsGeneratedC(t *testing.T) {
	source, dir, base, err := prepareBuildInput("build/blink.c", "", "ch32v003")
	if err != nil {
		t.Fatalf("prepareBuildInput error for generated C: %v", err)
	}
	if source != "build/blink.c" {
		t.Fatalf("unexpected generated C source path: %q", source)
	}
	if dir != "build" || base != "blink" {
		t.Fatalf("unexpected generated C output metadata: dir=%q base=%q", dir, base)
	}
}

func TestPrepareBuildInputTranspilesJikaSource(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "blink.jika")
	if err := os.WriteFile(inputPath, []byte("const led = pin(\"PC1\");\nled.output();\n"), 0o644); err != nil {
		t.Fatalf("write temp .jika file: %v", err)
	}
	outputPrefix := filepath.Join(tempDir, "firmware.elf")

	source, dir, base, err := prepareBuildInput(inputPath, outputPrefix, "ch32v003")
	if err != nil {
		t.Fatalf("prepareBuildInput error for .jika file: %v", err)
	}
	if !strings.HasSuffix(source, ".c") {
		t.Fatalf("expected generated C output path, got %q", source)
	}
	if dir != tempDir || base != "firmware" {
		t.Fatalf("unexpected .jika output metadata: dir=%q base=%q", dir, base)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("generated C file was not created: %v", err)
	}
}
