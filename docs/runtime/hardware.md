# GPIO & Hardware

Controlling physical pins in JikaScript is as simple as calling methods on an object.

```javascript
const led = pin("PC1");
const button = pin("PD2");

led.output();
button.input();

led.write(HIGH);
let pressed = button.read() == LOW;
```

---

## Binding a Pin

`pin()` binds a hardware pin to a variable. The argument must be a **string
literal** naming a pin the target supports — the compiler rejects variables,
expressions, and unknown pin names.

```javascript
const led = pin("PC1");
```

Names are case-insensitive and whitespace-trimmed, so `pin("pc1")` and
`pin(" PC1 ")` both resolve to `PC1`.

**Available pins on CH32V003:** `PA1`, `PA2`, `PC0`–`PC7`, `PD0`–`PD7`.

Because the name is known at compile time, `pin("PC1")` becomes the constant
`JIKA_PIN_PC1` in the generated C. There is no runtime string lookup and no
pin-name switch in the firmware.

---

## Pin Methods

A pin object supports exactly these five methods. Any other method name is a
compile error.

| Method | Arguments | Returns | Description |
| --- | --- | --- | --- |
| `.output()` | none | — | Configure the pin as a push-pull output. |
| `.input()` | none | — | Configure the pin as a floating input. |
| `.write(value)` | optional | — | Drive an output pin to `HIGH` or `LOW`. |
| `.read()` | none | `int` | Read the pin level: `1` for high, `0` for low. |
| `.toggle()` | none | — | Invert the current output level. |

`write()` accepts any value: non-zero drives the pin high, zero drives it low.
Calling `write()` with no argument writes `LOW`.

Passing arguments to any other method is a compile error.

---

## Pin State Constants

`HIGH` and `LOW` are compiler-provided constants with the fixed meanings
`HIGH == 1` and `LOW == 0`. They are not ordinary variables — you cannot
reassign or shadow them. If you want a named alias, create a separate binding
such as `const ledOn = HIGH`.

---

## UART

`uart()` binds the CH32V003 USART1 peripheral. It takes two **string literal**
arguments, TX then RX, and the compiler rejects any pair other than the default
`uart("PD5", "PD6")`.

```javascript
const serial = uart("PD5", "PD6");

serial.write("hello");
serial.println("ok");

let ch = serial.read();
```

The configuration is fixed at 8N1, 115200 baud. A uart object supports exactly
three methods:

| Method | Arguments | Returns | Description |
| --- | --- | --- | --- |
| `.write(text)` | string | — | Transmit a string. |
| `.println(text)` | string | — | Transmit a string followed by `\r\n`. |
| `.read()` | none | `int` | Return the next received byte, or `-1` if none is available. |

`read()` is non-blocking, so it is safe to poll from a task. Any other method
name, or the wrong number of arguments, is a compile error.

---

## Example: Button-Controlled LED

Poll a push-button every 20 ms and drive an LED to match its state:

```javascript
const led = pin("PC1");
const button = pin("PD2");

led.output();
button.input();

task("button_check", 20, () => {
    if (button.read() == LOW) {
        led.write(HIGH);
    } else {
        led.write(LOW);
    }
});

scheduler.start();
```

Everything compiles down to direct register writes through the CH32fun backend —
fast, efficient, and direct.

Curious how it all turns into C? Check out [How Compilation Works](../under-the-hood.md)!
