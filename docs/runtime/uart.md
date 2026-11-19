# UART

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

See also: [GPIO](gpio.md) · [PWM](pwm.md) · [ADC](adc.md) ·
[Peripherals overview](hardware.md)
