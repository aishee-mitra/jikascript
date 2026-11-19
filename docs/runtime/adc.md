# ADC

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

See also: [GPIO](gpio.md) · [UART](uart.md) · [PWM](pwm.md) ·
[Peripherals overview](hardware.md)
