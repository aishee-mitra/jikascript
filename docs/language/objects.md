# Objects & Records

JavaScript objects are famously dynamic: you can add properties at runtime, delete them, and change their types. On a tiny microcontroller with only 2 KB of RAM, dynamic dictionaries are slow and risk running out of memory.

JikaScript gives you the best of both worlds: **clean object-literal syntax that compiles directly into lean C structs!**

---

## Defining Objects

You write objects just like you would in JavaScript:

```javascript
const device = {
    name: "sensor-1",
    pin: 3,
    temperature: 24.5,
    enabled: true
};
```

Under the hood, the compiler creates a compact C `struct`:
- No hash maps or dictionaries.
- No heap allocations or garbage collection.
- Field accesses are direct, zero-overhead C struct member lookups (`device.temperature`).

---

## Modifying Properties

In JikaScript, `const` freezes the **binding** (you cannot assign a whole new object to `device`), but the internal properties remain mutable:

```javascript
const device = {
    name: "temp-sensor",
    reading: 100
};

// Updating properties:
device.reading = 105; // Allowed and fast!

// Reassigning the object binding:
// device = { reading: 200 }; // Compile error!
```

---

## Compile-Time Safety

Because object shapes are statically resolved:
- **Typo Protection:** Accessing an unknown property (e.g. `device.temprature`) triggers a compiler error before your code ever touches hardware.
- **Type Checking:** Assigning an incompatible type (e.g. `device.reading = "high"`) is caught at compile time.

Next, let's look at how [Arrays](language/arrays.md) work.
