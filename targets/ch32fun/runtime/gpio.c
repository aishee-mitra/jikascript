#include "gpio.h"

#include <string.h>

#include "ch32fun.h"

enum {
    MICROJS_PIN_INVALID = -1,
    MICROJS_PIN_PC0 = 0,
    MICROJS_PIN_PC1,
    MICROJS_PIN_PC2,
    MICROJS_PIN_PC3,
    MICROJS_PIN_PC4,
    MICROJS_PIN_PC5,
    MICROJS_PIN_PC6,
    MICROJS_PIN_PC7,
    MICROJS_PIN_PD0,
    MICROJS_PIN_PD1,
    MICROJS_PIN_PD2,
    MICROJS_PIN_PD3,
    MICROJS_PIN_PD4,
    MICROJS_PIN_PD5,
    MICROJS_PIN_PD6,
    MICROJS_PIN_PD7,
    MICROJS_PIN_PA1,
    MICROJS_PIN_PA2
};

static int resolve_fun_pin(int pin_id) {
    switch (pin_id) {
        case MICROJS_PIN_PC0: return PC0;
        case MICROJS_PIN_PC1: return PC1;
        case MICROJS_PIN_PC2: return PC2;
        case MICROJS_PIN_PC3: return PC3;
        case MICROJS_PIN_PC4: return PC4;
        case MICROJS_PIN_PC5: return PC5;
        case MICROJS_PIN_PC6: return PC6;
        case MICROJS_PIN_PC7: return PC7;
        case MICROJS_PIN_PD0: return PD0;
        case MICROJS_PIN_PD1: return PD1;
        case MICROJS_PIN_PD2: return PD2;
        case MICROJS_PIN_PD3: return PD3;
        case MICROJS_PIN_PD4: return PD4;
        case MICROJS_PIN_PD5: return PD5;
        case MICROJS_PIN_PD6: return PD6;
        case MICROJS_PIN_PD7: return PD7;
        case MICROJS_PIN_PA1: return PA1;
        case MICROJS_PIN_PA2: return PA2;
        default: return -1;
    }
}

int pin(const char *name) {
    if (name == 0) {
        return MICROJS_PIN_INVALID;
    }

    if (strcmp(name, "PC0") == 0) return MICROJS_PIN_PC0;
    if (strcmp(name, "PC1") == 0) return MICROJS_PIN_PC1;
    if (strcmp(name, "PC2") == 0) return MICROJS_PIN_PC2;
    if (strcmp(name, "PC3") == 0) return MICROJS_PIN_PC3;
    if (strcmp(name, "PC4") == 0) return MICROJS_PIN_PC4;
    if (strcmp(name, "PC5") == 0) return MICROJS_PIN_PC5;
    if (strcmp(name, "PC6") == 0) return MICROJS_PIN_PC6;
    if (strcmp(name, "PC7") == 0) return MICROJS_PIN_PC7;
    if (strcmp(name, "PD0") == 0) return MICROJS_PIN_PD0;
    if (strcmp(name, "PD1") == 0) return MICROJS_PIN_PD1;
    if (strcmp(name, "PD2") == 0) return MICROJS_PIN_PD2;
    if (strcmp(name, "PD3") == 0) return MICROJS_PIN_PD3;
    if (strcmp(name, "PD4") == 0) return MICROJS_PIN_PD4;
    if (strcmp(name, "PD5") == 0) return MICROJS_PIN_PD5;
    if (strcmp(name, "PD6") == 0) return MICROJS_PIN_PD6;
    if (strcmp(name, "PD7") == 0) return MICROJS_PIN_PD7;
    if (strcmp(name, "PA1") == 0) return MICROJS_PIN_PA1;
    if (strcmp(name, "PA2") == 0) return MICROJS_PIN_PA2;

    return MICROJS_PIN_INVALID;
}

void gpio_output(int pin_id) {
    int p = resolve_fun_pin(pin_id);
    if (p < 0) {
        return;
    }
    funPinMode(p, GPIO_CFGLR_OUT_10Mhz_PP);
}

void gpio_input(int pin_id) {
    int p = resolve_fun_pin(pin_id);
    if (p < 0) {
        return;
    }
    funPinMode(p, GPIO_CFGLR_IN_FLOATING);
}

void gpio_toggle(int pin_id) {
    int p = resolve_fun_pin(pin_id);
    if (p < 0) {
        return;
    }
    funDigitalWrite(p, !funDigitalRead(p));
}

int gpio_read(int pin_id) {
    int p = resolve_fun_pin(pin_id);
    if (p < 0) {
        return 0;
    }
    return (int)funDigitalRead(p);
}

void gpio_write(int pin_id, int value) {
    int p = resolve_fun_pin(pin_id);
    if (p < 0) {
        return;
    }
    funDigitalWrite(p, value ? FUN_HIGH : FUN_LOW);
}
