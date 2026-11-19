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

## PWM

`pwm()` binds a timer channel as a single PWM output. It takes one **string
literal** pin argument.

```javascript
const led = pwm("PC1");

led.set(50);           // 50% duty cycle
led.setFrequency(1000); // optional, default 1000 Hz
```

Supported pins on CH32V003:

- TIM1 default channels: `PD2`, `PA1`, `PC3`, `PC4`
- TIM2 partial remap 2 channels: `PC1`, `PD3`, `PC0`, `PD7`

TIM2 is programmed into partial remap 2 so `PC1` is available. The trade-off is
that `PD4` (TIM2 channel 1 in the default mapping) is not a PWM pin in v1. `PD7`
is the NRST pin, so using it requires disabling NRST on the target.

A pwm object supports exactly two methods:

| Method | Arguments | Description |
| --- | --- | --- |
| `.set(duty)` | 0–100 | Set the duty cycle as a percentage. |
| `.setFrequency(hz)` | hertz | Set the PWM frequency (default 1000 Hz). |

---

## ADC

`adc()` binds an analog input. It takes one **string literal** pin argument.

```javascript
const sensor = adc("PD4");

let value = sensor.read();
```

Supported pins on CH32V003 (the eight external channels A0–A7): `PA2`, `PA1`,
`PC4`, `PD2`, `PD3`, `PD5`, `PD6`, `PD4`.

An adc object supports exactly one method:

| Method | Arguments | Returns | Description |
| --- | --- | --- | --- |
| `.read()` | none | `int` | Single-ended conversion, raw 10-bit sample (0–1023). |

---

## Timer and Clock Assumptions

The cooperative scheduler and the PWM backend share one time base: the core
clock selected in `funconfig.h`.

- The scheduler runs from SysTick with a 1 ms tick (`periodic_millis()`,
  `periodic_tick_ms()`); `periodic_ch32fun.c` derives the tick from
  `DELAY_MS_TIME`.
- PWM frequencies are derived from `FUNCONF_SYSTEM_CORE_CLOCK` with a fixed
  period of 1000 timer ticks: `freq = core_clock / ((PSC + 1) * 1000)`.

Both must agree on the actual core clock. If `funconfig.h` selects a clock that
differs from `FUNCONF_SYSTEM_CORE_CLOCK`, scheduler milliseconds and PWM
frequencies are scaled by the same error.

### Limits

- PWM duty is a percentage (0–100) with a fixed 1000-step period.
- The PWM frequency range is bounded by the 16-bit prescaler; requests outside
  that range are clamped.
- `PD7` is PWM-capable but is also the NRST pin; disable NRST to use it.
- `PD4` is not a PWM pin because TIM2 is remapped to expose `PC1`.

---

## Compile-Time Validation

Argument counts and types are checked when the program is transpiled, so
erroneous API use fails with a clean error instead of reaching the C compiler
and breaking the firmware build. `uart.write`/`println` take a string;
`pwm.set`/`setFrequency` and `pin.write` take a number; `read` takes none. A
peripheral handle is opaque: it cannot be assigned to a scalar binding or
passed as a value argument to another peripheral method.

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
