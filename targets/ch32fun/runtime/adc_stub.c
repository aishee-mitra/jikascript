// adc_stub.c is a host-side test double for targets/ch32fun/runtime/adc.c.
// It implements the same adc.h API without requiring the CH32fun framework,
// so generated programs can be C99-compiled and run on the host for testing.
// It is not part of any firmware build.
#include "adc.h"

int adc_open(int pin) { (void)pin; return 0; }
int adc_read(int handle) { (void)handle; return 0; }
