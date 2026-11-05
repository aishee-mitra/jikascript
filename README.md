# JikaScript

JikaScript is a lightweight ECMAScript-inspired language for embedded development.
Source files (`.jika`) are compiled by a Go-based compiler into readable C99, then
built with target-native tooling.

Authoritative requirements live in [SPEC.md](/Users/sayan/Code/JikaScript/SPEC.md).

## Repository layout

- [cmd/jikascript/](/Users/sayan/Code/JikaScript/cmd/jikascript): CLI (`check`, `transpile`, `build`)
- [internal/](/Users/sayan/Code/JikaScript/internal): lexer, parser, AST, semantic checker, C code generator
- [runtime/periodic/](/Users/sayan/Code/JikaScript/runtime/periodic): standalone pure-C scheduler
- [targets/ch32fun/](/Users/sayan/Code/JikaScript/targets/ch32fun): CH32V003 target config and GPIO backend
- [examples/](/Users/sayan/Code/JikaScript/examples): `.jika` sample programs

## Prerequisites

- Go 1.22+
- C compiler for host-side runtime checks (`gcc` or compatible)

For CH32V003 firmware builds:
- CH32fun checkout
- RISC-V GNU toolchain with:
  - `riscv64-unknown-elf-gcc`
  - `riscv64-unknown-elf-objcopy`

## Building the CLI

```bash
go build ./cmd/jikascript
```

## CLI usage

```text
jikascript check <input.jika>
jikascript transpile <input.jika> [-o output.c] [--target ch32v003]
jikascript build <input.jika> [--target ch32v003] [--optimize size|speed]
```

### check

Validate syntax and semantics (no C generation):

```bash
jikascript check examples/blink.jika
```

### transpile

Generate readable C99:

```bash
jikascript transpile examples/blink.jika -o build/blink.c
```

### build (CH32V003 + CH32fun)

Set environment:

```bash
export JIKASCRIPT_CH32FUN=/absolute/path/to/ch32fun
```

Optional tool overrides:

```bash
export JIKASCRIPT_RISCV_GCC=/path/to/riscv64-unknown-elf-gcc
export JIKASCRIPT_OBJCOPY=/path/to/riscv64-unknown-elf-objcopy
```

Then build:

```bash
jikascript build examples/blink.jika --target ch32v003 --optimize size
```

Artifacts:

- `build/<name>.c`
- `build/<name>.elf`
- `build/<name>.hex`
- `build/<name>.bin`

## Examples

- [blink.jika](/Users/sayan/Code/JikaScript/examples/blink.jika)
- [multiple_tasks.jika](/Users/sayan/Code/JikaScript/examples/multiple_tasks.jika)
- [conditional_task.jika](/Users/sayan/Code/JikaScript/examples/conditional_task.jika)

## Validation commands

```bash
go test ./...
gcc -std=c99 -Wall -Wextra runtime/periodic/test_periodic.c -o /tmp/periodic_test && /tmp/periodic_test
```
