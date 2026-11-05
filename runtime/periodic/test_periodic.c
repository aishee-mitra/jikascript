#include <assert.h>
#include <stddef.h>
#include <stdint.h>

#include "periodic.h"

static uint32_t g_fast_count = 0U;
static uint32_t g_slow_count = 0U;
static uint32_t g_long_count = 0U;
static periodic_task_t g_fast_task;
static periodic_task_t g_slow_task;
static periodic_task_t g_long_task;

static void reset_counts(void) {
    g_fast_count = 0U;
    g_slow_count = 0U;
    g_long_count = 0U;
}

static void fast_cb(void) {
    g_fast_count += 1U;
}

static void slow_cb(void) {
    g_slow_count += 1U;
}

static void long_cb(void) {
    g_long_count += 1U;
    periodic_tick_ms(100U);
}

int main(void) {
    periodic_task_t slots[8];

    periodic_init();
    reset_counts();

    assert(periodic_task_add(&g_fast_task, fast_cb, 2U));
    periodic_run();
    assert(g_fast_count == 1U);
    assert(g_fast_task.last_run_ms == 0U);

    periodic_run();
    assert(g_fast_count == 1U);
    periodic_tick_ms(1U);
    periodic_run();
    assert(g_fast_count == 1U);
    periodic_tick_ms(1U);
    periodic_run();
    assert(g_fast_count == 2U);
    assert(g_fast_task.last_run_ms == 2U);

    periodic_init();
    reset_counts();
    assert(periodic_task_add(&g_fast_task, fast_cb, 2U));
    assert(periodic_task_add(&g_slow_task, slow_cb, 5U));
    periodic_run();
    assert(g_fast_count == 1U);
    assert(g_slow_count == 1U);
    periodic_tick_ms(4U);
    periodic_run();
    assert(g_fast_count == 2U);
    assert(g_slow_count == 1U);
    periodic_tick_ms(1U);
    periodic_run();
    assert(g_fast_count == 2U);
    assert(g_slow_count == 2U);

    periodic_init();
    reset_counts();
    assert(!periodic_task_add(&g_fast_task, fast_cb, 0U));
    assert(!periodic_task_add(&g_fast_task, NULL, 10U));
    assert(periodic_task_add(&g_fast_task, fast_cb, 1U));
    assert(!periodic_task_add(&g_fast_task, fast_cb, 1U));
    for (uint32_t i = 0U; i < 7U; ++i) {
        assert(periodic_task_add(&slots[i], fast_cb, 1U));
    }
    assert(!periodic_task_add(&slots[7], fast_cb, 1U));

    periodic_init();
    reset_counts();
    assert(periodic_task_add(&g_fast_task, fast_cb, 5U));
    periodic_run();
    assert(g_fast_count == 1U);
    periodic_tick_ms(17U);
    periodic_run();
    assert(g_fast_count == 2U);
    assert(g_fast_task.last_run_ms == 15U);

    periodic_init();
    reset_counts();
    assert(periodic_task_add(&g_long_task, long_cb, 50U));
    periodic_run();
    assert(g_long_count == 1U);
    periodic_run();
    assert(g_long_count == 2U);

    periodic_init();
    reset_counts();
    assert(periodic_task_add(&g_fast_task, fast_cb, 10U));
    periodic_run();
    periodic_tick_ms(UINT32_MAX - 5U);
    periodic_run();
    assert(g_fast_count == 2U);
    periodic_tick_ms(20U);
    periodic_run();
    assert(g_fast_count == 3U);

    return 0;
}
