# Variables & Types

JikaScript gives you the expressive flexibility of JavaScript declarations with the rock-solid type safety of C.

---

## Declaring Variables: `const` and `let`

Just like in modern JavaScript, use `const` for values that won't change, and `let` for variables you intend to reassign:

```javascript
const maxRetries = 3;
let currentAttempt = 0;

currentAttempt = currentAttempt + 1;
```

---

## Automatic Type Inference

Most of the time, you don't even need to write type annotations! JikaScript automatically infers the types from your values and maps them directly to efficient C primitives:

```javascript
let sensorName = "DHT22";  // Inferred as string (const char *)
let isReady = true;        // Inferred as boolean (bool)
let reading = 42;          // Inferred as int32_t
let voltage = 3.3;         // Inferred as float
```

---

## Explicit Type Annotations

When you're building firmware, exact bit-widths matter (for registers, memory packing, or hardware protocols). JikaScript lets you specify explicit types using TypeScript-style annotations:

```javascript
let dutyCycle: uint8 = 128;      // maps to uint8_t
let offset: int16 = -500;        // maps to int16_t
let counter: uint32 = 100000;    // maps to uint32_t
let tempCelsius: float32 = 24.5; // maps to float
```

Supported primitive types:
- **Integers:** `int8`, `uint8`, `int16`, `uint16`, `int32`, `uint32`
- **Floats:** `float32`
- **Others:** `boolean`, `string`

---

## Strict Equality: No JavaScript Coercion Surprises! 🚫

One of the trickiest parts of standard JavaScript is implicit type coercion (like `"5" == 5`). 

In JikaScript, **safety comes first**:
- `==` and `===` both perform **strict equality checks**.
- Comparing values of different types (like a `number` to a `string`) is caught at **compile time** with an error!

```javascript
let count = 5;

if (count == 5) {
    // Works as expected!
}

// Compile error! Mixed-type comparisons are rejected:
// if (count == "5") { ... }
```

Next, let's explore how [Functions](language/functions.md) work in JikaScript!
