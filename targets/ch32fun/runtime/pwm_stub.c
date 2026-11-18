// pwm_stub.c is a host-side test double for targets/ch32fun/runtime/pwm.c.
// It implements the same pwm.h API without requiring the CH32fun framework,
// so generated programs can be C99-compiled and run on the host for testing.
// It is not part of any firmware build.
#include "pwm.h"

int pwm_open(int pin) { (void)pin; return 0; }
void pwm_set(int handle, int duty) { (void)handle; (void)duty; }
void pwm_set_frequency(int handle, int freq_hz) { (void)handle; (void)freq_hz; }
