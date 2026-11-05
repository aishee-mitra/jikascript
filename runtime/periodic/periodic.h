#ifndef PERIODIC_H
#define PERIODIC_H

#include <stdbool.h>
#include <stdint.h>

typedef void (*periodic_callback_t)(void);

typedef struct {
    periodic_callback_t callback;
    uint32_t interval_ms;
    uint32_t last_run_ms;
    bool enabled;
} periodic_task_t;

typedef void (*periodic_critical_section_fn)(void);

void periodic_init(void);
bool periodic_task_add(periodic_task_t *task, periodic_callback_t callback, uint32_t interval_ms);
void periodic_run(void);
uint32_t periodic_millis(void);
void periodic_tick_ms(uint32_t elapsed_ms);
void periodic_set_critical_section(periodic_critical_section_fn enter_fn, periodic_critical_section_fn exit_fn);

#endif
