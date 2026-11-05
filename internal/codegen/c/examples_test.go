package cgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"jikascript/internal/checker"
	"jikascript/internal/parser"
)

// TestExamplesTranspileAndCompile transpiles every examples/*.jika program,
// checks it, generates C, and C99-compiles the output against the portable
// scheduler and GPIO runtime headers. Skipped when no host C compiler or
// the examples directory is unavailable.
func TestExamplesTranspileAndCompile(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc not available for generated-C compile check")
	}
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine source location")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	examplesDir := filepath.Join(repoRoot, "examples")
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("read examples dir: %v", err)
	}

	workDir := t.TempDir()
	periodicInc := filepath.Join(repoRoot, "runtime", "periodic")
	gpioInc := filepath.Join(repoRoot, "targets", "ch32fun", "runtime")

	found := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jika") {
			continue
		}
		found = true
		name := strings.TrimSuffix(entry.Name(), ".jika")
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(examplesDir, entry.Name()))
			if err != nil {
				t.Fatalf("read example: %v", err)
			}
			program, err := parser.ParseProgram(string(source))
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			if err := checker.Check(program); err != nil {
				t.Fatalf("check error: %v", err)
			}
			out, err := Generate(program)
			if err != nil {
				t.Fatalf("generate error: %v", err)
			}
			cFile := filepath.Join(workDir, name+".c")
			if err := os.WriteFile(cFile, []byte(out), 0o644); err != nil {
				t.Fatalf("write generated C: %v", err)
			}
			binary := filepath.Join(workDir, name+".bin")
			cmd := exec.Command("gcc", "-std=c99", "-Wall", "-Wextra", "-Werror",
				"-I", periodicInc, "-I", gpioInc,
				cFile,
				filepath.Join(periodicInc, "periodic.c"),
				filepath.Join(gpioInc, "gpio_stub.c"),
				"-o", binary)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("C99 compile failed for %s:\n%s", name, output)
			}
		})
	}
	if !found {
		t.Fatal("no .jika examples found")
	}
}
