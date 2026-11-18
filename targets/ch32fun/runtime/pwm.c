#include "pwm.h"

#include <ch32fun.h>

/*
 * CH32V003 PWM backend.
 *
 * The compiler validates the pin against targets/ch32fun/peripherals.go and
 * emits a JIKA_PIN_* constant, so this file maps that constant to a timer
 * channel and programs the peripheral. TIM1 runs in its default mapping and
 * TIM2 in partial remap 2 (CH1=PC1, CH2=PD3, CH3=PC0, CH4=PD7).
 *
 * Duty is a percentage (0..100) and the period is fixed at JIKA_PWM_STEPS
 * timer ticks, so a duty of 50 writes half the period to the compare register.
 *
 * Timing contract: the timer clock is the core clock, the same source the
 * SysTick scheduler uses (DELAY_MS_TIME in periodic_ch32fun.c). Both are
 * derived from FUNCONF_SYSTEM_CORE_CLOCK, so scheduler milliseconds and PWM
 * frequencies stay aligned as long as that macro matches the actual core
 * clock selected in funconfig.h.
 */

#define JIKA_PWM_STEPS 1000U
#define JIKA_PWM_DEFAULT_FREQ_HZ 1000U
#define JIKA_PWM_MAX_HANDLES 4

typedef struct {
    int used;
    int timer;
    int channel;
} pwm_handle_t;

static pwm_handle_t pwm_handles[JIKA_PWM_MAX_HANDLES];
static int tim1_ready;
static int tim2_ready;

static int pwm_lookup(int pin, int *timer, int *channel) {
    switch (pin) {
    case PD2: *timer = 1; *channel = 1; return 1;
    case PA1: *timer = 1; *channel = 2; return 1;
    case PC3: *timer = 1; *channel = 3; return 1;
    case PC4: *timer = 1; *channel = 4; return 1;
    case PC1: *timer = 2; *channel = 1; return 1;
    case PD3: *timer = 2; *channel = 2; return 1;
    case PC0: *timer = 2; *channel = 3; return 1;
    case PD7: *timer = 2; *channel = 4; return 1;
    default: return 0;
    }
}

static void pwm_write_compare(int timer, int channel, uint32_t value) {
    if (timer == 1) {
        switch (channel) {
        case 1: TIM1->CH1CVR = value; break;
        case 2: TIM1->CH2CVR = value; break;
        case 3: TIM1->CH3CVR = value; break;
        case 4: TIM1->CH4CVR = value; break;
        }
    } else {
        switch (channel) {
        case 1: TIM2->CH1CVR = value; break;
        case 2: TIM2->CH2CVR = value; break;
        case 3: TIM2->CH3CVR = value; break;
        case 4: TIM2->CH4CVR = value; break;
        }
    }
}

static void pwm_config_channel(int timer, int channel) {
    if (timer == 1) {
        switch (channel) {
        case 1: TIM1->CHCTLR1 |= TIM_OC1M_2 | TIM_OC1M_1 | TIM_OC1PE; TIM1->CCER |= TIM_CC1E; break;
        case 2: TIM1->CHCTLR1 |= TIM_OC2M_2 | TIM_OC2M_1 | TIM_OC2PE; TIM1->CCER |= TIM_CC2E; break;
        case 3: TIM1->CHCTLR2 |= TIM_OC3M_2 | TIM_OC3M_1 | TIM_OC3PE; TIM1->CCER |= TIM_CC3E; break;
        case 4: TIM1->CHCTLR2 |= TIM_OC4M_2 | TIM_OC4M_1 | TIM_OC4PE; TIM1->CCER |= TIM_CC4E; break;
        }
    } else {
        switch (channel) {
        case 1: TIM2->CHCTLR1 |= TIM_OC1M_2 | TIM_OC1M_1 | TIM_OC1PE; TIM2->CCER |= TIM_CC1E; break;
        case 2: TIM2->CHCTLR1 |= TIM_OC2M_2 | TIM_OC2M_1 | TIM_OC2PE; TIM2->CCER |= TIM_CC2E; break;
        case 3: TIM2->CHCTLR2 |= TIM_OC3M_2 | TIM_OC3M_1 | TIM_OC3PE; TIM2->CCER |= TIM_CC3E; break;
        case 4: TIM2->CHCTLR2 |= TIM_OC4M_2 | TIM_OC4M_1 | TIM_OC4PE; TIM2->CCER |= TIM_CC4E; break;
        }
    }
}

static uint32_t pwm_prescaler_for(uint32_t freq_hz) {
    uint32_t max_freq = FUNCONF_SYSTEM_CORE_CLOCK / JIKA_PWM_STEPS;
    if (freq_hz == 0U || freq_hz >= max_freq) {
        return 0U;
    }
    uint32_t psc = FUNCONF_SYSTEM_CORE_CLOCK / (freq_hz * JIKA_PWM_STEPS);
    if (psc > 0U) {
        psc -= 1U;
    }
    if (psc > 0xFFFFU) {
        psc = 0xFFFFU;
    }
    return psc;
}

static void pwm_init_timer(int timer) {
    if (timer == 1) {
        if (tim1_ready) {
            return;
        }
        RCC->APB2PCENR |= RCC_APB2Periph_TIM1;
        RCC->APB2PRSTR |= RCC_APB2Periph_TIM1;
        RCC->APB2PRSTR &= ~RCC_APB2Periph_TIM1;
        TIM1->PSC = pwm_prescaler_for(JIKA_PWM_DEFAULT_FREQ_HZ);
        TIM1->ATRLR = JIKA_PWM_STEPS - 1U;
        TIM1->SWEVGR |= TIM_UG;
        TIM1->BDTR |= TIM_MOE;
        TIM1->CTLR1 |= TIM_ARPE | TIM_CEN;
        tim1_ready = 1;
        return;
    }

    if (tim2_ready) {
        return;
    }
    RCC->APB2PCENR |= RCC_APB2Periph_AFIO;
    RCC->APB1PCENR |= RCC_APB1Periph_TIM2;
    AFIO->PCFR1 |= AFIO_PCFR1_TIM2_REMAP_PARTIALREMAP2;
    RCC->APB1PRSTR |= RCC_APB1Periph_TIM2;
    RCC->APB1PRSTR &= ~RCC_APB1Periph_TIM2;
    TIM2->PSC = pwm_prescaler_for(JIKA_PWM_DEFAULT_FREQ_HZ);
    TIM2->ATRLR = JIKA_PWM_STEPS - 1U;
    TIM2->SWEVGR |= TIM_UG;
    TIM2->CTLR1 |= TIM_ARPE | TIM_CEN;
    tim2_ready = 1;
}

int pwm_open(int pin) {
    int timer = 0;
    int channel = 0;
    if (!pwm_lookup(pin, &timer, &channel)) {
        return -1;
    }
    pwm_init_timer(timer);
    funPinMode(pin, GPIO_CFGLR_OUT_10Mhz_AF_PP);
    pwm_config_channel(timer, channel);
    pwm_write_compare(timer, channel, 0U);
    for (int i = 0; i < JIKA_PWM_MAX_HANDLES; i++) {
        if (!pwm_handles[i].used) {
            pwm_handles[i].used = 1;
            pwm_handles[i].timer = timer;
            pwm_handles[i].channel = channel;
            return i;
        }
    }
    return -1;
}

void pwm_set(int handle, int duty) {
    if (handle < 0 || handle >= JIKA_PWM_MAX_HANDLES || !pwm_handles[handle].used) {
        return;
    }
    if (duty < 0) {
        duty = 0;
    }
    if (duty > 100) {
        duty = 100;
    }
    uint32_t compare = ((uint32_t)duty * JIKA_PWM_STEPS) / 100U;
    pwm_write_compare(pwm_handles[handle].timer, pwm_handles[handle].channel, compare);
}

void pwm_set_frequency(int handle, int freq_hz) {
    if (handle < 0 || handle >= JIKA_PWM_MAX_HANDLES || !pwm_handles[handle].used) {
        return;
    }
    if (freq_hz <= 0) {
        return;
    }
    uint32_t psc = pwm_prescaler_for((uint32_t)freq_hz);
    if (pwm_handles[handle].timer == 1) {
        TIM1->PSC = psc;
    } else {
        TIM2->PSC = psc;
    }
}
