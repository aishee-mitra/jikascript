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

## Optional: justfile shortcuts

If you use [`just`](https://github.com/casey/just), the repo includes a [justfile](/Users/sayan/Code/MicroJS/justfile) with beginner-friendly recipes:

```bash
just help
just bin
just install
just get-ch32libs
just check
just transpile
just build
just build-c
```

## CLI usage

```text
jikascript check <input.jika>
jikascript transpile <input.jika> [-o output.c] [--target ch32v003]
jikascript build <input.jika|input.c> [-o output-prefix] [--target ch32v003] [--optimize size|speed]
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

`build` accepts either a `.jika` source file or a generated readable `.c` file. This lets you review or hand-edit the transpiled C before building firmware.

Set environment to the parent directory of a real CH32fun checkout:

```bash
export JIKASCRIPT_CH32FUN=/absolute/path/to/ch32fun-root
```

You can also fetch CH32fun with the helper recipe, then point the env var there:

```bash
just get-ch32libs
export JIKASCRIPT_CH32FUN="$(cd third_party/ch32fun && pwd)"
```

The build expects this layout under that path:

```text
/absolute/path/to/ch32fun-root/
  ch32fun/
    ch32fun.h
    ...
```

If `JIKASCRIPT_CH32FUN` is not set, the CH32fun build fails immediately with:

```text
JIKASCRIPT_CH32FUN must point to a CH32fun checkout
```

Optional tool overrides:

```bash
export JIKASCRIPT_RISCV_GCC=/path/to/riscv64-unknown-elf-gcc
export JIKASCRIPT_OBJCOPY=/path/to/riscv64-unknown-elf-objcopy
```

Build directly from source:

```bash
jikascript build examples/blink.jika --target ch32v003 --optimize size
```

Or generate readable C first, edit it, then build from the generated file:

```bash
jikascript transpile examples/blink.jika -o build/blink.c
# edit build/blink.c as needed
jikascript build build/blink.c --target ch32v003 --optimize size
```

Optional custom artifact prefix:

```bash
jikascript build examples/blink.jika -o dist/blink.elf
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

## Bounded arrays (v0.1)

JikaScript supports bounded, homogeneous arrays with static capacity and mutable logical length.

```javascript
let readings: int32[8] = [21, 23, 25];
readings.push(27);
const latest = readings.pop();
const count = readings.length;
readings[0] = latest;
```

`push()` is capacity-checked and `pop()`/index operations are bounds-checked in generated C.
In v0.1, array declarations are top-level only and array-literal reassignment is rejected.

## Static objects (v0.1)

Object literals are statically shaped records, not dynamic JavaScript objects.
Property names and inferred property types are checked at compile time; property access
compiles to direct C struct access with no heap allocation or runtime lookup.

```javascript
const device = { name: "sensor-1", pin: 3, enabled: true };
device.pin = 4;       // allowed: `const` freezes the binding, not its contents
```

Unknown properties, duplicate property names, and incompatible property assignments are
compile-time errors. Functions may accept and return inferred object records when their
object arguments and returned object literals are statically resolvable.

## Equality semantics (v0.1)

- `==` and `===` are strict, non-coercing equality for supported types.
- `!=` and `!==` are strict inequality.
- Mixed-type equality comparisons (for example `number` vs `string`) are compile-time errors.

## Scalar and function typing (v0.1)

Declarations without annotations are inferred and emitted with matching C types:

| Inferred JikaScript type | Generated C type |
| ------------------------ | ---------------- |
| string                   | `const char *`   |
| boolean                  | `bool`           |
| integer literal          | `int32_t`        |
| float literal            | `float`          |

Explicit annotations (`int8`, `uint8`, `int16`, `uint16`, `int32`, `uint32`,
`float32`, `boolean`, `string`) retain their mapped C types.

Top-level functions are precollected before bodies are checked, so calls may
appear before declarations. Parameter types are inferred from all call sites
(incompatible call-site types are rejected), and the return type is inferred
from all return expressions, including returns inside `if`, loops, blocks, and
switch cases. Mixed bare/value returns and incompatible value returns are
compile-time errors. A function whose parameter or return type cannot be
resolved is rejected rather than silently emitted as `int`. Generated C uses
matching parameter and return types.

## Scheduler capacity (v0.1)

The portable scheduler's task capacity is a build-time macro in
`runtime/periodic/periodic.h`:

```c
#ifndef PERIODIC_MAX_TASKS
#define PERIODIC_MAX_TASKS 8U
#endif
```

Override it at build time, for example:

```bash
gcc -DPERIODIC_MAX_TASKS=16 ...
```

Generated JikaScript C wraps every `periodic_task_add()` call so a failed
registration (invalid interval, duplicate task, or full capacity) enters a
shared non-returning `jikascript_fail()` helper instead of silently dropping
the task.

## CH32fun timer adapter (target-only)

When a CH32fun checkout is configured (`JIKASCRIPT_CH32FUN`), the target build
includes a timer adapter under `targets/ch32fun/runtime/` exposing exactly one
initializer:

```c
void periodic_ch32fun_init(void);
```

The adapter configures a hardware timer interrupt at 1 ms; the ISR calls only
`periodic_tick_ms(1U)`. Application callbacks are never executed from
interrupt context — generated startup calls, in order:

```c
periodic_init();
periodic_ch32fun_init();
```

## Validation status labels

- **Host-tested**: verified with the Go test suite and host C99 compilation
  (`go test ./...` plus the portable scheduler test below). All features
  documented above are host-tested.
- **Target-compiled**: verified by compiling firmware with the RISC-V
  toolchain against a CH32fun checkout. Not yet claimed for any feature.
- **Hardware-verified**: verified by flashing a physical CH32V003 board and
  observing behavior. Not yet claimed for any feature.

## Validation commands

```bash
go test ./...
gcc -std=c99 -Wall -Wextra runtime/periodic/test_periodic.c runtime/periodic/periodic.c -I runtime/periodic -o /tmp/periodic_test && /tmp/periodic_test
# Capacity override check:
gcc -std=c99 -Wall -Wextra -DPERIODIC_MAX_TASKS=2 runtime/periodic/test_periodic_capacity.c runtime/periodic/periodic.c -I runtime/periodic -o /tmp/periodic_cap && /tmp/periodic_cap
```

## Scheduler time source integration

The portable scheduler no longer advances time inside `periodic_run()`.
A platform timer (typically ISR-driven) must advance the scheduler clock via:

```c
periodic_tick_ms(elapsed_ms);
```

For interrupt/main-context synchronization, platforms may register critical-section hooks:

```c
periodic_set_critical_section(enter_fn, exit_fn);
```
