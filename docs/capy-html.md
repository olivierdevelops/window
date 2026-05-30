# Capy HTML — writing HTML *through* Capy

> **Status: proposal for review** (this doc covers the *keyword* form — `div`,
> `p`, `end`). **The angle-bracket form has since shipped** as the `.htmlx`
> filetype — real matched-pair `<tag>…</tag>` with nesting validated at
> transpile time. See [capy-angle-bracket-html-now-supported.md](capy-angle-bracket-html-now-supported.md),
> the embedded [`assets/htmlx.capy`](../assets/htmlx.capy), and
> [`demos/htmlx/`](../demos/htmlx/). This document below shows the keyword-form
> alternative; nothing here (the keyword library) is wired into `window` yet.
>
> Related: [using-capy.md](using-capy.md) (the high-level `.window` app
> language), [capy-full-apps.md](capy-full-apps.md). Capy is the transpiler
> engine; see the [Capy guide](https://github.com/olivierdevelops/capy).

---

## Table of contents

1. [The idea in one screen](#1-the-idea-in-one-screen)
2. [Why route HTML through Capy](#2-why-route-html-through-capy)
3. [It looks (almost) like HTML](#3-it-looks-almost-like-html)
4. [The cross-platform layer you get for free](#4-the-cross-platform-layer-you-get-for-free)
5. [Tag reference](#5-tag-reference)
6. [Attributes](#6-attributes)
7. [Components — your own tags](#7-components-your-own-tags)
8. [How it would run](#8-how-it-would-run)
9. [`.chtml` vs `.window`](#9-chtml-vs-window)
10. [Appendix — a starter `html.capy`](#appendix-a-starter-htmlcapy)
11. [Open questions for you](#11-open-questions-for-you)

---

## 1. The idea in one screen

A small Capy library — call it `html.capy` — defines one statement per HTML
tag. You author a page in a `.chtml` file that reads almost exactly like
HTML, and Capy emits a real, normalized HTML document.

```
# hello.chtml
page "Hello" 420 300
    div class="card"
        h1 "Welcome"
        p  "This page was written with Capy."
        a  "Docs" href="https://github.com/olivierdevelops/window"
    end
end
```

…transpiles to (real output shape — escaping + indentation handled for you):

```html
<!doctype html>
<html lang="en"><head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
  <meta name="color-scheme" content="dark light" />
  <title>Hello</title>
  <style>/* cross-platform reset — see §4 */</style>
</head><body>
  <div class="card">
    <h1>Welcome</h1>
    <p>This page was written with Capy.</p>
    <a href="https://github.com/olivierdevelops/window">Docs</a>
  </div>
</body></html>
```

The author wrote tags; Capy supplied the document shell, the meta tags, the
cross-platform reset, and HTML-escaped every text node.

---

## 2. Why route HTML through Capy

If it's "almost HTML," why not just write HTML? Because the indirection is a
**single controlled chokepoint** — every page flows through one library, so
improvements land everywhere at once:

- **One place for cross-platform fixes.** `window` runs on three different
  system webviews (WebKit on macOS, WebView2/Chromium on Windows, WebKitGTK
  on Linux). Vendor prefixes, `appearance` resets, `-webkit-` quirks, the
  meta tags, font stacks — fix them once in `html.capy` and *every* page
  inherits the fix. No find-and-replace across hand-written HTML files.
- **Escaping by default.** Text nodes pass through `escapeHtml`, so a stray
  `<` or `&` in content can't break layout or open an injection hole — unless
  you explicitly opt out with `raw`.
- **Validation / a real grammar.** A typo'd tag is a *compile error* with a
  `line:col`, not a silently-ignored element. You can constrain attributes,
  require alt text on images, reject inline event handlers, etc.
- **Components without a framework.** A `card "Title" … end` tag can expand to
  a whole normalized block (see §7) — reuse without React/Vue.
- **One source, many targets.** The same page source could emit a stricter
  AMP-style HTML, an email-safe table layout, or a print stylesheet, by
  swapping the library — the authored page never changes.

The trade-off: a thin layer to learn. The payoff: consistency across
platforms and a place to encode every lesson you learn about webview quirks.

---

## 3. It looks (almost) like HTML

The mapping is close to 1:1, with two ergonomic shifts:

| HTML | Capy HTML |
|------|-----------|
| `<div class="card">…</div>` | `div class="card"` … `end` |
| `<h1>Hello</h1>` | `h1 "Hello"` |
| `<a href="/x">Go</a>` | `a "Go" href="/x"` |
| `<img src="a.png" alt="A">` | `img src="a.png" alt="A"` |
| `<ul><li>One</li></ul>` | `ul` / `li "One"` / `end` |
| `<style>…</style>` | `style` … `end` (raw block) |

Two rules:

1. **Container tags open a block** closed by `end` (`div`, `section`, `ul`,
   `form`, …). Their children are nested, indented lines.
2. **Text tags take their text first**, then optional attributes (`h1 "Hi"
   class="big"`). Attributes are written exactly as in HTML and passed
   through verbatim.

That's the whole syntax. If you can write HTML, you can write this.

### Why `p`, not `<p>` / `(p)` / `/p`?

A deliberate choice, reinforced by how Capy closes blocks. Tested against the
engine:

- An **opening** `<p>` *can* be matched — but only as three separate literals
  (`<` `p` `>`); a single literal `"<p>"` isn't recognized because the lexer
  splits symbol tokens. So angle brackets cost more library code and buy
  nothing.
- The **closing** side is the real constraint: Capy closes a block with a
  *single shared keyword* (`end`) or *single-character delimiters*
  (`block_open "{" close "}"`). It has no per-tag symmetric closer, so a true
  `<p>…</p>` matched pair (with `</p>` vs `</div>`) cuts against the engine.
- Bare keywords are Capy's grain: every sample library uses them, and the
  engine *auto-prepends the function name* as a literal — `function p`
  naturally matches `p …`. Bare `p` + indentation `end` maps 1:1 onto Capy's
  block model.

In short: `p` is idiomatic, readable, and frictionless; `<p>…</p>` would be
more verbose to define *and* partly blocked on the close side. (If you'd
prefer the angle-bracket look anyway, opening tags are feasible — it's a
decision for [§11](#11-open-questions-for-you).)

### Multi-line and rich text

A single quoted string is fine for a label, but real prose is multi-line and
often has inline markup. Capy handles this by letting **one keyword carry two
forms**, chosen by whether an indented block follows
(`when_not_followed_by indent` / `when_followed_by indent`). So `p` is both a
one-liner *and* a container — and there's a verbatim form for raw prose.
All three below are validated against the engine:

```
# 1) inline one-liner
p "A short one-liner." class="lead"

# 2) block form — multi-line, with inline children (a, strong, em, code…)
p class="rich"
    text "This paragraph spans multiple"
    text "source lines and mixes in a"
    a "real link" href="/docs"
    strong "and bold text"
    text "— all inline & escaped."
end

# 3) verbatim block — for LARGE text: arbitrary length, raw bytes, escaped once
text
    A large block of copy — as many lines as you want.
    Write "quotes", & ampersands, <tags>, blank lines —

    all raw in source, escaped exactly once on output.
end
```

`text` is dual-form like the other text tags: `text "short"` for an inline
label, `text … end` (a verbatim block) for **large content**. The block form
is the right tool whenever content is long, multi-paragraph, or contains
characters that are annoying to quote — you never escape anything by hand.

…produces exactly:

```html
<p class="lead">A short one-liner.</p>
<p class="rich">This paragraph spans multiple source lines and mixes in a
  <a href="/docs">real link</a> <strong>and bold text</strong> — all inline &amp; escaped.</p>
<p>Raw prose block.
Newlines &amp; &lt; &gt; preserved,
escaped exactly once.</p>
```

So every text-bearing tag (`p`, `li`, `span`, `button`, `a`, headings…) can
be either a leaf (`tag "text"`) or a block (`tag … end`) with nested inline
tags — and `prose`/`text` blocks cover long copy. Multi-line content is a
first-class citizen, not a workaround.

### What about content that *is* the word `end`?

`end` closes a block, so `<div>end</div>` looks like it might collide. It
doesn't — **indentation disambiguates**: only an `end` at the *opener's*
indentation closes the block; anything indented under it is content. And text
is a quoted string, so it's never confused with the keyword:

```
div
    text "end"
end
```
→ `<div>end</div>` (validated). The same holds for attribute values
(`div class="end"`) and even raw verbatim blocks — a literal `end` line
inside `prose`/`style`/`script` survives as long as it's indented:

```
prose
    end
    if (x) end;
end
```
→ `<p>end\nif (x) end;</p>`. The only `end` that closes is the dedented one.

---

## 4. The cross-platform layer you get for free

Every `page` emits this normalized head — the reason to use the library at
all. It's the same hardening now baked into the `.window` app language, in
one shared place:

```css
/* injected into every page by html.capy */
html { -webkit-text-size-adjust: 100%; text-size-adjust: 100%; }
*, *::before, *::after { box-sizing: border-box; -webkit-tap-highlight-color: transparent; }
body { margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, Roboto,
               "Helvetica Neue", Arial, sans-serif;
  -webkit-font-smoothing: antialiased; -moz-osx-font-smoothing: grayscale; }
button, input, textarea, select { font-family: inherit; -webkit-appearance: none; appearance: none; }
img, video, canvas, svg { max-width: 100%; height: auto; }
a { color: inherit; }
p, pre, span { overflow-wrap: anywhere; word-break: break-word; }
```

…plus `charset`, a `viewport` with `viewport-fit=cover`, and
`color-scheme`. When we discover a new webview quirk, it goes here once.

You can layer your own CSS on top with a `style … end` block; the reset is
always emitted first so your rules win.

---

## 5. Tag reference

A representative set (extend freely — it's just a library):

**Containers (block, closed by `end`)**
`page` · `div` · `section` · `header` · `footer` · `main` · `nav` ·
`article` · `aside` · `ul` · `ol` · `form` · `label`

**Text tags (text first, then attrs — *or* a block, see [§3](#3-it-looks-almost-like-html))**
`h1` `h2` `h3` `h4` · `p` · `span` · `a` · `button` · `li` · `strong` ·
`em` · `small` · `code` — each works as a one-liner (`p "text"`) or a block
(`p … end`) with nested inline tags; `prose … end` is the verbatim long-copy form.

**Void / leaf**
`img` (attrs) · `input` (attrs) · `hr` · `br`

**Raw blocks**
`style … end` (CSS, verbatim) · `script … end` (JS, verbatim) ·
`text "…"` (an escaped text node) · `raw … end` (unescaped HTML passthrough —
the escape hatch)

Example exercising several:

```
page "Profile" 480 560
    style
        .card { background:#161b22; border-radius:14px; padding:20px; }
    end
    div class="card"
        img src="/static/avatar.png" alt="Avatar"
        h2 "Ada Lovelace"
        p  "First programmer."
        ul
            li "Mathematician"
            li "Writer"
        end
        a "Follow" href="#" class="btn"
    end
    script
        document.querySelector('.btn').onclick = () => alert('followed');
    end
end
```

---

## 6. Attributes

Attributes are written **exactly as in HTML** and passed through unchanged —
they're captured as the trailing part of the line:

```
button "Save" id="save" class="primary" data-role="submit"
input type="email" placeholder="you@example.com" required
```

Because they're verbatim, anything HTML allows works (`data-*`, `aria-*`,
boolean attrs, inline styles). Text *content* is escaped; attribute *values*
are your responsibility — same contract as writing HTML by hand. (We could
add an opt-in strict mode that validates/escapes attributes later.)

---

## 7. Components — your own tags

The real leverage: define a higher-level tag once, reuse it everywhere, and
it expands to normalized markup. This is "components" with no framework and
no runtime.

```
# in html.capy
function card
    arg literal "card"
    arg capture title string
    block_closer end
    write `<section class="card">
  <h3>${escapeHtml (unquote title)}</h3>
${indent 2 body}</section>
`
end
```

```
# in a .chtml page
card "Settings"
    p "Profile options go here."
    button "Save" class="primary"
end
```

→ a consistent `<section class="card">` with an escaped heading, every time.
Change the card's markup in one place; every page updates. A whole design
system can live in one `.capy` file.

---

## 8. How it would run

Same shape as the `.window` language: a new embedded library + a recognized
extension, so there's nothing extra to install.

```sh
window hello.chtml      # transpile via html.capy → window.yaml + static/, then run
```

Under the hood (proposed): `appio` sees `.chtml`, runs the source through the
embedded `html.capy` with the existing `infra.GenerateCapyApp`, writes the
files to a temp dir, and runs the generated `window.yaml`. The `page`
statement emits both the `window.yaml` (title/size) and `static/index.html`.

Because it produces an ordinary `window` app, everything else still applies:
`native_features`, `js_inject`, and the **in-process Go backend** —
`BACKEND.call(...)` works from a `script` block with no sidecar, exactly as
in the rest of `window`.

---

## 9. `.chtml` vs `.window`

They're complementary layers; pick by altitude:

| | `.window` (app language) | `.chtml` (Capy HTML) |
|---|---|---|
| Altitude | high-level verbs (`state`, `button`, `photobooth`) | low-level HTML tags |
| You get | reactive store, events, native widgets, in-process calls — generated | exact markup control + the cross-platform layer |
| Best for | apps: counters, dashboards, device demos | precise pages, custom layouts, design systems |
| Escape hatch | `inject`, raw setup | `raw`, `style`, `script` |

You could even mix: prototype fast in `.window`, drop to `.chtml` when you
need pixel control — both ride the same cross-platform reset and the same
in-process backend.

---

## Appendix — a starter `html.capy`

A trimmed but real library (validated against the engine). Note
`escapeHtml (unquote …)`: a string capture keeps its source quotes, so
`unquote` strips them before escaping.

```
extension html
comments
    line "#"
end

context
    title  "Page"
    width  480
    height 640
end

# ── document shell (emits the cross-platform reset) ──────────────────
function page
    arg literal "page"
    arg capture title string
    arg capture w int default "480"
    arg capture h int default "640"
    block_closer end
    set context.title title
    set context.width w
    set context.height h
end

# ── containers (block) ───────────────────────────────────────────────
function div
    arg literal "div"
    arg capture attrs tail default ""
    block_closer end
    write `<div ${attrs}>
${indent 2 body}</div>
`
end
# … section / header / footer / main / nav / ul / ol / form / label
#    are identical with their own tag name.

# ── text tags (text first, then attrs) ───────────────────────────────
function h1
    arg literal "h1"
    arg capture txt string
    arg capture attrs tail default ""
    write `<h1 ${attrs}>${escapeHtml (unquote txt)}</h1>
`
end
# … h2 / h3 / p / span / a / button / li / strong / em / small / code

# ── void / raw ───────────────────────────────────────────────────────
function img
    arg literal "img"
    arg capture attrs tail default ""
    write `<img ${attrs} />
`
end

function style
    arg literal "style"
    block_verbatim end
    write `<style>
${body}</style>
`
end

function script
    arg literal "script"
    block_verbatim end
    write `<script>
${body}</script>
`
end

function text
    arg literal "text"
    arg capture t string
    write `${escapeHtml (unquote t)}
`
end

function raw
    arg literal "raw"
    block_verbatim end
    write `${body}
`
end

function end
end

# ── output: window.yaml + the page, with the cross-platform reset ────
file "window.yaml"
    write `title: ${toJSON context.title}
entry_path: ./static/index.html
static_dirs:
  "/static": ./static/
size:
  width: ${context.width}
  height: ${context.height}
`
end

file "static/index.html"
    write `<!doctype html>
<html lang="en"><head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
<meta name="color-scheme" content="dark light" />
<title>${escapeHtml (unquote context.title)}</title>
<style>
  html { -webkit-text-size-adjust:100%; text-size-adjust:100%; }
  *,*::before,*::after { box-sizing:border-box; -webkit-tap-highlight-color:transparent; }
  body { margin:0; font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",system-ui,Roboto,"Helvetica Neue",Arial,sans-serif;
    -webkit-font-smoothing:antialiased; -moz-osx-font-smoothing:grayscale; }
  button,input,textarea,select { font-family:inherit; -webkit-appearance:none; appearance:none; }
  img,video,canvas,svg { max-width:100%; height:auto; }
  p,pre,span { overflow-wrap:anywhere; word-break:break-word; }
</style></head><body>
${indent 2 body}</body></html>
`
end
```

> The `body` local inside `page`'s block isn't used directly here; in the
> real library the page's children render through the `static/index.html`
> file block via shared state, the same pattern `window.capy` already uses.
> (One small design choice to settle — see below.)

---

## 11. Open questions for you

A few decisions before building this, if you want it:

1. **Extension** — `.chtml`? `.whtml`? `.page`? (I leaned `.chtml` = "Capy
   HTML".)
2. **Attribute strictness** — pass attributes through verbatim (HTML-like,
   shown here), or add a strict mode that validates/escapes them?
3. **Scope of the tag set** — ship a minimal set (≈20 tags) and let users add
   more, or a comprehensive set out of the gate?
4. **Components** — bundle a small built-in component set (card, row, button
   variants) on top of raw tags, or keep it pure tags?
5. **Relationship to `.window`** — keep them separate libraries, or allow
   embedding HTML tags inside a `.window` app for an escape hatch?

Tell me which way you want to go and I'll build it (library + `.chtml`
wiring + demos), the same way the `.window` language is wired today.
