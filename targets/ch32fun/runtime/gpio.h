#ifndef MICROJS_CH32FUN_GPIO_H
#define MICROJS_CH32FUN_GPIO_H

int pin(const char *name);
void gpio_output(int pin_id);
void gpio_input(int pin_id);
void gpio_toggle(int pin_id);
int gpio_read(int pin_id);
void gpio_write(int pin_id, int value);

#endif
