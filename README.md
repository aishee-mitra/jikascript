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
jikascript build <input.jika> [-o output-prefix] [--target ch32v003] [--optimize size|speed]
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

## Validation commands

```bash
go test ./...
gcc -std=c99 -Wall -Wextra runtime/periodic/test_periodic.c runtime/periodic/periodic.c -I runtime/periodic -o /tmp/periodic_test && /tmp/periodic_test
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
