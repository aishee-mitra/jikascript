package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"microjs/internal/checker"
	cgen "microjs/internal/codegen/c"
	"microjs/internal/parser"
	"microjs/targets/ch32fun"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
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
	fmt.Println("  microjs check <input.mic>")
	fmt.Println("  microjs transpile <input.mic> [-o output.c] [--target ch32v003]")
	fmt.Println("  microjs build <input.mic> [--target ch32v003] [--optimize size|speed]")
}

func runCheck(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing input file")
	}
	path := args[0]
	code, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	program, err := parser.ParseProgram(string(code))
	if err != nil {
		return err
	}
	if err := checker.Check(program); err != nil {
		return err
	}
	fmt.Printf("check: %s OK\n", path)
	return nil
}

func runTranspile(args []string) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if opts.input == "" {
		return fmt.Errorf("missing input file")
	}
	code, err := os.ReadFile(opts.input)
	if err != nil {
		return err
	}
	program, err := parser.ParseProgram(string(code))
	if err != nil {
		return err
	}
	if err := checker.Check(program); err != nil {
		return err
	}
	output := cgen.Generate(program)
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
	if opts.input == "" {
		return fmt.Errorf("missing input file")
	}
	base := strings.TrimSuffix(filepath.Base(opts.input), filepath.Ext(opts.input))
	outputDir := filepath.Join("build")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	generated := filepath.Join(outputDir, base+".c")
	if err := runTranspile([]string{opts.input, "-o", generated, "--target", opts.target}); err != nil {
		return err
	}
	if opts.target != "ch32v003" {
		return fmt.Errorf("unsupported target %q", opts.target)
	}
	if err := buildCH32fun(generated, outputDir, base, opts.optimize); err != nil {
		return err
	}
	fmt.Printf("build: generated %s, %s, %s\n",
		filepath.Join(outputDir, base+".elf"),
		filepath.Join(outputDir, base+".hex"),
		filepath.Join(outputDir, base+".bin"),
	)
	return nil
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

func buildCH32fun(sourceC, outputDir, base, optimize string) error {
	ch32funRoot := strings.TrimSpace(os.Getenv("MICROJS_CH32FUN"))
	if ch32funRoot == "" {
		return fmt.Errorf("MICROJS_CH32FUN must point to a CH32fun checkout")
	}

	target := ch32fun.NewTarget()
	cc := toolOrDefault("MICROJS_RISCV_GCC", target.ToolchainTriple+"-gcc")
	objcopy := toolOrDefault("MICROJS_OBJCOPY", target.ToolchainTriple+"-objcopy")

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
