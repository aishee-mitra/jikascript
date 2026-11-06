# Quickstart: Blink in 5 Minutes ⚡

Let's walk through your very first JikaScript program from start to finish!

---

## 1. Writing Your Code

Create a file named `blink.jika`:

```javascript
// Grab pin PC1 (often the built-in LED on dev boards)
const led = pin("PC1");

// Configure it as a digital output
led.output();

// Create a periodic task named "blink" running every 1000ms
task("blink", 1000, () => {
    led.toggle();
});

// Kick off the cooperative scheduler
scheduler.start();
```

Look familiar? It feels just like JavaScript, but this is real embedded code!

---

## 2. Checking & Transpiling

Before flashing any hardware, you can instantly validate your syntax and check out the generated C code with the `jikascript` CLI:

### Validate Syntax & Types
```bash
jikascript check blink.jika
```
If anything is typed incorrectly or out of bounds, the checker gives you clear, friendly compiler messages.

### Peek at the Generated C
Want to see how clean the generated C is? Run:

```bash
jikascript transpile blink.jika -o blink.c
```

You'll get clean, standard C99 that sets up the pin, registers the periodic task callback, and initializes the scheduler.

---

## 3. Building Firmware (CH32V003)

When you're ready to build for the CH32V003 using CH32fun:

```bash
# Optional: clone ch32fun if you haven't yet
just get-ch32libs
export JIKASCRIPT_CH32FUN="$(cd third_party/ch32fun && pwd)"

# Build flashable firmware (or use: just build)
jikascript build blink.jika --target ch32v003 --optimize size
```

*(Tip: If you use [`just`](https://github.com/casey/just), you can simply run `just build` or `just transpile`!)*

This produces ready-to-flash binaries: `build/blink.bin`, `build/blink.hex`, and `build/blink.elf`.

---

## What's Next?

Now that you've seen the basics, let's explore the language features in detail:
- Learn about [Variables & Types](language/variables-and-types.md)
- Write [Functions](language/functions.md) with smart type inference
- Work with [Objects & Records](language/objects.md) and [Arrays](language/arrays.md)
