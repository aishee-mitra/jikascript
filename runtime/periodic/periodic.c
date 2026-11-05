#include "periodic.h"

#include <stddef.h>

#define PERIODIC_MAX_TASKS 8U

static volatile uint32_t g_millis = 0U;
static periodic_task_t *g_tasks[PERIODIC_MAX_TASKS];
static size_t g_task_count = 0U;
static periodic_critical_section_fn g_enter_critical = NULL;
static periodic_critical_section_fn g_exit_critical = NULL;

static void lock_timebase(void) {
    if (g_enter_critical != NULL) {
        g_enter_critical();
    }
}

static void unlock_timebase(void) {
    if (g_exit_critical != NULL) {
        g_exit_critical();
    }
}

static bool task_due(const periodic_task_t *task, uint32_t now) {
    if (task == NULL || task->callback == NULL || !task->enabled || task->interval_ms == 0U) {
        return false;
    }

    if (task->last_run_ms == UINT32_MAX) {
        return true;
    }

    return (uint32_t)(now - task->last_run_ms) >= task->interval_ms;
}

static void task_advance_deadline(periodic_task_t *task, uint32_t now) {
    if (task == NULL || task->interval_ms == 0U) {
        return;
    }

    if (task->last_run_ms == UINT32_MAX) {
        task->last_run_ms = now;
        return;
    }

    uint32_t elapsed = (uint32_t)(now - task->last_run_ms);
    uint32_t missed = elapsed / task->interval_ms;
    if (missed > 0U) {
        task->last_run_ms += missed * task->interval_ms;
    }
}

void periodic_init(void) {
    lock_timebase();
    g_millis = 0U;
    unlock_timebase();
    for (size_t i = 0; i < PERIODIC_MAX_TASKS; ++i) {
        g_tasks[i] = NULL;
    }
    g_task_count = 0U;
}

bool periodic_task_add(periodic_task_t *task, periodic_callback_t callback, uint32_t interval_ms) {
    if (task == NULL || callback == NULL || interval_ms == 0U) {
        return false;
    }

    if (g_task_count >= PERIODIC_MAX_TASKS) {
        return false;
    }

    for (size_t i = 0; i < g_task_count; ++i) {
        if (g_tasks[i] == task) {
            return false;
        }
    }

    task->callback = callback;
    task->interval_ms = interval_ms;
    task->last_run_ms = UINT32_MAX;
    task->enabled = true;

    g_tasks[g_task_count++] = task;
    return true;
}

void periodic_run(void) {
    uint32_t now = periodic_millis();

    for (size_t i = 0; i < g_task_count; ++i) {
        periodic_task_t *task = g_tasks[i];
        if (task == NULL) {
            continue;
        }

        if (task_due(task, now)) {
            task_advance_deadline(task, now);
            task->callback();
        }
    }
}

uint32_t periodic_millis(void) {
    uint32_t snapshot = 0U;
    lock_timebase();
    snapshot = g_millis;
    unlock_timebase();
    return snapshot;
}

void periodic_tick_ms(uint32_t elapsed_ms) {
    if (elapsed_ms == 0U) {
        return;
    }
    lock_timebase();
    g_millis += elapsed_ms;
    unlock_timebase();
}

void periodic_set_critical_section(periodic_critical_section_fn enter_fn, periodic_critical_section_fn exit_fn) {
    g_enter_critical = enter_fn;
    g_exit_critical = exit_fn;
}
