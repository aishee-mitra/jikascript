// uart_stub.c is a host-side test double for targets/ch32fun/runtime/uart.c.
// It implements the same uart.h API without requiring the CH32fun framework,
// so generated programs can be C99-compiled and run on the host for testing.
// It is not part of any firmware build.
#include "uart.h"

int uart_open(int tx_pin, int rx_pin) { (void)tx_pin; (void)rx_pin; return 0; }
void uart_write(int handle, const char *text) { (void)handle; (void)text; }
void uart_println(int handle, const char *text) { (void)handle; (void)text; }
int uart_read(int handle) { (void)handle; return -1; }
