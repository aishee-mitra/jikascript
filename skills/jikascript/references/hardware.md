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

## PWM

`pwm()` binds a timer channel as a single PWM output. It takes one string-literal
pin argument.

```javascript
const led = pwm("PC1");
led.set(50);
led.setFrequency(1000);
```

Supported pins on CH32V003: TIM1 default channels `PD2`, `PA1`, `PC3`, `PC4`,
and TIM2 partial remap 2 channels `PC1`, `PD3`, `PC0`, `PD7`. TIM2 is remapped so
`PC1` is available; `PD4` (default TIM2 channel 1) is therefore not a PWM pin in
v1. A pwm object supports exactly two methods: `set(duty)` (0-100 percent) and
`setFrequency(hz)` (default 1000 Hz).

## ADC

`adc()` binds an analog input. It takes one string-literal pin argument.

```javascript
const sensor = adc("PD4");
let value = sensor.read();
```

Supported pins on CH32V003 (channels A0-A7): `PA2`, `PA1`, `PC4`, `PD2`, `PD3`,
`PD5`, `PD6`, `PD4`. An adc object supports exactly one method: `read()`, which
returns the raw 10-bit sample (0-1023).

## Reserved names

`pin`, `uart`, `pwm`, `adc`, `HIGH`, and `LOW` are reserved or special built-ins;
do not use them as variable names.

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