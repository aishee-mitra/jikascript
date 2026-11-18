# Arrays

In standard JavaScript, arrays grow dynamically and can hold mixed types. On microcontrollers, unbounded dynamic growth can fragment memory and cause crashes.

JikaScript introduces **Bounded Arrays**: arrays with a fixed maximum capacity and a mutable logical length.

---

## Declaring an Array

Declare an array by specifying its element type and maximum capacity in brackets:

```javascript
// A bounded array of int32 values, maximum capacity 8
let readings: int32[8] = [21, 23, 25];
```

In this example:
- **Capacity:** Can store up to 8 elements.
- **Initial Length:** Currently holds 3 elements.

---

## Array Methods: `push()` and `pop()`

You get the convenient methods you know from JavaScript:

```javascript
let queue: int32[4] = [10, 20];

// Add an element to the end:
queue.push(30);

// Pop the last element:
const latest = queue.pop(); // Returns 30

// Read the current logical length:
const count = queue.length; // Returns 2
```

---

## Safety & Bounds Checking

JikaScript keeps your firmware safe:
- **Capacity Checking:** Calling `push()` on a full array is checked in the generated C.
- **Bounds Checking:** Index access (like `queue[i]`) and `pop()` operations are bounds-checked in the generated C.
- **Homogeneous Elements:** Every element must match the declared type.

A failed check calls the generated `jikascript_fail()` helper, which halts the
device in an infinite loop. That is deliberate: on a microcontroller there is no
exception handler or console to report to, so stopping is safer than continuing
with corrupted state. Treat a hang as a bounds violation and fix the logic.

Constant indices are validated at compile time, so `queue[99]` on a
capacity-8 array is a compiler error rather than a runtime halt.

Next: [Control Flow](control-flow.md) in JikaScript.
