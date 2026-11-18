#include "uart.h"

#include <ch32fun.h>

/*
 * CH32V003 USART1 backend.
 *
 * The compiler validates the pin pair (PD5/PD6 on the default USART1 mapping)
 * and emits JIKA_PIN_* constants, so this file only programs the peripheral.
 * The configuration is fixed at 8N1, 115200 baud, matching the ch32fun UART
 * examples. Reads are non-blocking: uart_read() returns -1 when no byte is
 * available so a cooperative task never stalls waiting for input.
 */

#define JIKA_UART_BAUD 115200U

static int uart_ready;

static void uart_configure(int tx_pin, int rx_pin) {
    /* Both default USART1 pins live on port D. */
    RCC->APB2PCENR |= RCC_APB2Periph_GPIOD | RCC_APB2Periph_USART1;

    funPinMode(tx_pin, GPIO_CFGLR_OUT_10Mhz_AF_PP);
    funPinMode(rx_pin, GPIO_CFGLR_IN_FLOAT);

    USART1->CTLR1 = USART_WordLength_8b | USART_Parity_No | USART_Mode_Tx | USART_Mode_Rx;
    USART1->CTLR2 = USART_StopBits_1;
    USART1->CTLR3 = USART_HardwareFlowControl_None;
    USART1->BRR = (FUNCONF_SYSTEM_CORE_CLOCK + JIKA_UART_BAUD / 2U) / JIKA_UART_BAUD;
    USART1->CTLR1 |= CTLR1_UE_Set;
}

int uart_open(int tx_pin, int rx_pin) {
    if (!uart_ready) {
        uart_configure(tx_pin, rx_pin);
        uart_ready = 1;
    }
    return 0;
}

void uart_write(int handle, const char *text) {
    (void)handle;
    if (text == 0) {
        return;
    }
    while (*text != '\0') {
        while ((USART1->STATR & USART_STATR_TXE) == 0) {
        }
        USART1->DATAR = (uint8_t)*text++;
    }
}

void uart_println(int handle, const char *text) {
    uart_write(handle, text);
    uart_write(handle, "\r\n");
}

int uart_read(int handle) {
    (void)handle;
    if ((USART1->STATR & USART_STATR_RXNE) == 0) {
        return -1;
    }
    return (int)(USART1->DATAR & 0xFFU);
}
