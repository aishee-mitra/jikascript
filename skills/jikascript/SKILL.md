---
name: jikascript
description: Develop embedded firmware with JikaScript, a lightweight ECMAScript-inspired language that compiles ahead-of-time to pure C99 for microcontrollers (CH32V003 via CH32fun). Use when writing, building, transpiling, flashing, or debugging .jika source files, or explaining JikaScript language constructs, cooperative scheduling, or its CLI.
version: 0.3.4
author: Sayan Chakrabarti (sayanriju)
license: MIT
platforms: [linux, macos, windows]
metadata:
  hermes:
    tags: [embedded, risc-v, wch, ch32v003, ch32fun, jikascript, cooperative-scheduler, language, ecmascript, platformio]
    related_skills: [ch32fun]
---

# JikaScript

JikaScript is a lightweight, JavaScript-inspired programming language for
microcontrollers and resource-constrained embedded systems. `.jika` source files
are compiled ahead of time by a Go toolchain into **readable pure C99**, which is
then built into firmware with the target's native toolchain (CH32V003 + CH32fun
via PlatformIO).

There is **no interpreter, VM, garbage collector, or language runtime** on the
MCU. The cooperative scheduler is a standalone pure-C library reused by the
compiler.

> Authoritative requirements live in `SPEC.md` in the jikascript repository.
> This skill is a practical reference for implementing `.jika` programs and using
> the CLI. When in doubt, follow SPEC.md semantics.

## When to Use This Skill

- Writing `.jika` source files (variables, functions, arrays, objects, tasks).
- Explaining or implementing JikaScript language constructs and semantics.
- Building firmware: `jikascript build`, `transpile`, `check`, `flash`.
- Working with the cooperative periodic scheduler and task syntax.
- Using GPIO hardware abstration (`pin("PC1")`).
- Using the UART peripheral (`uart("PD5", "PD6")`).
- Reviewing or hand-editing transpiled C before flashing.

## Quick Start

A complete minimal program — blink an LED every 1000 ms:

```javascript
// blink.jika
const led = pin("PC1");

led.output();

task("blink", 1000, () => {
    led.toggle();
});

scheduler.start();
```

Build and flash it:

```bash
jikascript check examples/blink.jika          # validate syntax + semantics
jikascript transpile examples/blink.jika --output=out/blink.c   # readable C
jikascript build examples/blink.jika          # -> build/blink.elf, build/blink.bin
jikascript flash examples/blink.jika --port=/dev/ttyUSB0
```

## Language at a Glance

```javascript
// Variables: const (immutable binding) and let (mutable)
const maxCount = 100;
let counter = 0;

// Explicit types map directly to C primitives
let reading: uint16 = 1024;
let voltage: float32 = 3.3;
let enabled: boolean = true;

// Functions (type-inferred from call sites and returns)
function add(a, b) {
    return a + b;
}

// Arrow functions (single-expression and block forms)
const square = x => x * x;
const toggle = () => { led.toggle(); };

// Control flow
if (reading > 100) { ... }
switch (state) { case 1: ...; break; default: ... }
while (running) { ... }
for (let i = 0; i < 10; i = i + 1) { ... }

// Bounded arrays: fixed capacity, mutable logical length
let readings: int32[8] = [21, 23, 25];
readings.push(27);
const latest = readings.pop();
const count = readings.length;

// Static object records -> C structs
const device = { name: "sensor-1", pin: 3, temperature: 24.5, enabled: true };

// File includes (compile-time textual splice)
include "helpers.jika";

// GPIO
const led = pin("PC1");
led.output();
led.write(HIGH);

// UART on the CH32V003 USART1 default pins (TX=PD5, RX=PD6)
const serial = uart("PD5", "PD6");
serial.println("hello");

// Cooperative periodic tasks
task("heartbeat", 500, () => { led.toggle(); });

// Start the scheduler
scheduler.start();
```

## Reference Files

Detailed, exhaustive references available on demand:

- `references/language.md` — Full language construct reference: variables, types, functions, arrows, control flow, operators, equality semantics, reserved keywords, include.
- `references/arrays.md` — Bounded arrays: declaration, capacity inference, methods, bounds safety, failure semantics.
- `references/objects.md` — Static object records: declaration, struct lowering, mutable properties, compile-time safety.
- `references/tasks.md` — Cooperative periodic tasks and the scheduler model.
- `references/scheduler.md` — Standalone pure-C periodic scheduler API, scheduling rules, and timer adapter.
- `references/hardware.md` — GPIO and UART hardware API (`pin`, `uart`, `output`, `input`, `read`, `write`, `println`, `toggle`, `HIGH`, `LOW`).
- `references/includes.md` — Multi-file `include` splicing rules and example.
- `references/cli.md` — Install, build, CLI commands, flags, board/env selection, optimization, PlatformIO artifacts.
- `references/example-programs.md` — Complete end-to-end example programs.

## Best Practice Notes

- Prefer `const` over `let`; the compiler enforces immutable bindings.
- Use explicit integer widths (`uint8`, `uint16`, `int32`, ...) near registers
  and memory-mapped data.
- `==` and `===` are both strict and non-coercing. Mixed-type comparisons are
  compile-time errors — never compare a `number` to a `string`.
- Reserve the words `task`, `scheduler`, `start`, `pin`, `uart`, `include`,
  `HIGH`, and `LOW`: they are language syntax or fixed built-ins, not ordinary
  identifiers.
- `HIGH` and `LOW` are predefined pin-state constants with fixed meanings
  `HIGH == 1` and `LOW == 0`; do not shadow or rebind them in user code.
- Array push/pop and index access are bounds- and capacity-checked at runtime;
  statically provable violations are rejected at compile time.
- The scheduler runs callbacks in main context; a long callback delays others.
- `include` is a compile-time textual splice, resolved relative to the including
  file, top-level only, and rejected on circularity.