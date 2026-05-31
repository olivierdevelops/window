// capyx_testkit.js — the .capyx component test kernel.
//
// Runs in any JS environment (the Node DOM shim used by the headless .capytest
// runner, or the real webview used by the interactive Test Bench). It reads the
// globalThis.__CAPYX_TEST__ registry produced by the harness build and offers:
//
//   • mountIsolated(component, opts) — mount ONE component in a fresh, detached
//     root, with its initial state overridden and its capabilities replaced by
//     recording mocks. No siblings, no shared DOM — true isolation.
//   • drivers — click / input / fire / call — to act on the mounted component.
//   • mocks — canned return values (scalar, sequence, or function) per
//     capability method, with every call recorded for assertions.
//   • assertions — text / state / count / called / class.
//   • runScenario / runSuite — interpret a parsed .capytest spec and report
//     pass/fail per step. This is what makes browser-free, Selenium-free
//     interface testing possible.
(function (global) {
  "use strict";

  function reg() {
    var r = global.__CAPYX_TEST__;
    if (!r) throw new Error("no __CAPYX_TEST__ registry — build the app with the harness compiler");
    return r;
  }

  // ── tree walking (kernel-owned, independent of any host helpers) ───────────
  function walk(node, fn) {
    if (!node) return;
    fn(node);
    var kids = node.childNodes || [];
    for (var i = 0; i < kids.length; i++) walk(kids[i], fn);
  }

  function norm(s) {
    return String(s == null ? "" : s).replace(/\s+/g, " ").trim();
  }

  function isElement(n) {
    return n && (n.kind === "element" || n.nodeType === 1);
  }

  function tagOf(n) {
    return String(n._tag || n.tagName || "").toLowerCase();
  }

  function hasClass(n, c) {
    if (n.classList && n.classList.contains) return n.classList.contains(c);
    var cn = n.className || "";
    return (" " + cn + " ").indexOf(" " + c + " ") >= 0;
  }

  function matchesSelector(n, sel) {
    if (!isElement(n)) return false;
    if (sel === "*") return true;
    if (sel.charAt(0) === ".") return hasClass(n, sel.slice(1));
    return tagOf(n) === sel.toLowerCase();
  }

  // ── current mount ──────────────────────────────────────────────────────────
  var current = null; // { node, handler, root, calls, mockStore }
  // mocks set before a mount land here, then seed the next mount's store — so a
  // capability used by the handler's mount() lifecycle is already mocked.
  var pendingMocks = {}; // cap -> method -> returnSpec

  function freshRoot() {
    var el = global.document.createElement("div");
    el.setAttribute && el.setAttribute("data-capyx-isolated", "1");
    return el;
  }

  function resolveReturn(spec, callIndex) {
    if (typeof spec === "function") return spec;
    if (spec && spec.__seq && Array.isArray(spec.values)) {
      var v = spec.values[Math.min(callIndex, spec.values.length - 1)];
      return v;
    }
    return spec;
  }

  // build a mock provider object for one capability, reading canned returns from
  // mockStore at CALL time (so mocks can be set or changed before/after mount).
  function buildMock(cap, methodNames, calls, mockStore) {
    var obj = {};
    methodNames.forEach(function (m) {
      obj[m] = function () {
        var args = Array.prototype.slice.call(arguments);
        var entry = mockStore[cap] && mockStore[cap][m];
        var n = entry ? entry.calls++ : 0;
        calls.push({ cap: cap, method: m, args: args });
        if (!entry) return undefined;
        var r = resolveReturn(entry.spec, n);
        return typeof r === "function" ? r.apply(null, args) : r;
      };
    });
    return obj;
  }

  function unionMethods(cap, mockStore) {
    var meta = reg().meta || {};
    var caps = meta.capabilities || {};
    var set = {};
    (caps[cap] || []).forEach(function (m) { set[m] = true; });
    if (mockStore[cap]) for (var k in mockStore[cap]) set[k] = true;
    return Object.keys(set);
  }

  // mountIsolated(component, { handler, state, mocks, root }) → mount record.
  function mountIsolated(component, opts) {
    opts = opts || {};
    var R = reg();
    var componentFn = R.components[component];
    if (!componentFn) throw new Error("unknown component: " + component);
    var meta = R.meta || {};
    var cmeta = (meta.components || {})[component] || {};
    var handlerName = opts.handler || cmeta.defaultHandler || component;
    var factory = R.handlers[handlerName];

    var calls = [];
    var mockStore = {};
    // seed mocks: pending (set before this mount) first, then explicit opts.
    seedMocks(mockStore, pendingMocks);
    seedMocks(mockStore, opts.mocks);
    pendingMocks = {};

    var inst = null;
    if (factory) {
      var hmeta = (meta.handlers || {})[handlerName] || {};
      var ports = hmeta.ports || [];
      var saved = {};
      ports.forEach(function (p) {
        var provider = R.capImpl[p.cap];
        saved[provider] = R.providers[provider];
        R.providers[provider] = buildMock(p.cap, unionMethods(p.cap, mockStore), calls, mockStore);
      });
      try {
        inst = factory();
      } finally {
        for (var prov in saved) R.providers[prov] = saved[prov];
      }
      if (opts.state) for (var k in opts.state) inst[k] = opts.state[k];
    }

    var root = opts.root || freshRoot();
    var node = componentFn(inst);
    root.appendChild(node);
    current = { node: node, handler: inst, root: root, calls: calls, mockStore: mockStore };
    return current;
  }

  function requireMount() {
    if (!current) throw new Error("nothing mounted — call mount first");
    return current;
  }

  function seedMocks(store, src) {
    if (!src) return;
    for (var cap in src) {
      store[cap] = store[cap] || {};
      for (var m in src[cap]) store[cap][m] = { spec: src[cap][m], calls: 0 };
    }
  }

  // set/clear a mock return (scalar, sequence via seq([...]), or function).
  // Before a mount it buffers (so the handler's mount() lifecycle sees it);
  // after a mount it updates the live store, so behaviour can change mid-test.
  function setMock(cap, method, returns) {
    if (current) {
      current.mockStore[cap] = current.mockStore[cap] || {};
      current.mockStore[cap][method] = { spec: returns, calls: 0 };
    } else {
      pendingMocks[cap] = pendingMocks[cap] || {};
      pendingMocks[cap][method] = returns;
    }
  }

  function seq(values) { return { __seq: true, values: values }; }

  // ── finders / drivers ────────────────────────────────────────────────────
  function findAll(target) {
    var m = requireMount();
    var out = [];
    if (typeof target === "string") target = { selector: target };
    walk(m.root, function (n) {
      if (!isElement(n)) return;
      if (target.selector && !matchesSelector(n, target.selector)) return;
      if (target.text != null) {
        var t = norm(n.textContent);
        if (target.exact ? t !== norm(target.text) : t.indexOf(norm(target.text)) < 0) return;
      }
      out.push(n);
    });
    return out;
  }

  var INTERACTIVE = { button: 1, a: 1, input: 1, select: 1, textarea: 1, label: 1 };

  // find the best single match. For text targets, prefer the most specific
  // element (shortest text), breaking ties toward interactive controls — so
  // `click "add"` hits the <button>add</button>, not an ancestor row that
  // merely contains it.
  function find(target) {
    var all = findAll(target);
    if (typeof target === "object" && target.text != null) {
      all.sort(function (a, b) {
        var la = norm(a.textContent).length, lb = norm(b.textContent).length;
        if (la !== lb) return la - lb;
        return (INTERACTIVE[tagOf(a)] ? 0 : 1) - (INTERACTIVE[tagOf(b)] ? 0 : 1);
      });
    }
    return all[0] || null;
  }

  function dispatch(node, ev) {
    if (!node) throw new Error("element not found for event " + ev);
    if (node.dispatch) node.dispatch(ev); // Node shim
    else if (node.dispatchEvent) node.dispatchEvent(new global.Event(ev, { bubbles: true }));
  }

  function click(target) { dispatch(find(target), "click"); }

  function input(target, value) {
    var node = find(target);
    if (!node) throw new Error("input element not found");
    if (node.type === "checkbox") { node.checked = !!value; }
    else { node.value = String(value); }
    dispatch(node, "input");
  }

  function fire(event, target) { dispatch(find(target), event); }

  function call(method, args) {
    var m = requireMount();
    if (!m.handler || typeof m.handler[method] !== "function") {
      throw new Error("handler has no method: " + method);
    }
    return m.handler[method].apply(m.handler, args || []);
  }

  function state(field) { return requireMount().handler ? requireMount().handler[field] : undefined; }
  function text() { return norm(requireMount().root.textContent); }
  function calls() { return requireMount().calls; }

  function deepEq(a, b) {
    if (a === b) return true;
    try { return JSON.stringify(a) === JSON.stringify(b); } catch (e) { return false; }
  }

  // ── assertions: each returns { ok, message } ───────────────────────────────
  function ok(b, msg) { return { ok: !!b, message: msg }; }

  var assert = {
    text: function (sub, negate) {
      var has = text().indexOf(norm(sub)) >= 0;
      var pass = negate ? !has : has;
      return ok(pass, (negate ? "expected NOT to find text " : "expected text ") + JSON.stringify(sub) +
        (pass ? "" : " — actual: " + JSON.stringify(text())));
    },
    state: function (field, want) {
      var got = state(field);
      return ok(deepEq(got, want), "state " + field + " = " + JSON.stringify(got) + ", want " + JSON.stringify(want));
    },
    count: function (selector, want) {
      var got = findAll({ selector: selector }).length;
      return ok(got === want, "count(" + selector + ") = " + got + ", want " + want);
    },
    called: function (cap, method, wantN) {
      var n = 0;
      calls().forEach(function (c) { if (c.cap === cap && c.method === method) n++; });
      if (wantN == null) return ok(n > 0, cap + "." + method + " called " + n + " times, want ≥1");
      return ok(n === wantN, cap + "." + method + " called " + n + " times, want " + wantN);
    },
    cls: function (selector, cls) {
      var nodes = findAll({ selector: selector });
      var found = nodes.some(function (n) { return hasClass(n, cls); });
      return ok(found, "no " + selector + " has class " + JSON.stringify(cls));
    },
  };

  // ── scenario / suite interpreter ───────────────────────────────────────────
  // A step is { op, ... }. Returns { name, ok, steps:[{op,ok,message}], error }.
  function runScenario(sc) {
    current = null;
    pendingMocks = {};
    var result = { name: sc.name, ok: true, steps: [], error: null };
    function record(op, r) {
      var entry = { op: op, ok: r.ok, message: r.message || "" };
      result.steps.push(entry);
      if (!r.ok) result.ok = false;
    }
    try {
      var steps = sc.steps || [];
      for (var i = 0; i < steps.length; i++) {
        var s = steps[i];
        switch (s.op) {
          case "mount":
            mountIsolated(s.component, { handler: s.handler });
            record("mount " + s.component, ok(true));
            break;
          case "set":
            if (current && current.handler) current.handler[s.field] = s.value;
            record("set " + s.field, ok(!!(current && current.handler), current ? "" : "no handler to set state on"));
            break;
          case "mock":
            setMock(s.cap, s.method, s.returns);
            record("mock " + s.cap + "." + s.method, ok(true));
            break;
          case "click":
            click(s.selector ? { selector: s.selector } : { text: s.text });
            record("click " + (s.selector || JSON.stringify(s.text)), ok(true));
            break;
          case "input":
            input(s.selector ? { selector: s.selector } : { text: s.text }, s.value);
            record("input", ok(true));
            break;
          case "fire":
            fire(s.event, s.selector ? { selector: s.selector } : { text: s.text });
            record("fire " + s.event, ok(true));
            break;
          case "call":
            call(s.method, s.args);
            record("call " + s.method, ok(true));
            break;
          case "expectText":
            record("expect text", assert.text(s.value, s.negate));
            break;
          case "expectState":
            record("expect state", assert.state(s.field, s.value));
            break;
          case "expectCount":
            record("expect count", assert.count(s.selector, s.count));
            break;
          case "expectCalled":
            record("expect called", assert.called(s.cap, s.method, s.count));
            break;
          case "expectClass":
            record("expect class", assert.cls(s.selector, s["class"]));
            break;
          default:
            record(s.op || "?", ok(false, "unknown step op: " + s.op));
        }
      }
    } catch (e) {
      result.ok = false;
      result.error = String(e && e.message ? e.message : e);
    }
    return result;
  }

  function runSuite(suite) {
    var out = { name: suite.name || "", results: [], passed: 0, failed: 0 };
    (suite.scenarios || []).forEach(function (sc) {
      var r = runScenario(sc);
      out.results.push(r);
      if (r.ok) out.passed++; else out.failed++;
    });
    return out;
  }

  global.CAPYX_TEST_KIT = {
    mount: mountIsolated,
    setMock: setMock,
    seq: seq,
    find: find,
    findAll: findAll,
    click: click,
    input: input,
    fire: fire,
    call: call,
    state: state,
    text: text,
    calls: calls,
    assert: assert,
    runScenario: runScenario,
    runSuite: runSuite,
    current: function () { return current; },
    reset: function () { current = null; pendingMocks = {}; },
  };
})(typeof window !== "undefined" ? window : globalThis);
