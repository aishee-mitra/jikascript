# Reserved Keywords

JikaScript keeps a small set of words reserved for the language itself. These names are not available as ordinary variable names, function names, or other identifiers.

This keeps the language predictable and makes the embedded scheduler syntax read naturally.

---

## Why Some Names Are Reserved

A few built-in constructs use names that look like normal identifiers, but they are actually language syntax:

```javascript
task("blink", 1000, () => {
    // periodic task body
});

scheduler.start();
```

Here, `task` is not a variable name; it is the task declaration syntax. `scheduler` is not a normal object you can shadow; it is the scheduler API. `start` is the special scheduler method name. `include` is the file-include syntax, not a variable name.

---

## Example reserved words

The initial language subset reserves the following names:

- `let`, `const`, `function`, `return`
- `if`, `else`, `while`, `do`, `for`, `switch`, `case`, `default`
- `break`, `continue`
- `true`, `false`
- `include`
- `task`, `pin`, `uart`, `pwm`, `adc`, `scheduler`, `start`
- `HIGH`, `LOW` (predefined GPIO state constants)

`uart`, `pwm`, and `adc` are the peripheral constructors, and `pin` is the GPIO
constructor. See [Peripherals](../runtime/hardware.md) for the methods each one
provides.

`HIGH` and `LOW` are built-in pin-state constants, not ordinary user variables.
They are provided by the compiler and should be treated as fixed values for GPIO
logic. Do not shadow them with your own bindings.

---

## What happens if you try to reuse one?

Using a reserved word as a variable name will fail during parsing or checking.

```javascript
let task = 1;          // invalid
const scheduler = 5;   // invalid
let start = 1;         // invalid
include = 2;           // invalid
```

Use a different name instead:

```javascript
let myTask = 1;
const schedulerState = 5;
let startTime = 0;
```

This rule is part of the language design, not an accident of the compiler. It keeps the syntax simple and prevents ambiguous embedded constructs.

Next up: [Functions](functions.md)
