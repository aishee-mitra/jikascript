# GPIO & Hardware

Controlling physical pins in JikaScript is as simple as calling methods on an object.

---

## Configuring Pins

Select a pin using its hardware name (such as `"PC1"`, `"PD2"`, etc.):

```javascript
const led = pin("PC1");
const button = pin("PD2");

// Set pin directions
led.output();
button.input();
```

---

## Reading and Writing Pins

Interact with inputs and outputs cleanly:

```javascript
// Toggle pin state
led.toggle();

// Read input state
let state = button.read();
if (state == LOW) {
    // Button pressed!
}
```

---

## Interactive Example: Button Toggle

Here is a complete example checking a push-button every 20ms and toggling an LED when pressed:

```javascript
const led = pin("PC1");
const button = pin("PD2");

led.output();
button.input();

task("button_check", 20, () => {
    if (button.read() == LOW) {
        led.toggle();
    }
});

scheduler.start();
```

Everything compiles down to direct register writes through the CH32fun backend — fast, efficient, and direct.

Curious how it all turns into C? Check out [How Compilation Works](under-the-hood.md)!
