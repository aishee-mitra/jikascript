#include <assert.h>

#include "periodic.c"

static uint32_t tick_count = 0U;

static void blink(void) {
    tick_count += 1U;
}

int main(void) {
    periodic_task_t task;

    periodic_init();
    assert(periodic_task_add(&task, blink, 2U));
    assert(task.enabled == true);

    g_millis = 0U;
    periodic_run();
    assert(tick_count == 1U);
    assert(task.last_run_ms == 0U);

    g_millis = 5U;
    periodic_run();
    assert(tick_count == 2U);
    assert(task.last_run_ms == 4U);

    return 0;
}
