// Capacity-override test: compile with -DPERIODIC_MAX_TASKS=<n> (n >= 2).
// Verifies that the build-time capacity macro is honored by the scheduler.
#include <assert.h>
#include <stddef.h>
#include <stdint.h>

#include "periodic.h"

static uint32_t g_count = 0U;

static void cb(void) {
    g_count += 1U;
}

int main(void) {
    // The macro must be overridable and visible to both header and source.
#if defined(PERIODIC_MAX_TASKS) && PERIODIC_MAX_TASKS == 2U
    periodic_task_t slots[2];

    periodic_init();
    assert(periodic_task_add(&slots[0], cb, 1U));
    assert(periodic_task_add(&slots[1], cb, 1U));
    // Capacity is exhausted: further registration must fail, not silently drop.
    periodic_task_t extra;
    assert(!periodic_task_add(&extra, cb, 1U));

    periodic_tick_ms(1U);
    periodic_run();
    assert(g_count == 2U);
    return 0;
#else
    // Default build: capacity must be the documented default of 8.
    periodic_task_t slots[8];
    periodic_task_t extra;

    periodic_init();
    for (int i = 0; i < 8; ++i) {
        assert(periodic_task_add(&slots[i], cb, 1U));
    }
    assert(!periodic_task_add(&extra, cb, 1U));
    return 0;
#endif
}
