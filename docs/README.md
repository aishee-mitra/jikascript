# Welcome to JikaScript! 👋

Ever wanted to write embedded microcontroller firmware with the friendliness and clean ergonomics of JavaScript, but without the baggage of heavy runtimes, virtual machines, or garbage collectors?

That's **JikaScript**!

JikaScript brings modern ECMAScript syntax directly to tiny microcontrollers (starting with the ultra-low-cost WCH CH32V003 RISC-V chip). You write clean, expressive `.jika` code, and the JikaScript compiler translates it directly into readable, lightning-fast C99.

---

## Why You'll Love JikaScript

- **Zero VM Overhead:** There is no heavy JS engine on the chip. Your code compiles directly to native C99 and machine code.
- **Predictable & Safe:** Zero dynamic heap allocation by default. Everything has static capacity and bounds-checking.
- **Familiar Syntax:** Variables, arrow functions, object literals, and arrays feel just like modern JavaScript.
- **Built-in Cooperative Multitasking:** Forget juggling raw timer interrupts. Schedule periodic tasks cleanly with simple `task(...)` calls.
- **Inspectable C Code:** Transpile anytime to see the clean, human-readable C code emitted under the hood.

---

## A Taste of JikaScript

Here's how simple it is to blink an LED:

```javascript
const led = pin("PC1");

led.output();

task("blink", 1000, () => {
    led.toggle();
});

scheduler.start();
```

No obscure register bit-manipulation, no confusing ISR declarations — just clean, readable logic that compiles down to tiny microcontroller firmware.

Ready to dive in? Let's get [Blink running in 5 minutes](quickstart.md)!
