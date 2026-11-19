# Peripherals

GPIO, UART, PWM, and ADC are first-class builtins. Each constructor returns a
handle, and the handle's methods do the work — no register bit-poking.

| Peripheral | Constructor | Reference |
| --- | --- | --- |
| GPIO | `pin("PC1")` | [GPIO](gpio.md) |
| UART | `uart("PD5", "PD6")` | [UART](uart.md) |
| PWM | `pwm("PC1")` | [PWM](pwm.md) |
| ADC | `adc("PD4")` | [ADC](adc.md) |

```javascript
const led = pin("PC1");
const serial = uart("PD5", "PD6");
const dimmer = pwm("PC1");
const sensor = adc("PD4");

led.output();
serial.println("boot");
dimmer.set(50);
let level = sensor.read();
```

---

## Compile-Time Validation

Argument counts and types are checked when the program is transpiled, so
erroneous API use fails with a clean error instead of reaching the C compiler
and breaking the firmware build. `uart.write`/`println` take a string;
`pwm.set`/`setFrequency` and `pin.write` take a number; `read()` takes none. A
peripheral handle is opaque: it cannot be assigned to a scalar binding or
passed as a value argument to another peripheral method. Method calls require a
peripheral receiver — a `pin()`, `uart()`, `pwm()`, or `adc()` binding, or a
constructor used directly — and scalars have no properties or methods.

Everything compiles down to direct register writes through the CH32fun backend —
fast, efficient, and direct.

Curious how it all turns into C? Check out
[How Compilation Works](../under-the-hood.md)!
