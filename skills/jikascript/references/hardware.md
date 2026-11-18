# GPIO & Hardware Reference

Hardware functions are exposed through a language-level abstraction. The
compiler maps these functions to target-specific C functions (CH32fun on
CH32V003). The language core stays independent of the specific MCU.

## Pins

Select a pin by its hardware name (e.g. `"PC1"`, `"PD2"`):

```javascript
const led = pin("PC1");
const button = pin("PD2");
```

`pin` is a reserved name. The backend validates pin names and supported
operations. Do not assume all MCUs have identical pin names or peripheral
capabilities.

## Configuring pin direction

```javascript
led.output();    // make it an output
button.input();  // make it an input (with pull-up where applicable)
```

## Writing outputs

```javascript
led.write(HIGH);  // drive pin high
led.write(LOW);   // drive pin low
led.toggle();     // toggle pin state (output)
```

## Reading inputs

```javascript
let state = button.read();
if (state == LOW) {
    // button pressed (active-low)
}
```

`HIGH` and `LOW` are predefined GPIO state constants. They are compiler-provided
values with fixed meanings: `HIGH == 1` and `LOW == 0`.

Treat them as built-in logic constants, not ordinary mutable user variables.
Compare reads with the strict equality operators (`==` / `===`), which are
non-coercing.

## UART

`uart()` binds the CH32V003 USART1 peripheral. It takes two string-literal
arguments, TX then RX, and the compiler rejects any pair other than the default
`uart("PD5", "PD6")`.

```javascript
const serial = uart("PD5", "PD6");

serial.write("hello");
serial.println("ok");
let ch = serial.read();
```

The configuration is fixed at 8N1, 115200 baud. A uart object supports exactly
three methods: `write(text)`, `println(text)` (adds a CRLF line ending), and
`read()`. `read()` is non-blocking and returns `-1` when no byte is available.

## Reserved names

`pin`, `uart`, `HIGH`, and `LOW` are reserved or special built-ins; do not use
them as variable names.

## Complete interactive example: button toggle

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

## Future peripherals (planned, not in v0.1)

ADC, PWM, SPI, I2C, and hardware timers are planned extensions, not required for
the first compiler milestone.

## Hardware abstraction layers

1. Language-level hardware API.
2. Compiler hardware mapping.
3. Target-specific C implementation.

This separation is mandatory for portability.