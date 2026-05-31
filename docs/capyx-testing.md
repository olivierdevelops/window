# Testing `.capyx` components in isolation — no browser, no Selenium

Because a `.capyx` app is built from **isolated VHCO units** — dumb
`component` views, `handler` state/event units, and `capability` ports that are
the only seam to the outside world — every component can be tested *on its own*:
mounted alone, with its initial state set to anything, its capabilities replaced
by mocks, driven by events, and asserted on. No real services, no running
backend, and **no browser automation** (no Selenium, no Playwright, no
`window` even needs to open for the headless runner).

Two surfaces, one engine:

| Surface | Command | For |
|---|---|---|
| **Headless suite runner** | `window test suite.capytest` | CI, fast feedback, pass/fail in the terminal |
| **Interactive Test Bench** | `window test --ui app.capyx` | exploring a component by hand; non-devs; recording tests by clicking |

Both run the *same* kernel ([`assets/capyx_testkit.js`](../assets/capyx_testkit.js)).
The headless runner executes it under a tiny Node DOM shim; the bench runs it in
the real webview. Either way, mounting + mocking + assertions are identical.

---

## 1. Why this works — the harness build

The normal build auto-mounts your app. The **harness build** instead exposes
every component render function, handler factory, capability provider, and an
introspection *meta* on `globalThis.__CAPYX_TEST__`. The kernel uses that to:

- **mount one component in a fresh, detached root** — no siblings, no shared
  DOM. True isolation.
- **override initial state** — start the component in any scenario you want.
- **replace capabilities with recording mocks** — the handler's ports are
  swapped for canned implementations that record every call; the handler never
  touches the real world.
- **drive + assert** — click/input/fire/call, then assert on rendered text,
  handler state, element counts, or mock call counts.

Mutating one field still re-runs only the effects that read it (the same
fine-grained reactivity as production), so what you test is what ships.

---

## 2. The `.capytest` format

A suite names the app under test and lists scenarios. Each scenario is a
sequence of plain-English steps — readable and diff-friendly enough for a
non-developer to edit:

```capytest
suite "Counter component"
use ./counter.capyx

scenario "increments on +"
    mount counter
    click "+"
    click "+"
    expect state count 2
    expect text "2"

scenario "reset returns to zero from any state"
    mount counter
    set count 42
    click "reset"
    expect state count 0
    expect text "back to zero"
```

### Steps

| Step | Meaning |
|---|---|
| `mount <component> [use <handler>]` | mount one component in isolation (handler defaults to the same name) |
| `set <field> <json>` | set a handler state field (before *or* after mount) |
| `mock <Cap>.<method> returns <json>` | canned return for a capability method |
| `mock <Cap>.<method> returns seq [a, b, …]` | a different value per call (sequence) |
| `click "<text>"` / `click .selector` | click the matching element (text prefers buttons/links) |
| `input "<text>" <json>` / `input .sel <json>` | set an input's value and fire `input` |
| `fire <event> <target>` | dispatch any DOM event |
| `call <method> [jsonArgsArray]` | invoke a handler event method directly |
| `expect text "<substr>"` | rendered text contains the substring |
| `expect no text "<substr>"` | rendered text does **not** contain it |
| `expect state <field> <json>` | handler state field equals (deep) |
| `expect count <selector> <n>` | exactly `n` matching elements |
| `expect called <Cap>.<method> [<n>]` | the mock was called (n times) |
| `expect class <selector> "<class>"` | some matching element has the class |

Values are JSON (`5`, `"hi"`, `true`, `[]`, `{"id":1}`); a bare unquoted word is
taken as a string.

> **Tip — mock before mount.** A handler's `mount()` lifecycle often calls a
> capability (e.g. `items = store.seed()`). Put the `mock` step *before*
> `mount` so the lifecycle sees it. The kernel buffers pre-mount mocks for you.

### Mocking a capability — worked example

`notes.capyx` declares `capability Store { fn seed(); fn next(items) }` and the
handler's `mount()` calls `store.seed()`. Test the view against any data with no
real store:

```capytest
suite "Notes with a mocked Store"
use ./notes.capyx

scenario "renders notes from a mocked store"
    mock Store.seed returns [{"id":10,"text":"Mocked note"}]
    mount notes
    expect text "Mocked note"
    expect count li 1
    expect called Store.seed 1

scenario "reload re-reads the store"
    mock Store.seed returns seq [[{"id":1,"text":"first"}], [{"id":2,"text":"second"}]]
    mount notes
    expect text "first"
    click "reload from store"
    expect text "second"
    expect called Store.seed 2
```

---

## 3. Running headlessly

```sh
window test demos/capyx/counter.capytest
```

```
  Counter component
  ✓ starts at zero
  ✓ increments on +
  ✓ decrements on -
  ✓ reset returns to zero from any state
  ✓ shows climbing message when positive

  5 passed, 0 failed
```

Exit code is non-zero if any scenario fails, so it drops straight into CI. The
headless runner needs **Node.js** on `PATH` (it executes the kernel under a DOM
shim); the interactive bench does not.

You can also drive the same pipeline from Go — see
[`infra/capytest_run_test.go`](../infra/capytest_run_test.go), which parses a
suite, compiles the app in harness mode, and asserts the structured results.

---

## 4. The interactive Test Bench

```sh
window test --ui demos/capyx/counter.capyx
```

A window opens with:

- **left** — a picker listing every component in the app;
- **center** — a **live, isolated preview** of the selected component, plus
  live readouts of the rendered text and the handler's state;
- **right** — an **initial-state editor** (one field per state value), **event
  buttons** (one per handler `on` method), and **capability mock editors** (a
  JSON return box per method of each injected capability), plus one-click
  **assertions**;
- **bottom** — the **recorded scenario** as `.capytest` text, a **Run** button
  that executes it and shows per-step ✓/✗, and a pass/fail status line.

The workflow for a non-developer: pick a component, click its buttons in the
preview (each click is recorded), set state or mock a service from the side
panel, then hit **“text contains …”** or **“snapshot state as expectations”** to
record what *should* be true. Press **Run** to verify, and copy the generated
`.capytest` into your repo so CI re-runs it forever.

---

## 5. How the pieces fit

```
app.capyx ──CompileCapyxHarness──▶ __CAPYX_TEST__ registry + meta
                                      │
            capyx_testkit.js (kernel) ┤  mountIsolated · mocks · drivers · assert · runScenario
                                      │
   ┌──────────────────────────────────┴───────────────────────────────┐
   ▼                                                                    ▼
window test suite.capytest                                  window test --ui app.capyx
  (Go: parse .capytest → Node DOM shim → results)             (webview: capyx_testbench.js UI)
```

- [`infra/capyx_testkit.go`](../infra/capyx_testkit.go) — the harness build +
  introspection meta + bench page assembler.
- [`infra/capytest.go`](../infra/capytest.go) — the `.capytest` parser.
- [`infra/capytest_run.go`](../infra/capytest_run.go) — the headless Node runner.
- [`assets/capyx_testkit.js`](../assets/capyx_testkit.js) — the shared kernel.
- [`assets/capyx_testbench.js`](../assets/capyx_testbench.js) — the bench UI.

## See also

- [`capyx-reactive-vhco.md`](./capyx-reactive-vhco.md) — the `.capyx` model the
  isolation builds on.
- [`authoring-formats.md`](./authoring-formats.md) — `.capyx` vs `.htmlx` vs
  `.cs`.
- [`cli.md`](./cli.md) — the full `window` command reference.
