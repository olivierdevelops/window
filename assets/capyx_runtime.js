// capyx_runtime.js — fine-grained reactive runtime for .capyx apps.
//
// A tiny signals core (track/trigger over reactive proxies) plus DOM binding
// helpers the compiler targets. The design goal of the .capyx VHCO model is
// surgical updates: mutating one field re-runs only the effects that read that
// field, never re-rendering a whole element. Each {{ expr }}, class:x={…} and
// attr={…} compiles to its own effect; {#for}/{#if}/{#match} compile to dynamic
// regions whose control effect tracks only the condition, while inner bindings
// own their own effects.
(function (global) {
  "use strict";

  // ── Reactive core ──────────────────────────────────────────────────────────
  var activeEffect = null;
  var activeScope = null;
  var targetMap = new WeakMap(); // target -> (key -> Set<effect>)
  var proxyMap = new WeakMap(); // raw -> proxy
  var rawMap = new WeakMap(); // proxy -> raw
  var ITER = Symbol("iter"); // structural (length/keys) dependency key

  function track(target, key) {
    if (!activeEffect) return;
    var deps = targetMap.get(target);
    if (!deps) {
      deps = new Map();
      targetMap.set(target, deps);
    }
    var dep = deps.get(key);
    if (!dep) {
      dep = new Set();
      deps.set(key, dep);
    }
    dep.add(activeEffect);
    activeEffect.deps.push(dep);
  }

  function trigger(target, key) {
    var deps = targetMap.get(target);
    if (!deps) return;
    var run = new Set();
    var dep = deps.get(key);
    if (dep) dep.forEach(function (e) { run.add(e); });
    if (key === ITER) {
      // structural change: nothing extra
    } else {
      var iter = deps.get(ITER);
      if (iter && (key === "length" || typeof key === "number")) {
        iter.forEach(function (e) { run.add(e); });
      }
    }
    run.forEach(function (e) { e.run(); });
  }

  function isObject(v) {
    return v !== null && (typeof v === "object" || typeof v === "function");
  }

  function reactive(obj) {
    if (!isObject(obj) || typeof obj === "function") return obj;
    if (rawMap.has(obj)) return obj; // already a proxy
    var existing = proxyMap.get(obj);
    if (existing) return existing;
    var p = new Proxy(obj, {
      get: function (t, k, r) {
        if (k === "__isReactive") return true;
        var res = Reflect.get(t, k, r);
        if (typeof k === "symbol") return res;
        track(t, k === "length" ? "length" : k);
        return reactive(res);
      },
      set: function (t, k, v, r) {
        var old = t[k];
        var raw = isObject(v) && rawMap.has(v) ? rawMap.get(v) : v;
        var had = Object.prototype.hasOwnProperty.call(t, k);
        var ok = Reflect.set(t, k, raw, r);
        if (Array.isArray(t)) {
          if (!had || old !== raw) trigger(t, ITER);
          trigger(t, k);
        } else if (!had) {
          trigger(t, k);
          trigger(t, ITER);
        } else if (old !== raw) {
          trigger(t, k);
        }
        return ok;
      },
      deleteProperty: function (t, k) {
        var had = Object.prototype.hasOwnProperty.call(t, k);
        var ok = Reflect.deleteProperty(t, k);
        if (had) {
          trigger(t, k);
          trigger(t, ITER);
        }
        return ok;
      },
    });
    proxyMap.set(obj, p);
    rawMap.set(p, obj);
    return p;
  }

  function effect(fn) {
    var e = {
      deps: [],
      active: true,
      run: function () {
        if (!e.active) return fn();
        cleanup(e);
        var prevE = activeEffect;
        activeEffect = e;
        try {
          return fn();
        } finally {
          activeEffect = prevE;
        }
      },
    };
    if (activeScope) activeScope.push(e);
    e.run();
    return e;
  }

  function cleanup(e) {
    for (var i = 0; i < e.deps.length; i++) e.deps[i].delete(e);
    e.deps.length = 0;
  }

  function runInScope(scope, fn) {
    var prev = activeScope;
    activeScope = scope;
    try {
      return fn();
    } finally {
      activeScope = prev;
    }
  }

  function disposeScope(scope) {
    for (var i = 0; i < scope.length; i++) {
      scope[i].active = false;
      cleanup(scope[i]);
    }
    scope.length = 0;
  }

  // ── Formatting ─────────────────────────────────────────────────────────────
  function fmt(v) {
    if (v === null || v === undefined) return "";
    if (typeof v === "object") {
      try {
        return JSON.stringify(v);
      } catch (e) {
        return String(v);
      }
    }
    return String(v);
  }

  // ── DOM helpers ────────────────────────────────────────────────────────────
  function el(tag, setup) {
    var node = document.createElement(tag);
    if (setup) setup(node);
    return node;
  }

  function staticText(s) {
    return document.createTextNode(s);
  }

  function text(getter) {
    var node = document.createTextNode("");
    effect(function () {
      node.data = fmt(getter());
    });
    return node;
  }

  function attr(node, name, getter) {
    effect(function () {
      var v = getter();
      if (v === false || v === null || v === undefined) node.removeAttribute(name);
      else if (v === true) node.setAttribute(name, "");
      else node.setAttribute(name, v);
    });
  }

  function prop(node, name, getter) {
    effect(function () {
      node[name] = getter();
    });
  }

  function cls(node, name, getter) {
    effect(function () {
      node.classList.toggle(name, !!getter());
    });
  }

  function style(node, name, getter) {
    effect(function () {
      node.style.setProperty(name, getter());
    });
  }

  function on(node, ev, handler) {
    node.addEventListener(ev, handler);
  }

  function model(node, get, set) {
    effect(function () {
      var v = fmt(get());
      if (node.type === "checkbox") {
        node.checked = !!get();
      } else if (node.value !== v) {
        node.value = v;
      }
    });
    var evt = node.tagName === "SELECT" ? "change" : "input";
    node.addEventListener(evt, function () {
      set(node.type === "checkbox" ? node.checked : node.value);
    });
  }

  // anchored dynamic region: render() returns Node | Node[] | null and is
  // re-run when its tracked deps change. Inner bindings own their own effects.
  function region(render) {
    var frag = document.createDocumentFragment();
    var end = document.createComment("");
    frag.appendChild(end);
    var current = [];
    var scope = [];
    effect(function () {
      var produced = runInScope(scope, render);
      for (var i = 0; i < current.length; i++) {
        if (current[i].parentNode) current[i].parentNode.removeChild(current[i]);
      }
      // a fresh scope per render so the previous render's inner effects die
      var fresh = [];
      for (var j = 0; j < scope.length; j++) fresh.push(scope[j]);
      current = [];
      if (produced) {
        var nodes = Array.isArray(produced) ? produced : [produced];
        for (var k = 0; k < nodes.length; k++) {
          if (nodes[k]) {
            end.parentNode.insertBefore(nodes[k], end);
            current.push(nodes[k]);
          }
        }
      }
    });
    return frag;
  }

  // when: branchKey() is the only tracked read; the chosen builder runs without
  // re-running on inner changes (its bindings own their effects).
  function when(branchKey, builders) {
    var lastKey;
    var lastScope = [];
    return region(function () {
      var key = branchKey();
      if (key === lastKey && lastScope.__node) return lastScope.__node;
      disposeScope(lastScope);
      lastScope = [];
      lastKey = key;
      var build = builders[key];
      var node = build ? runInScope(lastScope, build) : null;
      lastScope.__node = node;
      return node;
    });
  }

  // each: keyed list. The control effect tracks only the list structure; each
  // item's bindings track that item's fields, so mutating one item updates only
  // its own nodes.
  function each(getList, keyOf, builder, emptyBuilder) {
    var frag = document.createDocumentFragment();
    var end = document.createComment("");
    frag.appendChild(end);
    var cache = new Map(); // key -> {node, scope}
    var emptyEntry = null;
    effect(function () {
      var list = getList() || [];
      var len = list.length;
      var seen = new Set();
      var anchor = end;
      // Insert in order (insertBefore on an existing node moves it).
      for (var i = 0; i < len; i++) {
        var item = list[i];
        var key = keyOf ? keyOf(item, i) : i;
        seen.add(key);
        var entry = cache.get(key);
        if (!entry) {
          var scope = [];
          var node = runInScope(scope, (function (it, idx) {
            return function () { return builder(it, idx); };
          })(item, i));
          entry = { node: node, scope: scope };
          cache.set(key, entry);
        }
        end.parentNode.insertBefore(entry.node, end);
      }
      cache.forEach(function (entry, key) {
        if (!seen.has(key)) {
          if (entry.node.parentNode) entry.node.parentNode.removeChild(entry.node);
          disposeScope(entry.scope);
          cache.delete(key);
        }
      });
      if (len === 0 && emptyBuilder) {
        if (!emptyEntry) {
          var es = [];
          emptyEntry = { node: runInScope(es, emptyBuilder), scope: es };
        }
        end.parentNode.insertBefore(emptyEntry.node, end);
      } else if (emptyEntry) {
        if (emptyEntry.node.parentNode) emptyEntry.node.parentNode.removeChild(emptyEntry.node);
        disposeScope(emptyEntry.scope);
        emptyEntry = null;
      }
    });
    return frag;
  }

  // ── App bootstrap ──────────────────────────────────────────────────────────
  function mount(root, viewFn) {
    var node = viewFn();
    root.appendChild(node);
    return node;
  }

  function makeHandler(spec) {
    // spec: { state: {...}, methods: {name: fn}, ports: {...} }
    var base = {};
    var k;
    for (k in spec.state) base[k] = spec.state[k];
    for (k in spec.methods) {
      Object.defineProperty(base, k, {
        value: spec.methods[k],
        enumerable: false,
        writable: true,
      });
    }
    if (spec.ports) {
      // configurable so the reactive proxy's get trap may return a wrapped
      // value without violating the non-configurable/non-writable invariant.
      Object.defineProperty(base, "$ports", {
        value: spec.ports,
        enumerable: false,
        configurable: true,
        writable: true,
      });
    }
    var self = reactive(base);
    if (typeof base.mount === "function") {
      try {
        base.mount.call(self);
      } catch (e) {
        console.error("handler mount failed:", e);
      }
    }
    return self;
  }

  global.CAPYX = {
    reactive: reactive,
    effect: effect,
    el: el,
    text: text,
    staticText: staticText,
    attr: attr,
    prop: prop,
    cls: cls,
    style: style,
    on: on,
    model: model,
    when: when,
    each: each,
    region: region,
    mount: mount,
    makeHandler: makeHandler,
    fmt: fmt,
  };
})(typeof window !== "undefined" ? window : globalThis);
