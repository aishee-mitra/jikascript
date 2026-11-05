package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"microjs/internal/checker"
	cgen "microjs/internal/codegen/c"
	"microjs/internal/parser"
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
	fmt.Println("usage: microjs <check|transpile|build> <input.mic> [output.c]")
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
	if len(args) == 0 {
		return fmt.Errorf("missing input file")
	}
	path := args[0]
	outPath := ""
	if len(args) > 1 {
		outPath = args[1]
	}
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
	output := cgen.Generate(program)
	if outPath == "" {
		fmt.Print(output)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte(output), 0o644)
}

func runBuild(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing input file")
	}
	input := args[0]
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	outputDir := filepath.Join("build")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	generated := filepath.Join(outputDir, base+".c")
	if err := runTranspile([]string{input, generated}); err != nil {
		return err
	}
	fmt.Printf("build: generated %s\n", generated)
	return nil
}
