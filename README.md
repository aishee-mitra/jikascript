# MicroJS

MicroJS is a lightweight ECMAScript-inspired language for embedded development.
Source files (`.mic`) are compiled by a Go-based compiler into readable C99, then
built with target-native tooling.

Authoritative requirements live in [SPEC.md](/Users/sayan/Code/MicroJS/SPEC.md).

## Repository layout

- [cmd/microjs/](/Users/sayan/Code/MicroJS/cmd/microjs): CLI (`check`, `transpile`, `build`)
- [internal/](/Users/sayan/Code/MicroJS/internal): lexer, parser, AST, semantic checker, C code generator
- [runtime/periodic/](/Users/sayan/Code/MicroJS/runtime/periodic): standalone pure-C scheduler
- [targets/ch32fun/](/Users/sayan/Code/MicroJS/targets/ch32fun): CH32V003 target config and GPIO backend
- [examples/](/Users/sayan/Code/MicroJS/examples): `.mic` sample programs

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
go build ./cmd/microjs
```

## CLI usage

```text
microjs check <input.mic>
microjs transpile <input.mic> [-o output.c] [--target ch32v003]
microjs build <input.mic> [--target ch32v003] [--optimize size|speed]
```

### check

Validate syntax and semantics (no C generation):

```bash
microjs check examples/blink.mic
```

### transpile

Generate readable C99:

```bash
microjs transpile examples/blink.mic -o build/blink.c
```

### build (CH32V003 + CH32fun)

Set environment:

```bash
export MICROJS_CH32FUN=/absolute/path/to/ch32fun
```

Optional tool overrides:

```bash
export MICROJS_RISCV_GCC=/path/to/riscv64-unknown-elf-gcc
export MICROJS_OBJCOPY=/path/to/riscv64-unknown-elf-objcopy
```

Then build:

```bash
microjs build examples/blink.mic --target ch32v003 --optimize size
```

Artifacts:

- `build/<name>.c`
- `build/<name>.elf`
- `build/<name>.hex`
- `build/<name>.bin`

## Examples

- [blink.mic](/Users/sayan/Code/MicroJS/examples/blink.mic)
- [multiple_tasks.mic](/Users/sayan/Code/MicroJS/examples/multiple_tasks.mic)
- [conditional_task.mic](/Users/sayan/Code/MicroJS/examples/conditional_task.mic)

## Validation commands

```bash
go test ./...
gcc -std=c99 -Wall -Wextra runtime/periodic/test_periodic.c -o /tmp/periodic_test && /tmp/periodic_test
```
