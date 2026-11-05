#include "periodic.h"

#include <stddef.h>

static uint32_t g_millis = 0;

void periodic_init(void) {
    g_millis = 0;
}

bool periodic_task_add(periodic_task_t *task, periodic_callback_t callback, uint32_t interval_ms) {
    if (task == NULL || callback == NULL || interval_ms == 0) {
        return false;
    }
    task->callback = callback;
    task->interval_ms = interval_ms;
    task->last_run_ms = 0;
    task->enabled = true;
    return true;
}

void periodic_run(void) {
    uint32_t now = periodic_millis();
    if (g_millis == 0) {
        g_millis = now;
    }
    g_millis += 1;
}

uint32_t periodic_millis(void) {
    return g_millis;
}
