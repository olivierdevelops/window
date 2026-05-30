# `.window` app demos

Each `*.window` file is a complete app written in **window's own declarative
app language**. Run any of them directly:

```sh
window demos/capy/counter.window
```

`window` recognizes the `.window` extension, transpiles the source to a full
app (`window.yaml` + `static/index.html` + `static/app.js`), and opens it —
all in one binary. The transpiler under the hood is the
[Capy](https://github.com/olivierdevelops/capy) engine, driven by the language
definition in [`assets/window.capy`](../../assets/window.capy); you never
invoke it directly.

**There is no separate backend process:** `BACKEND.call(...)` is served by
in-process Go handlers built into `window`
(see [`infra/handlers.go`](../../infra/handlers.go)).

## What each demo shows

| Demo | Feature exercised |
|------|-------------------|
| counter, todo, star-rating, brand-colors | UI + reactive state + events |
| tip-calculator, temperature-converter, bmi-calculator, miles-to-km | inputs + computed state |
| stopwatch, countdown | timers (`every`) |
| greeter, word-counter | input events + derived state |
| clock | in-process `time` handler + interval |
| dice-roller, coin-flip, random-number, guess-the-number | in-process `random` handler |
| calculator | in-process `math` handler |
| text-reverser | in-process `reverse` handler |
| uuid-generator, password-generator | in-process `uuid` handler |
| system-info | in-process `sysinfo` handler |
| key-value-store | in-process `store.set` / `store.get` handlers |
| http-fetch | in-process `http.get` (no browser CORS) |
| photo-booth | `camera` native feature |
| voice-recorder | `mic` |
| dictation | `speech` (TTS + STT) |
| screen-recorder | `screen` |
| sketchpad | `canvas` |
| terminal | `os` (exec) |
| notepad | `fs` (read/write) |
| file-picker | `dialogs` |
| command-palette | `input` (hotkeys) |
| chart-dashboard | `js_inject` (Chart.js from a CDN) |

## In-process backend handlers available to any app

`echo`, `time`, `uuid`, `random`, `math`, `sysinfo`, `reverse`,
`store.set` / `store.get` / `store.keys`, `http.get`. Call them from a
`.window` source with `get`/`send`, e.g.

```
get tick time into clock
send calc math into result args { a: state.a, b: state.b, op: 'add' }
```

Add more by registering them in `infra/handlers.go` — they run in the same
Go process, no sidecar.
