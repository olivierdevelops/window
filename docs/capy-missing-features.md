# Capy missing features — why angle-bracket HTML isn't possible

> **⚠️ SUPERSEDED — this document is wrong.** It was written from incomplete
> probes: I tested `arg literal "<div>"` and `block_open/close` but never
> discovered the **`block_close_seq`** block mode, which was already present in
> the engine. Angle-bracket, matched-pair HTML (`<p>…</p>`, `<div>…</div>`,
> *with* mismatched-nesting detection) **is** authorable in Capy today —
> verified by direct probe. See
> [capy-angle-bracket-html-now-supported.md](capy-angle-bracket-html-now-supported.md)
> for the working library and proof. The four "missing" features below are
> either shipped (`block_close_seq` covers §4–§6) or unnecessary (§3 — you
> capture the tag name instead of matching a `"<div>"` literal). The original
> text is kept below for the record only.
>
> Related: [capy-html.md](capy-html.md) (the keyword near-HTML library),
> [using-capy.md](using-capy.md). Capy is the transpiler engine; see the
> [Capy guide](https://github.com/olivierdevelops/capy).

---

## Table of contents

1. [The question this answers](#1-the-question-this-answers)
2. [What "angle-bracket HTML" would require](#2-what-angle-bracket-html-would-require)
3. [Missing feature 1 — multi-character symbol literals](#3-missing-feature-1--multi-character-symbol-literals)
4. [Missing feature 2 — per-tag symmetric block closers](#4-missing-feature-2--per-tag-symmetric-block-closers)
5. [Missing feature 3 — multi-token close delimiters](#5-missing-feature-3--multi-token-close-delimiters)
6. [Missing feature 4 — closers that depend on the opener](#6-missing-feature-4--closers-that-depend-on-the-opener)
7. [What Capy *can* do instead](#7-what-capy-can-do-instead)
8. [Summary table](#8-summary-table)

---

## 1. The question this answers

The natural instinct is to make Capy HTML look like *real* HTML:

```html
<div class="card">
  <p>Hello, <b>world</b>.</p>
</div>
```

…with matched opening and closing tags. The proposal in
[capy-html.md](capy-html.md) instead uses bare keywords and a single `end`:

```
div class="card"
    p "Hello, " b "world" "."
end
```

This doc explains, feature by feature, *why* the first form is not buildable on
today's Capy. Each section names one missing capability, shows the probe that
proves it's missing, and states what it would have enabled.

---

## 2. What "angle-bracket HTML" would require

To match `<p>some text</p>` as a block, a Capy library would need **four**
things to all be true at once:

1. The lexer must treat `<p>` as a **single matchable token** (or let a
   function declare `<`, `p`, `>` as a literal sequence that opens a block).
2. A block must be closeable by a **tag-specific** terminator — `</p>` closes a
   `p`, `</div>` closes a `div` — not by one shared keyword.
3. That terminator must be a **multi-character / multi-token delimiter**
   (`</p>` is four characters across three lexical pieces: `<`, `/p`, `>`).
4. The closer must be **bound to the opener** so that `</div>` cannot
   accidentally close a `<p>`.

Capy provides none of these four. The sections below take them in turn.

---

## 3. Missing feature 1 — multi-character symbol literals

**What's missing:** the ability to declare a multi-character symbol string as a
single literal token that a function matches on.

**Probe.** A function that tries to open on `<div>`:

```
function tag
    arg literal "<div>"
    ...
```

**Result:** the engine rejects it at load time:

```
no library function matches token `<`
```

The lexer splits `<div>` into separate symbol tokens (`<`, `div`, `>`); it does
not keep `<div>` together, and `arg literal` matches a *single* token. The only
way to match an opening tag is to spell it as three literals:

```
function tag
    arg literal "<"
    arg literal "p"
    arg literal ">"
```

That *does* parse — but it forces every tag into a three-literal dance, and it
still leaves the close side unsolved (see §4–§6).

**What it would enable if present:** `arg literal "<div>"` as one clean token,
making the opener read like HTML without lexer gymnastics.

---

## 4. Missing feature 2 — per-tag symmetric block closers

**What's missing:** the ability for each tag to declare *its own* closing
terminator. HTML's whole structure is symmetric pairs — `<p>` is closed only by
`</p>`, `<div>` only by `</div>`.

**How Capy actually closes blocks.** Capy offers exactly these block modes:

- `block_closer NAME` — **one shared keyword** closes the block (the `end`
  model). Every block uses the same closer.
- `block_open "X" close "Y"` — **single-character** open/close delimiters
  (e.g. `{` … `}`), declared on one line.
- `block_dedent` — the block ends when indentation decreases.
- `block_verbatim NAME` — raw bytes until the dedent.
- `block_sections … closer C` — fixed internal sections, one shared closer.

There is **no mode** where the closer token varies per function instance. A
library can have many functions, but they converge on a shared closer keyword
or a single delimiter char. You cannot say "this block closes on `</p>` and that
one closes on `</div>`" — the per-tag symmetry HTML depends on simply isn't a
dimension Capy exposes.

**What it would enable if present:** true matched pairs, where the parser knows
`</p>` belongs to the nearest open `<p>` and errors on mismatched nesting —
exactly HTML's contract.

---

## 5. Missing feature 3 — multi-token close delimiters

**What's missing:** a closing delimiter that is more than one character / one
token. Even setting aside per-tag symmetry, `</div>` is itself unrepresentable
as a closer.

**Probe.** Attempting a multi-character close delimiter:

```
function div
    block_open ">" close "</div>"
    ...
```

**Result:** the engine errors on the directive form. `block_open … close …`
expects **single-character** delimiters; `</div>` is rejected because the close
side is a multi-character string the lexer breaks into `<`, `/div`, `>`.

So the closing tag faces the same lexer-splitting problem as §3, but on the side
where Capy has no three-literal escape hatch — there is no "match three literals
to *close* a block" construct.

**What it would enable if present:** `</p>`, `</div>`, `</span>` as real closing
delimiters instead of a bare `end`.

---

## 6. Missing feature 4 — closers that depend on the opener

**What's missing:** any link between the closing token and which function
opened the block. HTML requires this: a stray `</div>` inside a `<p>` is an
error, and tooling relies on the closer naming its opener.

Because Capy's closers are either a fixed shared keyword (§4) or a fixed
single char, the closer carries **no identity**. Even if multi-token closers
existed (§5), nothing in the grammar would tie a particular `</p>` back to a
particular `<p>` opener — the engine has no notion of "this terminator matches
that opener." Nesting validation by tag name is therefore impossible.

**What it would enable if present:** mismatch detection (`<p>…</div>` → error),
which is the single biggest correctness win HTML's verbosity buys you, and the
main reason one might want the angle-bracket form at all.

---

## 7. What Capy *can* do instead

The missing features above are exactly why [capy-html.md](capy-html.md) lands on
the keyword form. The good news: the features Capy *does* have cover the real
goals (controlled output, escaping, cross-platform normalization) without the
angle brackets:

- **Auto-name-prepend** — a function with no `arg literal` auto-prepends its own
  name as a literal, so `p "text"` and `div …` read as bare tags for free.
- **`when_followed_by indent` / `when_not_followed_by indent`** — one keyword can
  be both a flat leaf (`p "short text"`) and a block (`p` + indented children),
  recovering HTML's dual nature for text tags.
- **`block_verbatim NAME`** — raw multi-line content (large prose, `style`,
  `script`) with base indent stripped and relative indent preserved, escaped
  once. This is how long `<p>` bodies and literal `end`-as-content are handled.
- **Shared `end` closer + indentation** — `div` … `end` nests cleanly, and
  indentation disambiguates an inner literal `end` from the block closer.

Net: Capy can produce normalized, escaped, cross-platform HTML through a
keyword DSL — it just can't *look* like angle-bracket HTML, because the four
features in §3–§6 don't exist.

---

## 8. Summary table

| # | Missing feature | Probe result | Blocks |
|---|---|---|---|
| 1 | Multi-char symbol literal as one token | `arg literal "<div>"` → `no library function matches token \`<\`` | Clean `<div>` opener |
| 2 | Per-tag symmetric block closer | No grammar mode for per-instance closer | `</p>` closing only `<p>` |
| 3 | Multi-token close delimiter | `block_open ">" close "</div>"` → directive error | `</div>` as a real closer |
| 4 | Closer bound to its opener | Closers carry no identity | Mismatched-nesting detection |

All four would need to land together to make `<p>…</p>` / `<div>…</div>`
matched-pair HTML authorable in Capy. Today, none are present — so the keyword
form is not a stylistic preference alone, it's the only form the engine
supports.
