# Device Capabilities: cam, mic, speech, screen & input in one line

`webview_gui` renders your UI in a real, modern webview. That webview
already ships the browser's media stack — `getUserMedia`, `MediaRecorder`,
the Web Speech API, `getDisplayMedia`, DOM input events. The problem is
never *capability*; it's *ceremony*. Taking one webcam photo is ~25 lines
of stream juggling, a hidden `<video>`, a `<canvas>`, and manual track
teardown. Every app re-writes the same boilerplate, and every app leaks a
camera light because someone forgot `track.stop()`.

**Device capabilities** are a set of native features that collapse that
ceremony into one-liners on a single discoverable namespace:
`window.NATIVE`. Turn one on in `window.yaml`, call one function from your
frontend.

```yaml
native_features:
  - camera   # window.NATIVE.camera.*
  - mic      # window.NATIVE.mic.*
  - speech   # window.NATIVE.speech.*
  - screen   # window.NATIVE.screen.*
  - input    # window.NATIVE.input.*
```

```js
const photo = await NATIVE.camera.snapshot()   // → data URL, camera already released
await NATIVE.speech.say("Smile!")              // → spoken aloud
```

These join the existing native features (`fs`, `os`, `dialogs`, `canvas`)
on the same `NATIVE` object and follow the same rules: enable per app,
exposed before your page's JS runs, every async call returns a Promise.

---

## Table of contents

1. [The 30-second cheat sheet](#cheat-sheet) ← *start here*
2. [Why these are native features, not "just use the browser"](#why)
3. [How it works (and what it costs you: nothing)](#how)
4. [`camera` — photos & video](#camera)
5. [`mic` — recording & live levels](#mic)
6. [`speech` — text-to-speech & speech-to-text](#speech)
7. [`screen` — screenshots & screen recording](#screen)
8. [`input` — keys, hotkeys & mouse](#input)
9. [Recipes: complete mini-apps](#recipes)
10. [Permissions, platforms & gotchas](#permissions)
11. [Design notes & the native-Go frontier](#design-notes)

---

<a name="cheat-sheet"></a>

## 1. The 30-second cheat sheet

Everything you can call, in one table. Each row is a complete call — no
setup, no teardown. **This is the part to bookmark.**

| You want to… | One line | Returns |
|---|---|---|
| **Take a webcam photo** | `await NATIVE.camera.snapshot()` | data URL (PNG) |
| Show the live camera in a `<video>` | `await NATIVE.camera.stream({into:'#cam'})` | `MediaStream` |
| Record a 5s webcam clip | `(await NATIVE.camera.record({ms:5000})).done` | data URL (webm) |
| **Record audio** | `r = await NATIVE.mic.record(); …; await r.stop()` | data URL (webm) |
| Show a live mic level meter | `await NATIVE.mic.meter(v => bar.style.width = v*100+'%')` | `{stop()}` |
| **Speak text aloud** | `await NATIVE.speech.say("Hello")` | resolves when done |
| **Transcribe the mic to text** | `NATIVE.speech.listen(r => box.value = r.text)` | `{stop()}` |
| **Screenshot the desktop** | `await NATIVE.screen.snapshot()` | data URL (PNG) |
| Record the screen for 10s | `(await NATIVE.screen.record({ms:10000})).done` | data URL (webm) |
| **Bind a hotkey** (⌘S / Ctrl+S) | `NATIVE.input.hotkey('ctrl+s', save)` | `{stop()}` |
| Watch every keypress | `NATIVE.input.onKey(e => console.log(e.key))` | `{stop()}` |
| Track mouse clicks | `NATIVE.input.onMouse(e => …)` | `{stop()}` |
| Read the pointer position | `NATIVE.input.pointer()` | `{x, y}` |

Anything that opens a device returns either a Promise of a result (photos,
transcripts) or a small **handle** with a `.stop()` method (recordings,
listeners, meters). Recordings' `.stop()` returns a Promise of the
finished media. There is no separate "release the camera" step — it's
automatic.

---

<a name="why"></a>

## 2. Why these are native features, not "just use the browser"

The honest framing: these are **convenience wrappers** around Web APIs the
webview already has. That is a feature, not an apology — and here is why it
earns a place in `NATIVE`:

- **The boilerplate is the bug.** The raw `getUserMedia → hidden video →
  canvas → toDataURL → stop tracks` dance is where camera-light leaks,
  unstopped recorders, and forgotten `await video.play()` live. Wrapping it
  once, correctly, removes a whole class of mistakes from every app.

- **One discoverable surface.** A developer (or an AI agent — see the
  [capy integration](capy-integration.md)) who types `NATIVE.` should see
  *everything the machine can do* — files, OS, dialogs, **and the camera,
  mic, speaker, screen, and keyboard** — in one place, not scattered across
  four different W3C specs with four different shapes.

- **Capability is opt-in and visible in config.** `native_features:
  [camera, mic]` in `window.yaml` is a one-glance manifest of what an app
  touches. Nothing is exposed unless you list it.

- **Zero added weight.** These wrappers are pure JavaScript injected before
  your page. No new Go dependencies, no CGo, no change to the build matrix,
  and they work on every platform the webview runs on.

What this approach deliberately does **not** do — silent full-desktop
screenshots with no picker, or global keyboard hooks that fire when your
app isn't focused — is covered in [§11](#design-notes). Those need native
OS code; everything in this document needs none.

---

<a name="how"></a>

## 3. How it works (and what it costs you: nothing)

When you list a feature in `native_features`, the orchestrator injects that
feature's small JS shim via the webview's init hook — the same mechanism
behind `fs`/`os`/`dialogs`/`canvas`. The shim defines
`window.NATIVE.<feature>` before your page's scripts run, so you can call it
from an inline `<script>` or after `DOMContentLoaded`.

```
window.yaml: native_features: [camera]
        │
        ▼
orchestrator/usecases/make_init_window.go   win.Init(h, assets.NativeCameraJS)
        │
        ▼
window.NATIVE.camera = { snapshot, stream, record, list, stop }   ← ready before your JS
```

Unlike `fs`/`os` (which bind Go functions because the work happens in Go),
these features have **no Go side** — the capability lives entirely in the
webview, so the shim *is* the whole implementation. That's why there's no
subprocess, no socket, and no per-call IPC: a `NATIVE.camera.snapshot()` is
just JavaScript running in your page.

---

<a name="camera"></a>

## 4. `camera` — photos & video

Enable with `native_features: [camera]`. Exposes `window.NATIVE.camera`.

```js
// Take one photo. The camera opens, captures, and is released — all here.
const dataURL = await NATIVE.camera.snapshot({
  width: 640,          // optional; defaults to the sensor's native size
  height: 480,
  type: "image/jpeg",  // default "image/png"
  quality: 0.9,        // for jpeg/webp
  facingMode: "user",  // "user" (front) | "environment" (rear)
});
document.querySelector("img").src = dataURL;

// Show a live preview in a <video> element.
await NATIVE.camera.stream({ into: "#preview", audio: false });

// Record a clip. Either pass {ms} for a fixed length, or call stop().
const rec = await NATIVE.camera.record({ ms: 5000, audio: true });
const videoURL = await rec.done;        // data URL (video/webm)
// …or manually:  const url = await rec.stop();

// Enumerate cameras.
const cams = await NATIVE.camera.list();  // [{deviceId, label}]

// Release any stream this page opened (preview, dangling record, etc.).
NATIVE.camera.stop();
```

`snapshot()` and the fixed-length `record({ms})` are fully self-cleaning.
`stream()` stays live until you call `NATIVE.camera.stop()` (or stop the
returned `MediaStream`'s tracks yourself).

---

<a name="mic"></a>

## 5. `mic` — recording & live levels

Enable with `native_features: [mic]`. Exposes `window.NATIVE.mic`.

```js
// Record audio until you stop it.
const rec = await NATIVE.mic.record();
recordButton.onclick = async () => {
  const audioURL = await rec.stop();   // data URL (audio/webm)
  player.src = audioURL;
};

// Or auto-stop after a duration:
const clip = await (await NATIVE.mic.record({ ms: 3000 })).done;

// Live input level — perfect for a VU meter / "is it hearing me?" bar.
const meter = await NATIVE.mic.meter(level => {     // level: 0.0 .. 1.0
  bar.style.width = Math.min(100, level * 200) + "%";
});
// later: meter.stop();

const mics = await NATIVE.mic.list();   // [{deviceId, label}]
NATIVE.mic.stop();                       // release everything
```

`meter()` uses the Web Audio analyser under the hood, so you get smooth
per-frame RMS levels without touching `AudioContext` yourself.

---

<a name="speech"></a>

## 6. `speech` — text-to-speech & speech-to-text

Enable with `native_features: [speech]`. Exposes `window.NATIVE.speech`.

```js
// ── Text → speech ──────────────────────────────────────────────
await NATIVE.speech.say("Your download is complete.", {
  rate: 1.0,     // 0.1 .. 10
  pitch: 1.0,    // 0 .. 2
  volume: 1.0,   // 0 .. 1
  lang: "en-US",
  voice: "Samantha",            // optional, by name
});
const voices = NATIVE.speech.voices();  // [{name, lang, default}]
NATIVE.speech.stop();                    // cancel mid-sentence

// ── Speech → text ──────────────────────────────────────────────
const session = NATIVE.speech.listen(({ text, isFinal }) => {
  liveCaption.textContent = text;        // interim results update live
  if (isFinal) transcript.value += text + " ";
}, { lang: "en-US", continuous: true, interim: true });
// later: session.stop();
```

`say()` resolves when the utterance finishes, so you can `await` a line and
then do the next thing. `listen()` streams interim results as the user
speaks and marks each chunk `isFinal` when committed.

> Speech-to-text availability depends on the platform's webview engine
> (it's a vendor-prefixed, network-backed API on some). `say()` is broadly
> available; `listen()` throws `"speech recognition unavailable"` where it
> isn't, so feature-check before wiring a mic-only UI.

---

<a name="screen"></a>

## 7. `screen` — screenshots & screen recording

Enable with `native_features: [screen]`. Exposes `window.NATIVE.screen`.

```js
// Screenshot — the OS shows a one-time "pick a screen/window" chooser.
const shot = await NATIVE.screen.snapshot();   // data URL (PNG)

// Record the screen (optionally with system/mic audio).
const rec = await NATIVE.screen.record({ audio: true });
stopButton.onclick = async () => {
  const movieURL = await rec.stop();           // data URL (video/webm)
};
// or fixed length:  await (await NATIVE.screen.record({ ms: 10000 })).done

// Live mirror into a <video> (e.g. a presenter preview).
await NATIVE.screen.stream({ into: "#mirror" });
NATIVE.screen.stop();
```

This uses `getDisplayMedia`, so the user **chooses** what to share through
the OS picker — by design, it cannot silently grab the whole desktop. For
that (a server-side, no-prompt full-screen grab) see
[§11](#design-notes).

---

<a name="input"></a>

## 8. `input` — keys, hotkeys & mouse

Enable with `native_features: [input]`. Exposes `window.NATIVE.input`.

```js
// Bind an app hotkey. preventDefault is handled for you.
NATIVE.input.hotkey("ctrl+s", () => saveDocument());
NATIVE.input.hotkey("ctrl+shift+k", () => clearConsole());
// "cmd+..." is accepted as an alias for the meta key on macOS.

// Raw key stream (keydown; pass {up:true} to also get keyup).
const keys = NATIVE.input.onKey(e => {
  // e = {key, code, type, ctrl, shift, alt, meta, repeat}
  if (e.key === "Escape") closeModal();
}, { up: true });
// keys.stop();

// Mouse events (default: mousedown, mouseup, click; add 'mousemove'/'wheel').
const mouse = NATIVE.input.onMouse(e => {
  // e = {type, x, y, button, dx, dy, deltaY}
}, { events: ["click", "wheel"] });

// Synchronous pointer read for game loops / drag math.
const { x, y } = NATIVE.input.pointer();
```

These capture input **within your window** — exactly what an app needs for
shortcuts, canvas tools, and games. System-wide hooks (capturing keys while
your app is in the background) are a native-Go feature, noted in
[§11](#design-notes).

---

<a name="recipes"></a>

## 9. Recipes: complete mini-apps

### Webcam photo booth (no backend, ~12 lines)

```yaml
# window.yaml
title: "Photo Booth"
entry_path: ./static/index.html
native_features: [camera, speech]
```

```html
<video id="cam" autoplay muted></video>
<button id="shoot">Take photo</button>
<img id="shot">
<script>
  NATIVE.camera.stream({ into: "#cam" });
  shoot.onclick = async () => {
    await NATIVE.speech.say("Three. Two. One.");
    shot.src = await NATIVE.camera.snapshot();
  };
  NATIVE.input.hotkey("space", () => shoot.click()); // needs `input` too
</script>
```

### Voice notes (mic + speech-to-text, save with `fs`)

```yaml
native_features: [mic, speech, fs]
```

```js
const session = NATIVE.speech.listen(({ text, isFinal }) => {
  if (isFinal) note.value += text + " ";
});
saveBtn.onclick = () =>
  NATIVE.fs.writeFile(`/tmp/note-${Date.now()}.txt`, note.value);
```

### One-click screen recorder

```yaml
native_features: [screen]
```

```js
let rec;
recBtn.onclick = async () => {
  if (!rec) { rec = await NATIVE.screen.record({ audio: true }); recBtn.textContent = "Stop"; }
  else { download.href = await rec.stop(); rec = null; recBtn.textContent = "Record"; }
};
```

---

<a name="permissions"></a>

## 10. Permissions, platforms & gotchas

- **OS permission prompts are real.** The first `camera`/`mic` call triggers
  the platform's permission dialog; `screen` shows the picker every time.
  Your code should assume the user can deny — wrap calls in `try/catch` and
  show a fallback.
- **Secure-context APIs.** `getUserMedia`/`getDisplayMedia` require a secure
  context. `window` serves your app over `http://127.0.0.1`, which counts as
  secure for these APIs, so the default server mode works.
- **Always release.** The wrappers stop tracks for you on `snapshot()`,
  fixed-length `record({ms})`, and `*.stop()`. If you hold a live
  `stream()`, call `NATIVE.<feature>.stop()` when done or the camera light
  stays on.
- **Feature-detect speech.** `say()` is widely available; `listen()` is not
  universal — it throws where unsupported. Check before building a
  voice-only flow.
- **Data URLs vs Blobs.** Recorders return a data URL by default (easy to
  drop into `src`/`href`). Pass `{ blob: true }` if you'd rather have a
  `Blob` to upload or hand to `fs`.

---

<a name="design-notes"></a>

## 11. Design notes & the native-Go frontier

These features are intentionally **pure-JS wrappers**: no new Go
dependencies, no CGo, identical behavior everywhere the webview runs, and
nothing exposed unless `window.yaml` opts in. That keeps the build simple
and the capability surface honest — every wrapper maps to a Web API the
webview genuinely has.

Two genuinely-native capabilities are **out of scope here** because they
can only be done with OS-level Go code, which would add dependencies and
platform-specific build steps:

| Capability | Why it needs native Go | Possible future home |
|---|---|---|
| **Silent full-desktop screenshot** (no picker, capture any window) | `getDisplayMedia` always prompts and is sandboxed to the chosen surface | a Go `infra/native` screenshot adapter (e.g. `kbinani/screenshot`) bound like `fs`/`os` |
| **Global key/mouse hooks** (fire while app is unfocused) | DOM events only reach a focused webview | a Go hook adapter (e.g. `robotn/gohook`, CGo + X11 headers on Linux) |

If those land, they'd follow the existing `fs`/`os` pattern exactly — a raw
adapter in `infra/native/`, a capability struct in `features/`, a
`MakeNative*` builder, and Go functions bound under the same `NATIVE.screen`
/ `NATIVE.input` namespaces — so the frontend API would extend rather than
change. Until then, the JS wrappers cover the overwhelmingly common case:
capture inside an app the user is actively using.
