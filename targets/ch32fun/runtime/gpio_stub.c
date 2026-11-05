// gpio_stub.c is a host-side test double for targets/ch32fun/runtime/gpio.c.
// It implements the same gpio.h API without requiring the CH32fun framework,
// so generated programs can be C99-compiled and run on the host for testing.
// It is not part of any firmware build.
#include "gpio.h"

#include <stddef.h>

int pin(const char *name) {
    if (name == 0) {
        return -1;
    }
    // Canonical uppercase pin names, mirroring the firmware lookup table.
    static const char *known[] = {
        "PC0", "PC1", "PC2", "PC3", "PC4", "PC5", "PC6", "PC7",
        "PD0", "PD1", "PD2", "PD3", "PD4", "PD5", "PD6", "PD7",
        "PA1", "PA2",
    };
    for (int i = 0; i < (int)(sizeof(known) / sizeof(known[0])); ++i) {
        const char *a = name;
        const char *b = known[i];
        while (*a && *a && *a == *b) {
            ++a;
            ++b;
        }
        if (*a == *b) {
            return i;
        }
    }
    return -1;
}

void gpio_output(int pin_id) { (void)pin_id; }
void gpio_input(int pin_id) { (void)pin_id; }
void gpio_toggle(int pin_id) { (void)pin_id; }
int gpio_read(int pin_id) { (void)pin_id; return 0; }
void gpio_write(int pin_id, int value) { (void)pin_id; (void)value; }
