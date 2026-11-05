# MicroJS

This repository contains incremental implementation work for the MicroJS project
described in `SPEC.md`, including scheduler, compiler foundation, language semantics,
and initial CH32fun integration.

## Components

- `runtime/periodic`: standalone pure-C periodic scheduler library.
- `cmd/microjs`: minimal Go CLI for `check`, `transpile`, and `build`.
- `internal/lexer`, `internal/parser`, `internal/checker`, `internal/codegen`: compiler foundation.
- `targets/ch32fun`: CH32V003 target configuration and GPIO backend bindings.

## Example

```text
microjs transpile examples/blink.mic -o /tmp/blink.c
```

The generated C remains readable and deterministic, matching the specification's goal for
inspectable output.

## CH32fun build integration

`microjs build` now invokes a RISC-V toolchain for the `ch32v003` target and emits:

- `build/<name>.elf`
- `build/<name>.hex`
- `build/<name>.bin`

Required environment:

- `MICROJS_CH32FUN`: absolute path to a CH32fun checkout.

Optional overrides:

- `MICROJS_RISCV_GCC` (default: `riscv64-unknown-elf-gcc`)
- `MICROJS_OBJCOPY` (default: `riscv64-unknown-elf-objcopy`)
