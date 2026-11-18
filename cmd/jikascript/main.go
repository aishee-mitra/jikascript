package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"jikascript/internal/backend/platformio"
	"jikascript/internal/checker"
	cgen "jikascript/internal/codegen/c"
	"jikascript/internal/parser"
	"jikascript/internal/version"
	"jikascript/targets/ch32fun"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "-h", "--help", "help":
		printUsage()
	case "version", "--version":
		fmt.Printf("JikaScript %s (language spec %s)\n", version.Compiler, version.Spec)
	case "check":
		if err := runCheck(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "transpile":
		if err := runTranspile(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "build":
		if err := runBuild(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "flash":
		if err := runFlash(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("usage:")
	fmt.Println("  jikascript version")
	fmt.Println("  jikascript check <input.jika>")
	fmt.Println("  jikascript transpile <input.jika> [--output=<file.c>] [--target=<target>]")
	fmt.Println("  jikascript build <input.jika|input.c> [--output=<prefix>] [--target=<target>] [--board=<board>] [--env=<env>] [--optimize=<size|speed>]")
	fmt.Println("  jikascript flash <input.jika|input.c> [--board=<board>] [--env=<env>] [--port=<device>] [--optimize=<size|speed>]")
	fmt.Println()
	fmt.Println("options accept both --name=value and --name value; -o is a short form of --output.")
}

func runCheck(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing input file")
	}
	path := args[0]
	if filepath.Ext(path) != ".jika" {
		return fmt.Errorf("input must be a .jika source file: %s", path)
	}
	program, err := parser.ParseFile(path)
	if err != nil {
		return err
	}
	if err := checker.Check(program); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	fmt.Printf("check: %s OK\n", path)
	return nil
}

func runTranspile(args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if err := validateTarget(opts.target); err != nil {
		return err
	}
	if opts.input == "" {
		return fmt.Errorf("missing input file")
	}
	if filepath.Ext(opts.input) != ".jika" {
		return fmt.Errorf("input must be a .jika source file: %s", opts.input)
	}
	program, err := parser.ParseFile(opts.input)
	if err != nil {
		return err
	}
	if err := checker.Check(program); err != nil {
		return fmt.Errorf("%s: %w", opts.input, err)
	}
	output, err := cgen.Generate(program)
	if err != nil {
		return fmt.Errorf("%s: %w", opts.input, err)
	}
	if opts.output == "" {
		fmt.Print(output)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(opts.output), 0o755); err != nil {
		return err
	}
	return os.WriteFile(opts.output, []byte(output), 0o644)
}

// generateProject writes a PlatformIO project for the given .jika source by
// staging the generated main.c, runtime files, and PlatformIO configuration.
func generateProject(inputPath string, project platformio.Project) error {
	program, err := parser.ParseFile(inputPath)
	if err != nil {
		return err
	}
	if err := checker.Check(program); err != nil {
		return fmt.Errorf("%s: %w", inputPath, err)
	}
	mainC, err := cgen.Generate(program)
	if err != nil {
		return fmt.Errorf("%s: %w", inputPath, err)
	}

	srcDir := filepath.Join(project.Dir, "src")
	includeDir := filepath.Join(project.Dir, "include")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.c"), []byte(mainC), 0o644); err != nil {
		return err
	}
	if err := ensureFunConfig(includeDir); err != nil {
		return err
	}
	for _, file := range runtimeFiles {
		sourcePath, err := resolveProjectPath(file.source)
		if err != nil {
			return err
		}
		if err := copyFile(sourcePath, filepath.Join(project.Dir, file.destination)); err != nil {
			return err
		}
	}
	return project.WriteConfig()
}

var runtimeFiles = []struct {
	source      string
	destination string
}{
	{source: "runtime/periodic/periodic.c", destination: "src/runtime/periodic.c"},
	{source: "runtime/periodic/periodic.h", destination: "include/periodic.h"},
	{source: "targets/ch32fun/runtime/gpio.c", destination: "src/runtime/gpio.c"},
	{source: "targets/ch32fun/runtime/gpio.h", destination: "include/gpio.h"},
	{source: "targets/ch32fun/runtime/jikascript_pins.h", destination: "include/jikascript_pins.h"},
	{source: "targets/ch32fun/runtime/periodic_ch32fun.c", destination: "src/runtime/periodic_ch32fun.c"},
	{source: "targets/ch32fun/runtime/periodic_ch32fun.h", destination: "include/periodic_ch32fun.h"},
	{source: "targets/ch32fun/runtime/uart.c", destination: "src/runtime/uart.c"},
	{source: "targets/ch32fun/runtime/uart.h", destination: "include/uart.h"},
	{source: "targets/ch32fun/runtime/pwm.c", destination: "src/runtime/pwm.c"},
	{source: "targets/ch32fun/runtime/pwm.h", destination: "include/pwm.h"},
}

func copyFile(source, destination string) error {
	contents, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, contents, 0o644)
}

func resolveProjectPath(relativePath string) (string, error) {
	if filepath.IsAbs(relativePath) {
		return relativePath, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, relativePath)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("unable to locate project file %q from current directory", relativePath)
}

func runBuild(args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if err := validateTarget(opts.target); err != nil {
		return err
	}
	if opts.input == "" {
		return fmt.Errorf("missing input file")
	}

	project, outputDir, base, err := preparePlatformIOProject(opts)
	if err != nil {
		return err
	}
	runner := platformio.NewRunner()
	if err := runner.EnsureAvailable(); err != nil {
		return err
	}
	if err := runner.Build(project); err != nil {
		return err
	}
	artifacts, err := platformio.CopyArtifacts(project, outputDir, base)
	if err != nil {
		return err
	}
	fmt.Printf("build: generated %s\n", strings.Join(artifacts, ", "))
	return nil
}

// runFlash builds the firmware with PlatformIO and uploads it to the board.
func runFlash(args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if err := validateTarget(opts.target); err != nil {
		return err
	}
	if opts.input == "" {
		return fmt.Errorf("missing input file")
	}

	project, outputDir, base, err := preparePlatformIOProject(opts)
	if err != nil {
		return err
	}
	runner := platformio.NewRunner()
	if err := runner.EnsureAvailable(); err != nil {
		return err
	}
	if err := runner.Upload(project, opts.port); err != nil {
		return err
	}
	artifacts, err := platformio.CopyArtifacts(project, outputDir, base)
	if err != nil {
		return err
	}
	copiedELF := false
	for _, artifact := range artifacts {
		if filepath.Ext(artifact) == ".elf" {
			copiedELF = true
			break
		}
	}
	if !copiedELF {
		fmt.Fprintf(os.Stderr, "flash: warning: upload succeeded but expected artifact %s was not found\n", project.ArtifactPath("firmware.elf"))
	}
	if len(artifacts) == 0 {
		fmt.Println("flash: upload succeeded")
		return nil
	}
	fmt.Printf("flash: uploaded %s\n", strings.Join(artifacts, ", "))
	return nil
}

// preparePlatformIOProject materializes a PlatformIO project for the build
// input and returns the project plus the artifact destination.
func preparePlatformIOProject(opts cliOptions) (platformio.Project, string, string, error) {
	board, err := platformio.ResolveBoard(opts.board)
	if err != nil {
		return platformio.Project{}, "", "", err
	}
	optimize, err := platformio.ResolveOptimize(opts.optimize)
	if err != nil {
		return platformio.Project{}, "", "", err
	}
	env := strings.TrimSpace(opts.env)
	if env == "" {
		env = platformio.DefaultEnv
	}
	outputDir, base := resolveBuildOutput(opts.input, opts.output)
	project := platformio.Project{
		Dir:      filepath.Join("out", base+"-pio"),
		Env:      env,
		Board:    board,
		Optimize: optimize,
	}

	switch strings.ToLower(filepath.Ext(opts.input)) {
	case ".jika":
		if err := generateProject(opts.input, project); err != nil {
			return platformio.Project{}, "", "", err
		}
	case ".c":
		if err := generateProjectFromC(opts.input, project); err != nil {
			return platformio.Project{}, "", "", err
		}
	default:
		return platformio.Project{}, "", "", fmt.Errorf("input must be a .jika source file or generated .c file: %s", opts.input)
	}
	return project, outputDir, base, nil
}

// generateProjectFromC stages a hand-edited generated C file into a PlatformIO
// project so the review/edit loop keeps working without a .jika source.
func generateProjectFromC(inputPath string, project platformio.Project) error {
	mainC, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	srcDir := filepath.Join(project.Dir, "src")
	includeDir := filepath.Join(project.Dir, "include")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.c"), mainC, 0o644); err != nil {
		return err
	}
	if err := ensureFunConfig(includeDir); err != nil {
		return err
	}
	for _, file := range runtimeFiles {
		sourcePath, err := resolveProjectPath(file.source)
		if err != nil {
			return err
		}
		if err := copyFile(sourcePath, filepath.Join(project.Dir, file.destination)); err != nil {
			return err
		}
	}
	return project.WriteConfig()
}

type cliOptions struct {
	input    string
	output   string
	target   string
	optimize string
	board    string
	env      string
	port     string
}

func parseOptions(args []string) (cliOptions, error) {
	opts := cliOptions{
		target: ch32fun.NewTarget().Name,
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--target=") {
			opts.target = strings.TrimPrefix(arg, "--target=")
			continue
		}
		if strings.HasPrefix(arg, "--optimize=") {
			opts.optimize = strings.TrimPrefix(arg, "--optimize=")
			continue
		}
		if strings.HasPrefix(arg, "--board=") {
			opts.board = strings.TrimPrefix(arg, "--board=")
			continue
		}
		if strings.HasPrefix(arg, "--env=") {
			opts.env = strings.TrimPrefix(arg, "--env=")
			continue
		}
		if strings.HasPrefix(arg, "--port=") {
			opts.port = strings.TrimPrefix(arg, "--port=")
			continue
		}
		if strings.HasPrefix(arg, "--output=") {
			opts.output = strings.TrimPrefix(arg, "--output=")
			continue
		}
		switch arg {
		case "--target":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --target")
			}
			i++
			opts.target = args[i]
		case "--optimize":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --optimize")
			}
			i++
			opts.optimize = args[i]
		case "--board":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --board")
			}
			i++
			opts.board = args[i]
		case "--env":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --env")
			}
			i++
			opts.env = args[i]
		case "--port":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --port")
			}
			i++
			opts.port = args[i]
		case "--output":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for --output")
			}
			i++
			opts.output = args[i]
		case "-o":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("missing value for -o")
			}
			i++
			opts.output = args[i]
		default:
			if strings.HasPrefix(arg, "-") {
				return opts, fmt.Errorf("unknown option: %s", arg)
			}
			if opts.input == "" {
				opts.input = arg
				continue
			}
			if opts.output == "" {
				opts.output = arg
				continue
			}
			return opts, fmt.Errorf("unexpected argument: %s", arg)
		}
	}
	if opts.optimize != "" && opts.optimize != "size" && opts.optimize != "speed" {
		return opts, fmt.Errorf("unsupported optimize mode %q (use size or speed)", opts.optimize)
	}
	return opts, nil
}

func validateTarget(target string) error {
	if target != ch32fun.NewTarget().Name {
		return fmt.Errorf("unsupported target %q", target)
	}
	return nil
}

func resolveBuildOutput(inputPath, output string) (string, string) {
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	if strings.TrimSpace(output) == "" {
		return filepath.Join("build"), base
	}
	trimmed := strings.TrimSpace(output)
	withoutExt := strings.TrimSuffix(trimmed, filepath.Ext(trimmed))
	return filepath.Dir(withoutExt), filepath.Base(withoutExt)
}

func ensureFunConfig(dir string) error {
	configPath := filepath.Join(dir, "funconfig.h")
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}
	content := `#ifndef _FUNCONFIG_H
#define _FUNCONFIG_H

#define FUNCONF_USE_HSI 1
#define FUNCONF_SYSTICK_USE_HCLK 1
#define FUNCONF_USE_PLL 0
#define FUNCONF_USE_DEBUGPRINTF 0
#define FUNCONF_USE_UARTPRINTF 0

#endif
`
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath, []byte(content), 0o644)
}
