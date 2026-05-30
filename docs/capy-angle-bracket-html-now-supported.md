# Angle-bracket HTML is now supported — reply to capy-missing-features.md

> **Status: shipped.** This document is a direct reply to
> [capy-missing-features.md](capy-missing-features.md). That doc was correct
> *at the time it was written*: the four features it catalogs really were
> missing, and the keyword form in [capy-html.md](capy-html.md) was the only
> honest design then. **All four have since landed** in Capy (the "round-6"
> grammar work: `block_close_seq` + function-as-type captures). Symmetric,
> matched-pair angle-bracket HTML — `<p>…</p>`, `<div>…</div>`, with
> mismatched-nesting detection — is now buildable. Everything below is
> grounded in direct probes against the current Capy CLI.
>
> Related: [capy-missing-features.md](capy-missing-features.md) (the original
> findings), [capy-html.md](capy-html.md), [using-capy.md](using-capy.md).

---

## Table of contents

1. [TL;DR](#1-tldr)
2. [The two new primitives](#2-the-two-new-primitives)
3. [Re-checking the four "missing" features](#3-re-checking-the-four-missing-features)
4. [A complete angle-bracket HTML library](#4-a-complete-angle-bracket-html-library)
5. [Mismatched-nesting detection (the big win)](#5-mismatched-nesting-detection-the-big-win)
6. [The one real caveat: text nodes](#6-the-one-real-caveat-text-nodes)
7. [Should you switch from the keyword form?](#7-should-you-switch-from-the-keyword-form)
8. [Summary table (revisited)](#8-summary-table-revisited)

---

## 1. TL;DR

Your original doc said all four of these would "need to land together":

| # | Missing feature (your §) | Status now |
|---|---|---|
| 1 | Multi-char symbol literal as one token (§3) | **Not needed** — see below |
| 2 | Per-tag symmetric block closer (§4) | ✅ **Shipped** |
| 3 | Multi-token close delimiter (§5) | ✅ **Shipped** |
| 4 | Closer bound to its opener (§6) | ✅ **Shipped** |

The mechanism is one new block mode plus one new capture kind:

- **`block_close_seq`** — a block can close on an exact *sequence of tokens*
  (e.g. `</`, `div`, `>`), not just one keyword or one delimiter char.
- **Capture-bound closer** — a segment of that sequence can be a **reference
  to a capture the opener bound**, so the closer *depends on the opener*.
- **Function-as-type captures** — an `arg capture` can name another library
  function (a "named nonterminal"), with `*` / `+` repetition and an optional
  `sep`. This is how attributes (`class="card" id="x" …`) are matched.

Feature #1 (a literal `"<div>"` token) turns out to be unnecessary: you
capture the tag *name* and let the closer reference it, so a **single**
function handles every tag generically.

---

## 2. The two new primitives

### `block_close_seq` — multi-token sequence closers

```
function p
    arg literal "<"
    arg literal "p"
    arg literal ">"
    block_close_seq "</p>"
    write `<p>${body}</p>`
end
```

`block_close_seq` lists the closer's segments. Each segment is either a
**quoted literal** (pre-tokenized the way the lexer sees it — `"</p>"`
becomes `</`, `p`, `>`) or a **bare capture-name reference**. Inside a
sequence-closed block, newlines and indentation are insignificant — the
structure comes from the tags, exactly like real HTML.

This directly answers your §5 ("multi-token close delimiters"): `</p>`,
`</div>`, `</span>` are now real closers.

### Capture-bound closers — one function for *every* tag

Reference a captured value in the closer and the closer depends on the
opener. Capture the tag name once and `</NAME>` closes only the matching
tag:

```
function element
    arg literal "<"
    arg capture name ident
    arg literal ">"
    block_close_seq "</" name ">"      # `name` is a ref to the capture
    write `<${name}>${body}</${name}>`
end
```

Now `<div>` closes only on `</div>` and `<p>` only on `</p>` — from **one**
function. This answers both your §4 (per-tag symmetric closers) and §6
(closer bound to opener).

### Function-as-type captures — for attributes

An `arg capture`'s type may name another function. Add `*` / `+` to repeat:

```
function attribute
    arg capture key ident
    arg literal "="
    arg capture val raw
    write ` ${key}="${val}"`
end
```

`arg capture attrs attribute*` then matches zero-or-more attributes on the
opener. (`sep "X"` adds a separator literal between repetitions when you
need one.)

---

## 3. Re-checking the four "missing" features

Each of your probes, re-run against today's engine:

### §3 — multi-character symbol literals → *sidestepped, not needed*

Your probe `arg literal "<div>"` still fails — the lexer still splits
punctuation runs, by design. **But this was never the real blocker.** You
don't write `<div>` as one literal; you write `<` + a captured `name` +
`>`, and the captured name drives the closer. One function, every tag. The
"three-literal dance" you lamented collapses to a single capture.

### §4 — per-tag symmetric block closers → ✅ solved

`block_close_seq "</" name ">"` makes the closer vary per *instance*: the
bound `name` is substituted at parse time, so each open tag demands its own
matching close tag. This is precisely the "closer token varies per function
instance" dimension your doc said Capy didn't expose.

### §5 — multi-token close delimiters → ✅ solved

`block_close_seq` *is* a multi-token closer. `</div>` (three lexical pieces)
is a first-class closing sequence. The whitespace tolerance even lets `</p >`
close a `<p>`.

### §6 — closers that depend on the opener → ✅ solved

A ref segment ties the closer to a capture the opener bound. The engine now
has exactly the "this terminator matches that opener" notion your doc said
was absent — so a stray `</div>` inside a `<p>` is a hard parse error (see §5
below for the demo).

---

## 4. A complete angle-bracket HTML library

This single library parses arbitrary well-formed HTML (and XML — the same
mechanism is tag-agnostic). Drop it in as `lib.capy`:

```
# Generic angle-bracket HTML: ONE function parses any <tag …>…</tag>.
extension html

function element
    arg literal "<"
    arg capture name ident
    arg capture attrs attribute*
    arg literal ">"
    block_close_seq "</" name ">"
    write `<${name}${attrs}>${body}</${name}>`
end

# An attribute is a named nonterminal: key="value".
# Use `raw` (one token) for the value, NOT `string` — see §6.
function attribute
    arg capture key ident
    arg literal "="
    arg capture val raw
    write ` ${key}="${val}"`
end

# Text node: a quoted string becomes escaped literal content.
function text
    bare
    arg capture s raw
    write `${escapeHtml s}`
end
```

Source (`script.capy`), in the exact multi-line, indented shape your doc
opened with:

```
<div class="card">
  <p>"Hello, "<b>"world"</b>"."</p>
</div>
```

Output (verified):

```html
<div class="card"><p>Hello, <b>world</b>.</p></div>
```

Indentation and line breaks inside the tags are insignificant, so you can
format the source however you like. Attributes, nesting, and inline tags all
work from this one `element` function. (If you prefer explicit per-tag
functions — e.g. to give `<p>` and `<div>` different templates or
validation — write one function per tag with `block_close_seq "</p>"`,
`block_close_seq "</div>"`, etc. Both styles are supported.)

This sample ships in the Capy repo as
[`samples/html-xml-parser/`](https://github.com/olivierdevelops/capy/tree/main/samples/html-xml-parser)
and is interactive in the [playground](https://olivierdevelops.github.io/capy/playground/)
under **✨ Features → Parse HTML / XML**.

---

## 5. Mismatched-nesting detection (the big win)

This was, in your words, "the single biggest correctness win HTML's verbosity
buys you." It now works. Each open tag demands its own matching close tag:

```
# <p> is never closed; </div> can't satisfy it.
<div class="card"><p>"hi"</div>
```

```
error: no library function matches token "</"
  hint: did you mean "<"?
  1 │ <div class="card"><p>"hi"</div>
    │                   ... ^
```

Likewise a wrong closer (`<p>"hi"</b>`) or an unclosed tag at EOF
(`<p>"hi"`) are both hard parse errors. The parser knows `</p>` belongs to
the nearest open `<p>` and rejects anything else.

---

## 6. The one real caveat: text nodes

The structural problem (§3–§6) is fully solved. The remaining wrinkle is
**bare prose between tags**. In the library above, text is supplied as a
quoted string (`"Hello, "`) rather than bare words (`Hello,`). Two reasons:

1. **Bare prose mixed with inline tags is ambiguous.** `Hello <b>world</b>`
   would need a catch-all that stops at the next `<`; that interacts awkwardly
   with the tag grammar. Quoted text nodes keep the boundary crisp.
2. **Use `raw`, not `string`, for delimited values.** A `string` capture runs
   through the expression parser, which treats a following `>` as a *less-than/
   greater-than operator* and swallows it — breaking the tag. `raw` consumes
   exactly one token, so the `>` that closes the tag stays a tag. (This applies
   to attribute values too, which is why `attribute` above uses `arg capture
   val raw`.)

If bare, unquoted prose between tags matters for your authors, the keyword
form in [capy-html.md](capy-html.md) still handles that more naturally (a
`tail` capture slurps the rest of the line). So the trade is:

- **Angle-bracket form** (this doc): real `<tag>` syntax + matched-pair
  validation; text nodes are quoted.
- **Keyword form** (capy-html.md): bare prose flows freely; structure comes
  from indentation + a shared `end`.

Both are now viable. The angle-bracket form is no longer *impossible* — it's a
design choice with one ergonomic cost (quoted text nodes).

---

## 7. Should you switch from the keyword form?

Not necessarily — it depends on what you're optimizing for:

| If you want… | Use… |
|---|---|
| Source that *looks* like HTML; matched-pair validation; tag-name mismatch errors | **Angle-bracket form** (§4) |
| Bare unquoted prose; minimal punctuation; indentation-driven structure | **Keyword form** ([capy-html.md](capy-html.md)) |
| Both, for different authors | Ship both libraries; they target the same HTML output |

The point of this reply isn't "abandon the keyword form" — it's that the
angle-bracket form is **no longer blocked by the engine**. The decision is now
purely about authoring ergonomics, not capability.

---

## 8. Summary table (revisited)

Your original §8 table, updated:

| # | Feature | Then | Now |
|---|---|---|---|
| 1 | Multi-char symbol literal as one token | Missing | **Unnecessary** — capture the name instead |
| 2 | Per-tag symmetric block closer | Missing | ✅ `block_close_seq` |
| 3 | Multi-token close delimiter | Missing | ✅ `block_close_seq` segments |
| 4 | Closer bound to its opener | Missing | ✅ capture-name ref segment (`"</" name ">"`) |

All four are resolved. `<p>…</p>` / `<div>…</div>` matched-pair HTML is
authorable today.

### Where to read more

- Block functions, Mode C + named nonterminals:
  [block-functions.md](https://olivierdevelops.github.io/capy/block-functions/)
- One-page grammar brief:
  [CAPY_FOR_LLMS.md](https://olivierdevelops.github.io/capy/CAPY_FOR_LLMS/)
- Runnable sample:
  [`samples/html-xml-parser/`](https://github.com/olivierdevelops/capy/tree/main/samples/html-xml-parser)
- Live playground: **✨ Features → Parse HTML / XML**
  ([playground](https://olivierdevelops.github.io/capy/playground/))
