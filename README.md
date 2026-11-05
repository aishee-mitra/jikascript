# MicroJS

This repository contains the initial Milestone 1 and Milestone 2 implementation work
for the MicroJS project described in `SPEC.md`.

## Components

- `runtime/periodic`: standalone pure-C periodic scheduler library.
- `cmd/microjs`: minimal Go CLI for `check`, `transpile`, and `build`.
- `internal/lexer`, `internal/parser`, `internal/checker`, `internal/codegen`: compiler foundation.

## Example

```text
microjs transpile examples/blink.mic -o /tmp/blink.c
```

The generated C remains readable and deterministic, matching the specification's goal for
inspectable output.
