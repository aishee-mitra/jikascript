#include "gpio.h"

#include <ch32fun.h>

/*
 * GPIO operations for the CH32V003 backend.
 *
 * Pin identifiers arrive as JIKA_PIN_* constants from jikascript_pins.h, which
 * expand to CH32fun pin constants at compile time. The checker validates pin
 * names before code generation, so no runtime pin lookup or validation is
 * needed here.
 */

void gpio_ch32fun_init(void) {
    SystemInit();
    funGpioInitAll();
}

void gpio_output(int pin_id) {
    funPinMode(pin_id, GPIO_CFGLR_OUT_10Mhz_PP);
}

void gpio_input(int pin_id) {
    funPinMode(pin_id, GPIO_CFGLR_IN_FLOAT);
}

void gpio_toggle(int pin_id) {
    funDigitalWrite(pin_id, !funDigitalRead(pin_id));
}

int gpio_read(int pin_id) {
    return (int)funDigitalRead(pin_id);
}

void gpio_write(int pin_id, int value) {
    funDigitalWrite(pin_id, value ? FUN_HIGH : FUN_LOW);
}
