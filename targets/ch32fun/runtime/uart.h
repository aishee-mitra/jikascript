#ifndef JIKASCRIPT_CH32FUN_UART_H
#define JIKASCRIPT_CH32FUN_UART_H

int uart_open(int tx_pin, int rx_pin);
void uart_write(int handle, const char *text);
void uart_println(int handle, const char *text);
int uart_read(int handle);

#endif
