package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"jikascript/internal/checker"
	cgen "jikascript/internal/codegen/c"
	"jikascript/internal/parser"
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
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("usage:")
	fmt.Println("  jikascript check <input.jika>")
	fmt.Println("  jikascript transpile <input.jika> [-o output.c] [--target ch32v003]")
	fmt.Println("  jikascript build <input.jika|input.c> [--target ch32v003] [--optimize size|speed]")
}

func runCheck(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing input file")
	}
	path := args[0]
	if filepath.Ext(path) != ".jika" {
		return fmt.Errorf("input must be a .jika source file: %s", path)
	}
	code, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	program, err := parser.ParseProgram(string(code))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
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
	code, err := os.ReadFile(opts.input)
	if err != nil {
		return err
	}
	program, err := parser.ParseProgram(string(code))
	if err != nil {
		return fmt.Errorf("%s: %w", opts.input, err)
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

	source, outputDir, base, err := prepareBuildInput(opts.input, opts.output, opts.target)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	if err := buildCH32fun(source, outputDir, base, opts.optimize); err != nil {
		return err
	}
	fmt.Printf("build: generated %s, %s, %s\n",
		filepath.Join(outputDir, base+".elf"),
		filepath.Join(outputDir, base+".hex"),
		filepath.Join(outputDir, base+".bin"),
	)
	return nil
}

func prepareBuildInput(inputPath, output, target string) (string, string, string, error) {
	ext := strings.ToLower(filepath.Ext(inputPath))
	outputDir, base := resolveBuildOutput(inputPath, output)

	switch ext {
	case ".jika":
		generated := filepath.Join(outputDir, base+".c")
		if err := runTranspile([]string{inputPath, "-o", generated, "--target", target}); err != nil {
			return "", "", "", err
		}
		if err := ensureFunConfig(outputDir); err != nil {
			return "", "", "", err
		}
		return generated, outputDir, base, nil
	case ".c":
		if err := ensureFunConfig(filepath.Dir(inputPath)); err != nil {
			return "", "", "", err
		}
		return inputPath, outputDir, base, nil
	default:
		return "", "", "", fmt.Errorf("input must be a .jika source file or generated .c file: %s", inputPath)
	}
}

type cliOptions struct {
	input    string
	output   string
	target   string
	optimize string
}

func parseOptions(args []string) (cliOptions, error) {
	opts := cliOptions{
		target:   ch32fun.NewTarget().Name,
		optimize: ch32fun.NewTarget().DefaultOptimizeOpt,
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
	if opts.optimize != "size" && opts.optimize != "speed" {
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

func buildCH32fun(sourceC, outputDir, base, optimize string) error {
	ch32funRoot := strings.TrimSpace(os.Getenv("JIKASCRIPT_CH32FUN"))
	if ch32funRoot == "" {
		return fmt.Errorf("JIKASCRIPT_CH32FUN must point to a CH32fun checkout")
	}
	if err := ensureFunConfig(filepath.Dir(sourceC)); err != nil {
		return err
	}

	target := ch32fun.NewTarget()
	cc := toolOrDefault("JIKASCRIPT_RISCV_GCC", target.ToolchainTriple+"-gcc")
	objcopy := toolOrDefault("JIKASCRIPT_OBJCOPY", target.ToolchainTriple+"-objcopy")

	optFlag := "-Os"
	if optimize == "speed" {
		optFlag = "-O2"
	}

	elfOut := filepath.Join(outputDir, base+".elf")
	hexOut := filepath.Join(outputDir, base+".hex")
	binOut := filepath.Join(outputDir, base+".bin")

	args := []string{
		optFlag,
		"-march=rv32ec",
		"-mabi=ilp32e",
		"-I", filepath.Dir(sourceC),
		"-I", filepath.Join(ch32funRoot, "ch32fun"),
		"-I", filepath.Join(".", "runtime", "periodic"),
		"-I", filepath.Join(".", "targets", "ch32fun", "runtime"),
		sourceC,
		filepath.Join("runtime", "periodic", "periodic.c"),
		filepath.Join("targets", "ch32fun", "runtime", "gpio.c"),
		"-o", elfOut,
	}

	if err := runCmd(cc, args...); err != nil {
		return fmt.Errorf("risc-v gcc failed: %w", err)
	}

	if err := runCmd(objcopy, "-O", "ihex", elfOut, hexOut); err != nil {
		return fmt.Errorf("objcopy hex failed: %w", err)
	}
	if err := runCmd(objcopy, "-O", "binary", elfOut, binOut); err != nil {
		return fmt.Errorf("objcopy bin failed: %w", err)
	}
	return nil
}

func ensureFunConfig(dir string) error {
	configPath := filepath.Join(dir, "funconfig.h")
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}
	content := `#ifndef _FUNCONFIG_H
#define _FUNCONFIG_H

#define FUNCONF_USE_HSI 1
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

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func toolOrDefault(envVar, fallback string) string {
	v := strings.TrimSpace(os.Getenv(envVar))
	if v == "" {
		return fallback
	}
	return v
}
