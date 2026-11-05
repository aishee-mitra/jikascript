#include "periodic.h"

#include <assert.h>

static uint32_t tick_count = 0U;

static void blink(void) {
    tick_count += 1U;
}

int main(void) {
    periodic_task_t task;

    periodic_init();
    assert(periodic_task_add(&task, blink, 2U));
    assert(task.enabled == true);

    periodic_run();
    assert(tick_count == 1U);
    assert(task.last_run_ms == 0U);

    periodic_run();
    assert(tick_count == 2U);

    periodic_task_remove(&task);
    assert(task.enabled == false);

    return 0;
}
