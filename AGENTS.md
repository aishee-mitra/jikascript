# AGENTS.md — MicroJS Development Instructions

## 1. Primary Instruction

Before making any changes, read:

- `SPEC.md` — authoritative MicroJS language, compiler, runtime, scheduler, and target specification.
- `README.md` — project overview and current setup instructions, if present.

**SPEC.md is the source of truth for what MicroJS must implement.**

This file complements SPEC.md by defining how implementation work should be performed.

Do not duplicate, redefine, or contradict technical requirements in SPEC.md.

If an implementation detail conflicts with SPEC.md, follow SPEC.md and report the discrepancy.

If SPEC.md is ambiguous, choose the simplest implementation consistent with its architecture. Document the assumption rather than silently introducing a new design.

## 2. Project Philosophy

Keep MicroJS:

- Simple.
- Lightweight.
- Portable.
- Predictable in memory and execution.
- Easy to understand and maintain.

Avoid over-engineering.

Prefer a small working implementation over elaborate abstractions or speculative features.

Do not introduce dependencies, frameworks, compiler infrastructure, or runtime components without a clear requirement.

## 3. Architecture Boundaries

Respect the separation between:

1. Golang-based MicroJS compiler.
2. MicroJS language and its ECMAScript-compatible semantics.
3. Standalone C scheduler/runtime.
4. Target-specific hardware backends.
5. Native C toolchain and firmware build process.

Keep the compiler independent of individual MCU targets.

Keep the scheduler independently usable without MicroJS.

Keep hardware-specific functionality out of the language core.

Do not introduce architectural coupling between these components.

## 4. Repository Exploration

Before implementation:

1. Inspect the repository structure.
2. Read relevant existing source files.
3. Identify existing libraries and build conventions.
4. Inspect the existing scheduler implementation before modifying it.
5. Verify hardware APIs against the actual target framework.
6. Identify existing tests and preserve them.

Never assume an API exists simply because it appears in an example or proposed design.

Existing working implementations should be reused where appropriate rather than replaced without justification.

## 5. Implementation Workflow

Work incrementally.

For each task:

1. Read the relevant sections of SPEC.md.
2. Inspect the existing implementation.
3. Identify the smallest complete change.
4. Briefly state the implementation approach.
5. Implement the change.
6. Add or update tests.
7. Run relevant tests.
8. Fix regressions.
9. Update documentation where necessary.
10. Summarize completed work and remaining issues.

Do not attempt to implement the entire specification in one operation.

Do not move to the next milestone while the current milestone has unresolved critical failures.

## 6. Coding Standards

### Go

- Follow idiomatic Go conventions.
- Prefer the standard library.
- Keep packages focused.
- Avoid unnecessary interfaces and abstractions.
- Handle errors explicitly.
- Avoid global mutable state.
- Use clear names and maintainable code.

### C

- Follow the C standard specified in SPEC.md.
- Use fixed-width integer types where appropriate.
- Avoid undefined behavior.
- Avoid unnecessary dynamic allocation.
- Keep hardware-specific code isolated.
- Handle interrupt/main-context shared state correctly.
- Enable compiler warnings.
- Keep the scheduler portable.

### Generated code

- Follow the output requirements in SPEC.md.
- Ensure generated code is deterministic.
- Ensure generated C is valid for the specified C standard.
- Keep inspection output readable.
- Never introduce a C++ dependency when pure C is required.

## 7. Testing Requirements

Tests are part of implementation, not an optional final step.

For every feature:

- Add unit tests.
- Test valid inputs.
- Test invalid inputs.
- Test boundary conditions.
- Test generated code where applicable.

Use host-side tests wherever practical to avoid requiring physical hardware for every development cycle.

For hardware-dependent features, provide target-specific integration tests or reproducible hardware test examples.

Never claim a feature works on hardware unless it has actually been verified on hardware.

Do not remove or weaken existing tests simply to make a build pass.

## 8. ECMAScript Compatibility

Use SPEC.md as the authority for the supported ECMAScript subset.

When implementing language features:

- Consult ECMAScript semantics for supported constructs.
- Do not invent alternative behavior without documenting it.
- Do not claim compatibility based only on syntax parsing.
- Add tests for semantic behavior.
- Explicitly report unsupported constructs.

Avoid implementing features outside the agreed language scope.

## 9. Dependencies and Optimization

Keep dependencies minimal.

Do not introduce LLVM, a VM, an interpreter, a garbage collector, or other major infrastructure unless explicitly approved.

Prioritize correctness before optimization.

Production optimizations must preserve program behavior.

Generated inspection output and optimized production output must remain semantically equivalent.

Do not sacrifice correctness or debuggability for marginal code-size improvements.

## 10. Change Management

Do not silently change:

- Language syntax or semantics.
- Compiler architecture.
- Scheduler behavior.
- Public scheduler APIs.
- Hardware APIs.
- CLI behavior.
- Target configuration.

When a change affects an established interface or architectural decision:

1. Explain why the change is necessary.
2. Identify affected components.
3. Describe compatibility implications.
4. Update SPEC.md when the specification itself changes.
5. Obtain approval for significant architectural changes.

Avoid unrelated refactoring.

## 11. Progress Reporting

At the end of each implementation task, report:

- What was implemented.
- Files created or modified.
- Tests added.
- Tests executed and their results.
- Hardware validation performed, if any.
- Known limitations.
- Recommended next task.

Distinguish clearly between implemented, tested, and hardware-verified functionality.

## 12. Starting Instructions

When first entering the repository:

1. Read SPEC.md completely.
2. Inspect the repository and existing scheduler source.
3. Compare the repository against the specification.
4. Identify missing components and existing reusable code.
5. Produce a concise implementation plan.
6. Begin with the first incomplete milestone defined in SPEC.md.

Do not start by generating the entire project or implementing every feature.

## 13. Suggested Repo structure

```
  MicroJS/
  ├── AGENTS.md       # How the coding agent works
  ├── SPEC.md         # What MicroJS must become
  ├── README.md       # Project overview and getting started
  ├── go.mod
  ├── cmd/
  ├── internal/
  ├── targets/
  ├── runtime/
  └── examples/
```

## 14. Git Version Control

Use disciplined and focused git practices during implementation.

- Keep commits focused by milestone or tightly related change set.
- Use Conventional Commit/Commitizen-style commit messages.
- Do not mix unrelated changes in a single commit.
- Do not rewrite history (`rebase`, `reset --hard`, `push --force`) unless explicitly requested.
- Do not revert user changes unless explicitly requested.
- Check `git status` before and after edits.
- Include a short verification summary with each commit-ready milestone.

---
**Guiding principle: Follow the specification, make the smallest correct change, test it, and keep the architecture simple.**