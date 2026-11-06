# Control Flow

JikaScript supports all the familiar JavaScript control structures you use every day.

---

## Conditionals: `if`, `else if`, `else`

Branching works exactly as you'd expect:

```javascript
if (temperature > 75) {
    fan.turnOn();
} else if (temperature > 50) {
    fan.setLowSpeed();
} else {
    fan.turnOff();
}
```

---

## Loops: `while` and `for`

Run loops using standard syntax:

```javascript
let i = 0;
while (i < 5) {
    led.toggle();
    i = i + 1;
}

for (let j = 0; j < 10; j = j + 1) {
    // Process items
}
```

---

## `switch` Statements

Organize multi-way branching cleanly:

```javascript
switch (state) {
    case 1:
        runInit();
        break;
    case 2:
        runProcessing();
        break;
    default:
        runIdle();
        break;
}
```

Now let's see how JikaScript handles the hardware world: [Tasks & Scheduling](runtime/tasks.md)!
