#ifndef JIKASCRIPT_CH32FUN_PWM_H
#define JIKASCRIPT_CH32FUN_PWM_H

int pwm_open(int pin);
void pwm_set(int handle, int duty);
void pwm_set_frequency(int handle, int freq_hz);

#endif
