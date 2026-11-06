# Tasks & Scheduling

In traditional Arduino or C code, multitasking usually means writing messy `millis()` math or complex interrupt service routines (ISRs).

JikaScript includes a **pure-C periodic cooperative scheduler** that lets you write clean, concurrent tasks effortlessly!

---

## Defining a Task

Create tasks with the `task()` function:

```javascript
// task(name, interval_ms, callback)
task("heartbeat", 500, () => {
    led.toggle();
});
```

- **`name`:** A unique identifier string for the task.
- **`interval_ms`:** How often (in milliseconds) the task callback should run.
- **`callback`:** The function to execute.

---

## Multiple Concurrent Tasks

You can register multiple periodic tasks that run concurrently without blocking each other:

```javascript
const led1 = pin("PC1");
const led2 = pin("PC2");

led1.output();
led2.output();

// Flash led1 quickly every 500ms
task("led1", 500, () => {
    led1.toggle();
});

// Flash led2 more slowly every 1000ms
task("led2", 1000, () => {
    led2.toggle();
});

// Start executing scheduled tasks
scheduler.start();
```

---

## How It Works Under the Hood

- **Non-Preemptive & Cooperative:** Tasks run to completion when triggered. There is no complicated thread context-switching or stack overflow worries.
- **Timer Driven:** A clean 1 ms hardware timer advances the scheduler clock. Your application callbacks run in main context, safely away from interrupt restrictions.
- **Zero Memory Allocation:** Task slots are allocated statically at compile time (default 8 tasks, customizable).

Next, let's look at [GPIO & Hardware](runtime/hardware.md).
