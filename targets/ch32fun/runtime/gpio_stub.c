// gpio_stub.c is a host-side test double for targets/ch32fun/runtime/gpio.c.
// It implements the same gpio.h API without requiring the CH32fun framework,
// so generated programs can be C99-compiled and run on the host for testing.
// It is not part of any firmware build.
#include "gpio.h"

void gpio_ch32fun_init(void) {}

void gpio_output(int pin_id) { (void)pin_id; }
void gpio_input(int pin_id) { (void)pin_id; }
void gpio_toggle(int pin_id) { (void)pin_id; }
int gpio_read(int pin_id) { (void)pin_id; return 0; }
void gpio_write(int pin_id, int value) { (void)pin_id; (void)value; }
