# PWM

`pwm()` binds a timer channel as a single PWM output. It takes one **string
literal** pin argument.

```javascript
const led = pwm("PC1");

led.set(50);             // 50% duty cycle
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

See also: [GPIO](gpio.md) · [UART](uart.md) · [ADC](adc.md) ·
[Peripherals overview](hardware.md)
