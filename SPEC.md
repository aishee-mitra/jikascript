# JikaScript
## Language, Compiler & Runtime Specification

**Specification version:** 0.1.0  
**Status:** Initial implementation specification  
**Language reference:** ECMAScript 2024  
**Compiler implementation:** Go  
**Initial MCU target:** WCH CH32V003  
**Hardware framework:** CH32fun  
**Compiler output:** Pure C (C99)  
**Scheduler:** Standalone cooperative scheduling library written in C  
**Source extension:** `.jika`

---

# 1. Project Overview

JikaScript is a lightweight, JavaScript-inspired programming language designed for microcontrollers and resource-constrained embedded systems.

The language should follow ECMAScript conventions and semantics as closely as practical, while implementing only a carefully selected subset suitable for embedded development.

JikaScript programs are compiled ahead of time into native C code. The resulting C code is compiled using the target's native C toolchain to produce firmware.

JikaScript does not require an interpreter, virtual machine, garbage collector, or language runtime engine on the microcontroller.

The initial implementation targets WCH CH32V003 using CH32fun.

The project consists of three independently useful components:

1. **JikaScript compiler:** A Go-based source-to-C compiler.
2. **JikaScript language:** An ECMAScript-inspired embedded programming language.
3. **Periodic scheduler:** A standalone, reusable cooperative scheduling library written in pure C.

The scheduler must be usable independently of JikaScript in ordinary C embedded applications.

---

# 2. Core Design Principles

The following principles are mandatory architectural requirements.

1. **ECMAScript compatibility:** Follow ECMAScript syntax and semantics wherever practical for supported features.
2. **Simple programming experience:** Make embedded programming approachable to developers familiar with JavaScript.
3. **Native compilation:** Generate pure C source code, not C++.
4. **Cooperative scheduling:** Use cooperative periodic tasks as the primary programming model.
5. **Standalone runtime:** Keep the scheduler independent of the compiler and language.
6. **Static resource usage:** Prefer compile-time allocation and predictable memory consumption.
7. **No mandatory heap:** The initial language implementation must not require dynamic memory allocation.
8. **Hardware abstraction:** Keep language features independent of specific microcontroller hardware.
9. **Readable generated code:** Developers must be able to inspect and understand generated C.
10. **Portable compiler:** The Go compiler must not be tied to CH32V003-specific functionality.

---

# 3. Overall Architecture

The project contains three logical layers.

## 3.1 JikaScript compiler

Written in Go.

Responsibilities:

- Read `.jika` files.
- Perform lexical analysis.
- Parse source code.
- Construct an abstract syntax tree.
- Perform semantic analysis and type checking.
- Generate C code.
- Support target-specific code generation.
- Provide command-line build and inspection operations.

## 3.2 JikaScript runtime

Written in pure C.

Responsibilities:

- Cooperative periodic task scheduling.
- Task registration.
- Task execution.
- Timer integration.
- Minimal runtime support where required.

The runtime must be independent of JikaScript.

## 3.3 Hardware backend

Target-specific hardware support.

Initial backend:

- WCH CH32V003.
- CH32fun.
- GPIO.
- Hardware timer integration.
- Later: ADC, PWM, UART, SPI and I2C.

Hardware-specific functions should be exposed through a consistent JikaScript hardware API.

## 3.4 Compilation pipeline

```text
             JikaScript source (.jika)
                      |
                      v
                 Go compiler
                      |
          +-----------+-----------+
          |                       |
       Lexer                   Parser
          |                       |
          +-----------+-----------+
                      |
                      v
              Abstract Syntax Tree
                      |
                      v
              Semantic Analysis
                      |
                      v
               C Code Generator
                      |
             +--------+--------+
             |                 |
          transpile           build
             |                 |
             v                 v
        Readable C        Optimized C
                               |
                               v
                       C compiler (GCC)
                               |
                               v
                        Firmware output
```

The compiler must generate pure C99-compatible code.

Do not generate C++.

Do not introduce LLVM or a complex intermediate representation in v0.1.

An AST-based direct C generator is sufficient.

---

# 4. JikaScript Language Design

## 4.1 ECMAScript compatibility strategy

ECMAScript 2024 will be used as the reference specification.

JikaScript is not intended to be a complete JavaScript implementation.

Instead, it implements a well-defined subset of ECMAScript.

Important requirement:

**When a language feature is supported, its behavior should match ECMAScript semantics as closely as practical.**

Do not implement JavaScript-looking syntax with arbitrary or inconsistent behavior.

For example:

- Operator precedence should follow ECMAScript.
- Lexical scoping should follow ECMAScript.
- `const` and `let` should follow their ECMAScript rules.
- Function declarations and arrow functions should follow ECMAScript conventions.
- Equality operators must have clearly defined semantics.
- Truthiness and short-circuit evaluation should be consistent with ECMAScript for supported values.

Where full ECMAScript behavior is impractical, explicitly document the restriction.

## 4.2 Supported language features

### Variables

Support:

```javascript
let counter = 0;
const maxCount = 100;
```

Requirements:

- `let` declares mutable bindings.
- `const` declares immutable bindings.
- Lexical block scoping.
- Variable initialization.
- Variable assignment.
- Compile-time detection of invalid assignments to constants.
- No implicit global variables.

`var` is excluded from v0.1.

### Primitive types

The initial embedded type system should support:

| Type | Description |
|---|---|
| `boolean` | Boolean value |
| `number` | Numeric value, with documented embedded representation |
| `string` | Compile-time/static strings |
| `void` | No return value |

Because embedded hardware has limited resources, numeric representation must be explicit in the compiler's semantic model.

Initial numeric representation:

- Integer literals: select a suitable signed or unsigned integer representation.
- Floating-point literals: `float32`.
- Explicit numeric type annotations should be supported.
- Unsupported conversions must produce compile-time errors.

The compiler must document numeric ranges and overflow behavior.

Do not silently generate unsafe narrowing conversions.

Full JavaScript `Number`, `BigInt`, and arbitrary-precision semantics are not required in v0.1.

### Type annotations

JikaScript may support optional embedded-specific type annotations:

```javascript
let count: uint32 = 100;
let temperature: float32 = 25.5;
let enabled: boolean = true;
```

Supported hardware-oriented types should include:

- `int8`
- `uint8`
- `int16`
- `uint16`
- `int32`
- `uint32`
- `float32`
- `boolean`
- `string`

These annotations are a JikaScript extension, not ECMAScript syntax.

Type inference should be supported where the type is unambiguous.

### Functions

Support ordinary function declarations:

```javascript
function add(a, b) {
    return a + b;
}
```

Support return values and parameters.

Arrow functions:

```javascript
const toggleLED = () => {
    led.toggle();
};
```

Single-expression arrow functions:

```javascript
const square = x => x * x;
```

Arrow functions are particularly important for scheduler callbacks.

Initially, functions must be statically resolvable.

Do not implement dynamic function construction or `eval`.

### Control flow

Support:

- `if`
- `else`
- `switch`
- `while`
- `do...while`
- `for`
- `break`
- `continue`
- `return`

Use ECMAScript operator precedence and syntax.

### Operators

Support the appropriate ECMAScript operators:

Arithmetic:

```javascript
+ - * / %
```

Comparison:

```javascript
< > <= >=
```

Equality:

```javascript
== != === !==
```

Logical:

```javascript
&& || !
```

Assignment:

```javascript
= += -= *= /=
```

Increment/decrement:

```javascript
++ --
```

Bitwise operators should also be supported where appropriate for embedded register and GPIO operations.

Do not collapse `==` and `===` into identical behavior without explicitly defining and documenting the semantics.

JikaScript v0.1 equality semantics:

- `==` and `===` are both strict, non-coercing equality for supported types.
- `!=` and `!==` are both strict inequality.
- Valid comparisons:
  - numeric-to-numeric
  - same-type non-numeric (`boolean` with `boolean`, `string` with `string`)
- Mixed-type comparisons such as `number` vs `string` must be rejected at compile time.

### Arrays

Support **bounded, dynamically sized, homogeneous arrays** backed by static storage.

Example:

```javascript
const readings: int32[8] = [21, 23, 25];

readings.push(27);
const latest = readings.pop();
const count = readings.length;
readings[0] = latest;
```

Requirements:

- Static allocation only (no heap allocation).
- Compile-time known capacity.
- Runtime logical length (`length`) that changes via `push()` and `pop()`.
- Homogeneous element type.
- Indexed reads/writes.
- Bounds and capacity checks.
- Array-literal reassignment is not supported in v0.1.
- Array declarations are restricted to top-level scope in v0.1.

### Array declaration syntax (v0.1)

Use a single declaration form:

```javascript
let values: int32[8] = [1, 2, 3];
```

Rules:

- `T[N]` declares element type `T` and fixed capacity `N`.
- `N` must be a positive integer literal.
- Initializer length may be less than or equal to capacity.
- Empty arrays require explicit type and capacity:

```javascript
let queue: uint16[16] = [];
```

Array capacity inference is allowed only when no explicit capacity is declared:

```javascript
let values = [1, 2, 3]; // inferred capacity = 3
```

### Array operations (v0.1)

Supported:

- Array literals: `[10, 20, 30]`
- Indexed reads: `values[i]`
- Indexed writes: `values[i] = x`
- Length property: `values.length`
- `values.push(x)`
- `values.pop()`

Not supported in v0.1:

- Sparse arrays
- Mixed-type arrays
- Automatic resizing
- Prototype behavior
- `shift()` / `unshift()`

### Array safety and failure semantics

The compiler should reject statically provable invalid cases (for example, initializer length above declared capacity, or literal index out of bounds).

For runtime-invalid operations, generated C must fail predictably using a minimal fail-fast trap function (non-returning infinite loop):

- `push()` on a full array.
- `pop()` on an empty array.
- Read with invalid index.
- Write with invalid index.

This avoids undefined behavior and out-of-bounds memory access without introducing exceptions or heap-based error systems.

### Strings

Strings should initially be immutable, statically allocated values.

Example:

```javascript
const message = "Hello JikaScript";
```

Strings should be stored in read-only memory where supported by the target toolchain.

Dynamic string allocation and garbage collection are excluded.

---

# 5. Cooperative Scheduling Model

Cooperative periodic task scheduling is the primary programming model of JikaScript.

There will be one task model in v0.1.

JikaScript will reuse the standalone C scheduler described in Section 6.

## 5.1 Task syntax

Proposed syntax:

```javascript
task("blink", 1000, () => {
    led.toggle();
});
```

Arguments:

1. Task name.
2. Execution interval in milliseconds.
3. Callback function.

## 5.2 Multiple tasks

Example:

```javascript
const led = pin("PC1");
const button = pin("PD2");

task("blink", 500, () => {
    led.toggle();
});

task("button", 20, () => {
    if (button.read() == LOW) {
        led.toggle();
    }
});

scheduler.start();
```

The scheduler invokes each callback when its interval expires.

## 5.3 Scheduling semantics

- Tasks execute in the main execution context.
- Tasks are periodically invoked.
- Tasks are cooperative.
- A callback runs until it returns.
- No preemptive task switching.
- No separate task stacks.
- No mandatory dynamic allocation.
- Task registration should be statically resolvable at compile time.

A long-running callback delays the execution of other due callbacks. This must be documented.

## 5.4 Important distinction

The v0.1 task API is based on periodic callbacks.

The following are not part of v0.1:

- `async` functions.
- `await`.
- Promises.
- Arbitrary task suspension and resumption.
- Stackful coroutines.
- Preemptive threads.

Do not implement these features as part of the initial scheduler.

## 5.5 Scheduler initialization

The language-level API:

```javascript
scheduler.start();
```

The compiler generates the appropriate C runtime initialization, task registration and main execution loop.

The generated program must not require users to manually implement scheduler internals.

---

# 6. Standalone Periodic Scheduler Library

This is a separate project component and must remain independently usable.

## 6.1 Requirements

The scheduler must:

- Be written in pure C.
- Target C99 compatibility.
- Work independently of JikaScript.
- Support periodic callback execution.
- Use function pointers for callbacks.
- Use static allocation.
- Avoid mandatory heap allocation.
- Have configurable task capacity.
- Support hardware timer integration.
- Be portable across microcontrollers.
- Have no dependency on CH32fun in its core scheduling logic.

CH32fun-specific timer support should be implemented as a separate platform integration layer.

## 6.2 Proposed API

The following API is a starting point. The implementation agent should preserve the behavior of the existing scheduler while adapting its interface to pure C.

Header:

```c
#ifndef PERIODIC_H
#define PERIODIC_H

#include <stdint.h>
#include <stdbool.h>

typedef void (*periodic_callback_t)(void);
typedef void (*periodic_critical_section_fn)(void);

typedef struct {
    periodic_callback_t callback;
    uint32_t interval_ms;
    uint32_t last_run_ms;
    bool enabled;
} periodic_task_t;

void periodic_init(void);

bool periodic_task_add(
    periodic_task_t *task,
    periodic_callback_t callback,
    uint32_t interval_ms
);

void periodic_run(void);

uint32_t periodic_millis(void);
void periodic_tick_ms(uint32_t elapsed_ms);
void periodic_set_critical_section(
    periodic_critical_section_fn enter_fn,
    periodic_critical_section_fn exit_fn
);

#endif
```

=> Usage Example:

```c
#include "periodic.h"

void blink(void) {
    gpio_toggle(PC1);
}

int main(void) {
    periodic_task_t blink_task;

    periodic_init();

    periodic_task_add(
        &blink_task,
        blink,
        1000
    );

    while (1) {
        periodic_run();
    }
}
```

The API above is provisional until checked against the original scheduler implementation.


=> Reference Implementation in C++: https://gist.github.com/sayanriju/75df673450f3ac391e5e866cf7913b50 [Do NOT reuse blindly without translating to pure C]

## 6.3 Scheduler implementation requirements

- Use unsigned integer arithmetic for elapsed-time calculations.
- Handle millisecond counter rollover safely.
- Keep interrupt handlers short.
- Do not execute application callbacks inside interrupt context.
- Ensure timer state is accessed safely between interrupt and main contexts.
- Do not advance time inside `periodic_run()`.
- Advance scheduler time from a platform timer source (typically ISR-driven) via `periodic_tick_ms(...)`.
- Define behavior for invalid intervals.
- Define task registration failure behavior.
- Define task removal and reconfiguration behavior if supported.
- Missed-period policy: when multiple intervals elapse between scheduler runs, execute at most one callback per `periodic_run()` and advance the task deadline to the latest elapsed interval boundary (skip catch-up bursts).

The initial implementation does not need advanced scheduling policies.

## 6.4 Scheduler project structure

The scheduler should be independently buildable:

```text
periodic/
├── include/
│   └── periodic.h
├── src/
│   └── periodic.c
├── platforms/
│   └── ch32fun/
│       └── periodic_ch32fun.c
├── examples/
│   ├── blink.c
│   └── multiple_tasks.c
├── tests/
└── README.md
```

The core scheduler must not include JikaScript-specific code.

---

# 7. Hardware API

Hardware functions are exposed through a language-level abstraction.

The compiler maps these functions to target-specific C functions.

## 7.1 GPIO

Example:

```javascript
const led = pin("PC1");

led.output();
led.write(HIGH);
led.toggle();
```

Input:

```javascript
const button = pin("PD2");

button.input();

if (button.read() == LOW) {
    led.toggle();
}
```

The backend must validate pin names and supported operations.

Do not assume all MCUs have identical pin names or peripheral capabilities.

## 7.2 Future peripherals

The following are planned extensions:

- ADC
- PWM
- UART
- SPI
- I2C
- Hardware timers

These are not required for the first compiler milestone.

## 7.3 Hardware abstraction

Separate:

1. Language-level hardware API.
2. Compiler hardware mapping.
3. Target-specific C implementation.

This separation is mandatory for portability.

---

# 8. Compiler Implementation

## 8.1 Language

The compiler must be implemented in Go.

## 8.2 Compilation stages

Implement the following stages:

1. Source file loading.
2. Lexer.
3. Token definitions.
4. Parser.
5. AST construction.
6. Semantic analysis.
7. Type checking.
8. Target-specific code generation.
9. Output file generation.

## 8.3 Initial project structure

```text
jikascript/
├── go.mod
├── cmd/
│   └── jikascript/
│       └── main.go
│
├── internal/
│   ├── lexer/
│   │   ├── lexer.go
│   │   └── token.go
│   │
│   ├── parser/
│   │   └── parser.go
│   │
│   ├── ast/
│   │   └── ast.go
│   │
│   ├── checker/
│   │   └── checker.go
│   │
│   └── codegen/
│       ├── generator.go
│       └── c/
│           └── generator.go
│
├── targets/
│   └── ch32fun/
│       ├── target.go
│       └── gpio.go
│
├── runtime/
│   └── periodic/
│       ├── periodic.h
│       └── periodic.c
│
├── examples/
│   ├── blink.jika
│   └── multiple_tasks.jika
│
├── tests/
│
└── README.md
```

This structure is a starting point. Keep the implementation simple and avoid unnecessary abstractions.

## 8.4 Lexer

The lexer must recognize:

- Identifiers.
- Keywords.
- Numeric literals.
- String literals.
- Boolean literals.
- Operators.
- Punctuation.
- Comments.

Support:

```javascript
// Single-line comment

/*
   Multi-line comment
*/
```

Provide useful source-position information for diagnostics.

## 8.5 Parser

The parser should implement the supported ECMAScript subset.

Use a maintainable parsing strategy, such as recursive descent with Pratt parsing for expressions.

The parser must produce an AST.

Do not generate C directly from the lexer or parser.

## 8.6 Semantic analysis

The checker should validate:

- Variable declarations.
- Scope.
- Constant assignments.
- Function calls.
- Function arguments.
- Return types.
- Function return consistency (no mixed bare/value returns in one function).
- Type compatibility.
- Equality-operand compatibility under the strict v0.1 equality policy.
- Unsupported language constructs.
- Hardware API usage.
- Task declarations.
- Duplicate task names.

Provide meaningful error messages with source locations.

## 8.7 C code generation

Generate valid C99 code.

Requirements:

- No C++ syntax.
- No C++ standard library.
- No classes.
- No lambdas.
- No templates.
- Use ordinary functions and function pointers.
- Use static allocation wherever practical.
- Generate deterministic output.
- Generate valid C identifiers from JikaScript names.
- Avoid identifier collisions.
- Include required runtime and hardware headers.

Arrow callbacks must be converted into ordinary C functions.

Example JikaScript:

```javascript
task("blink", 1000, () => {
    led.toggle();
});
```

Conceptual generated C:

```c
static void jikascript_task_blink(void) {
    gpio_toggle(PC1);
}

int main(void) {
    periodic_init();

    periodic_task_add(
        &jikascript_blink_task,
        jikascript_task_blink,
        1000
    );

    while (1) {
        periodic_run();
    }
}
```

The generated code must be adapted to the final scheduler API.

---

# 9. CLI Specification

The compiler executable is:

```text
jikascript
```

Source extension:

```text
.jika
```

## 9.1 build

Command:

```bash
jikascript build blink.jika
# or, after a transpile step and optional hand-edit:
jikascript build blink.c
```

Purpose:

Generate production firmware from either a `.jika` source file or a generated readable C file.

Requirements:

- Parse and validate `.jika` source when the input is a source file.
- Generate readable C during the transpile step if the input is `.jika`.
- Accept a generated `.c` file as a build input so the generated code may be reviewed or hand-edited before flashing.
- Invoke the configured target C toolchain.
- Link the required runtime and hardware libraries.
- Produce firmware artifacts.
- Report build errors clearly.

The native compiler should use an optimization level suitable for embedded production builds.

Default optimization should initially favor code size, such as GCC `-Os`.

Allow the user to override optimization settings.

Expected output:

```text
build/
├── blink.elf
├── blink.hex
├── blink.bin
└── ...
```

The exact output formats depend on the target toolchain.

Important: the compiler must not sacrifice correctness to minimize generated code.

## 9.2 transpile

Command:

```bash
jikascript transpile blink.jika
```

Purpose:

Generate readable, human-inspectable C source code.

Requirements:

- Preserve clear function names.
- Use sensible indentation.
- Add useful comments where appropriate.
- Keep generated code deterministic.
- Do not apply readability-destroying minification.
- Do not invoke the native firmware compiler unless explicitly requested.
- Allow the resulting C to be checked into a review/edit loop before invoking `build`.

Example:

```bash
jikascript transpile blink.jika -o blink.c
# review/edit blink.c
jikascript build blink.c
```

## 9.3 check

Command:

```bash
jikascript check blink.jika
```

Purpose:

Validate syntax and semantics without generating firmware.

## 9.4 Additional options

Potential options:

```bash
jikascript build blink.jika --target ch32v003
jikascript build blink.c --target ch32v003
jikascript build blink.jika --optimize size
jikascript build blink.c --optimize speed
jikascript build blink.jika -o dist/blink.elf
jikascript transpile blink.jika --target ch32v003
```

The initial release only needs essential functionality. Do not implement every possible option prematurely.

---

# 10. Optimization Strategy

Optimization happens at two levels.

## 10.1 Compiler-level optimization

Safe optimizations may include:

- Constant folding.
- Constant propagation.
- Dead code elimination.
- Unused variable elimination.
- Simplification of constant expressions.
- Static allocation of known objects.
- Elimination of unnecessary runtime calls.

Only perform optimizations that preserve the language's specified semantics.

## 10.2 Native compiler optimization

Use the target GCC toolchain for:

- Instruction selection.
- Register allocation.
- Machine-level optimization.
- Link-time optimization where supported.
- Dead-code elimination.
- Binary size optimization.

The production build should favor small firmware size by default.

The readable `transpile` output should remain easy to inspect.

---

# 11. Portability

The compiler and language core must be independent of the target MCU.

Initial target:

- WCH CH32V003.
- CH32fun.
- RISC-V GCC.
- C99 output.

Potential future targets:

- STM32.
- AVR.
- RP2040.
- ESP32.

Each target should provide:

- Hardware API implementation.
- Pin and peripheral definitions.
- Startup and build configuration.
- Scheduler timer integration.
- Target compiler configuration.

Adding a new target should not require changes to the language grammar or core parser.

---

# 12. Error Handling and Diagnostics

The compiler should produce understandable diagnostics.

Example:

```text
blink.jika:8:12: error:
Unknown hardware pin "PX99"

blink.jika:12:5: error:
Cannot assign to constant variable "led"

blink.jika:18:1: error:
Invalid task interval
```

Requirements:

- File name.
- Line number.
- Column number.
- Error category.
- Helpful explanation.

Avoid generic errors such as "Compilation failed" when a more useful explanation is available.

---

# 13. Testing Requirements

## 13.1 Compiler tests

Test each stage independently:

- Lexer tests.
- Parser tests.
- AST tests.
- Semantic checker tests.
- Code generation tests.
- CLI tests.

## 13.2 Generated C tests

For each example:

1. Compile the `.jika` file.
2. Generate C.
3. Verify generated C syntax.
4. Compile the C with a host compiler where practical.
5. Compile against the target toolchain.
6. Compare behavior with expected output or hardware behavior.

## 13.3 Scheduler tests

Test independently of JikaScript:

- Single periodic task.
- Multiple periodic tasks.
- Different intervals.
- Callback execution.
- Timer rollover.
- Invalid interval.
- Task registration failure.
- Long callback behavior.
- Repeated scheduler execution.

Use a mock timer implementation for host-side tests.

## 13.4 Hardware tests

Initial physical demonstration:

- CH32V003 board.
- LED connected to a supported GPIO pin.
- One periodic blink task.
- Successful firmware flashing.
- Correct LED blinking.

---

# 14. Initial Example Programs

## 14.1 Blink

```javascript
const led = pin("PC1");

led.output();

task("blink", 1000, () => {
    led.toggle();
});

scheduler.start();
```

## 14.2 Multiple tasks

```javascript
const led1 = pin("PC1");
const led2 = pin("PC2");

led1.output();
led2.output();

task("led1", 500, () => {
    led1.toggle();
});

task("led2", 1000, () => {
    led2.toggle();
});

scheduler.start();
```

## 14.3 Conditional task

```javascript
const led = pin("PC1");
const button = pin("PD2");

led.output();
button.input();

task("button_check", 20, () => {
    if (button.read() == LOW) {
        led.toggle();
    }
});

scheduler.start();
```

---

# 15. Explicit Non-Goals for v0.1

The following features are excluded from the initial implementation:

- Full JavaScript compatibility.
- JavaScript interpreter.
- Virtual machine.
- Garbage collector.
- Dynamic object allocation.
- Dynamic array resizing.
- Promises.
- `async` / `await`.
- Preemptive threads.
- Stackful coroutines.
- `eval()`.
- Dynamic imports.
- Reflection.
- Proxies.
- Full JavaScript object model.
- Complex module loading.
- LLVM.
- C++ code generation.

These may be reconsidered in later versions.

---

# 16. Development Milestones

## Milestone 1: Standalone scheduler

- Port existing C++ periodic scheduler to pure C.
- Preserve existing scheduling behavior.
- Define public C API.
- Separate portable core from CH32fun timer integration.
- Create standalone examples.
- Test on CH32V003.

Deliverable: Independently usable C scheduler library.

## Milestone 2: Go compiler foundation

- Create Go module.
- Implement token definitions.
- Implement lexer.
- Implement parser.
- Implement AST.
- Add basic compiler diagnostics.

Deliverable: Compiler that can parse the initial language subset.

## Milestone 3: Language semantics

- Variables.
- Constants.
- Types.
- Functions.
- Arrow functions.
- Expressions.
- Control flow.
- Semantic checker.

Deliverable: Validated JikaScript programs.

## Milestone 4: C transpiler

- Generate C99 code.
- Generate ordinary C functions for callbacks.
- Generate static task registrations.
- Generate initialization code.
- Generate scheduler loop.
- Generate GPIO operations.

Deliverable: Readable, compilable C output.

## Milestone 5: CH32fun integration

- Add CH32V003 target configuration.
- Add GPIO backend.
- Integrate standalone scheduler.
- Invoke RISC-V GCC.
- Produce firmware artifacts.
- Test actual hardware.

Deliverable: Working JikaScript blink firmware.

## Milestone 6: First usable release

- Complete CLI.
- Add error diagnostics.
- Add automated tests.
- Add examples.
- Add installation and usage documentation.

Deliverable: JikaScript v0.1.

---

# 17. Definition of Done

JikaScript v0.1 is complete when:

1. The scheduler exists as an independently usable pure C library.
2. The Go compiler can read `.jika` files.
3. The compiler supports the documented ECMAScript subset.
4. The compiler performs semantic checking.
5. The compiler generates valid C99 code.
6. Arrow callbacks compile into ordinary C functions.
7. Periodic tasks integrate with the standalone scheduler.
8. GPIO operations work on CH32V003 through CH32fun.
9. `jikascript transpile` produces readable C.
10. `jikascript build` produces optimized firmware.
11. The generated firmware runs on a physical CH32V003 board.
12. The generated code has no C++ dependency.

---

# 18. Instructions for the Implementation Agent

Treat this document as the primary implementation specification.

Follow these rules:

1. Implement the project incrementally, milestone by milestone.
2. Do not implement features outside the v0.1 scope without explicit approval.
3. Do not introduce unnecessary frameworks, dependencies, or abstractions.
4. Do not use LLVM.
5. Do not generate C++.
6. Keep the scheduler standalone.
7. Keep the compiler independent of the CH32V003 target wherever possible.
8. Follow ECMAScript semantics for supported language features.
9. Document any intentional deviations from ECMAScript.
10. Write automated tests alongside each implementation milestone.
11. Do not silently invent hardware APIs or scheduler functions. Verify them against the actual library implementations.
12. Keep generated C inspectable and deterministic.

**Important implementation order:**

First port and validate the existing scheduler as a standalone C library. Then build the Go compiler and its C backend. Finally integrate the two using the CH32fun target.

Do not attempt to build the entire language in one step.

The first end-to-end demonstration must be:

```text
blink.jika
    ↓
jikascript build
    ↓
generated C
    ↓
RISC-V GCC + CH32fun
    ↓
CH32V003 firmware
    ↓
Blinking LED
```

**End of specification.**

# Recommended Repo structure

```
JikaScript/
├── AGENTS.md       # How the coding agent works
├── SPEC.md         # What JikaScript must become
├── README.md       # Project overview and getting started
├── go.mod
├── cmd/
├── internal/
├── targets/
├── runtime/
└── examples/
```