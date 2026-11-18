#include "adc.h"

#include <ch32fun.h>

/*
 * CH32V003 ADC backend.
 *
 * The compiler validates the pin against targets/ch32fun/peripherals.go and
 * emits a JIKA_PIN_* constant. The CH32V003 ADC has eight external channels
 * (A0..A7) and returns 10-bit samples. Initialization uses ch32fun's
 * funAnalogInit() to set the sample time and calibrate; each read selects the
 * handle's channel and runs the regular conversion path.
 */

#define JIKA_ADC_MAX_HANDLES 4

static int adc_used[JIKA_ADC_MAX_HANDLES];
static int adc_channels[JIKA_ADC_MAX_HANDLES];
static int adc_ready;

static int adc_lookup(int pin) {
    switch (pin) {
    case PA2: return 0;
    case PA1: return 1;
    case PC4: return 2;
    case PD2: return 3;
    case PD3: return 4;
    case PD5: return 5;
    case PD6: return 6;
    case PD4: return 7;
    default: return -1;
    }
}

int adc_open(int pin) {
    int channel = adc_lookup(pin);
    if (channel < 0) {
        return -1;
    }
    if (!adc_ready) {
        funAnalogInit();
        adc_ready = 1;
    }
    funPinMode(pin, GPIO_CFGLR_IN_ANALOG);
    for (int i = 0; i < JIKA_ADC_MAX_HANDLES; i++) {
        if (!adc_used[i]) {
            adc_used[i] = 1;
            adc_channels[i] = channel;
            return i;
        }
    }
    return -1;
}

int adc_read(int handle) {
    if (handle < 0 || handle >= JIKA_ADC_MAX_HANDLES || !adc_used[handle]) {
        return 0;
    }
    return funAnalogRead(adc_channels[handle]);
}
