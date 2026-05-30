# CapyScript (`.cs`) demos

CapyScript is a tiny JavaScript-like scripting language that **compiles to real
JavaScript** and runs inside a native window. The whole language is defined in
[`assets/capyscript.capy`](../../assets/capyscript.capy) — a Capy library, the
same metaprogramming engine that powers `.window` and `.htmlx`.

Run any demo with the `window` binary:

```sh
window demos/cs/fib.cs
```

A console-style window opens; everything you `log` is printed onto the page.

## Language

| CapyScript | Compiles to |
|---|---|
| `let x = 1` / `const y = 2` | `let x = 1;` / `const y = 2;` |
| `fn name(args) … end` | `function name(args) { … }` |
| `if cond … else … end` | `if (cond) { … } else { … }` |
| `for v in iter … end` | `for (const v of iter) { … }` |
| `while cond … end` | `while (cond) { … }` |
| `return expr` | `return expr;` |
| `log expr` | `console.log(expr)` (mirrored on-page) |
| `do EXPR` | `EXPR;` — escape hatch for any raw JS (assignment, arrow fns, breaks) |

Blocks (`fn`, `if`, `for`, `while`) all close with `end`. There is no
catch-all line rule — arbitrary statements go through `do` so the grammar stays
unambiguous.

## Demos

| File | Shows |
|---|---|
| `hello.cs` | minimal program; `log` + `const` |
| `fib.cs` | recursion, `if/else`, `while` |
| `fizzbuzz.cs` | nested `if/else`, `for … in` over a range |
| `primes.cs` | `while`, early `return`, `do` for mutation |
| `closures.cs` | higher-order functions via `do` passing arrow fns through |
