// capyx_testbench.js — the interactive Test Bench UI.
//
// Built on the same kernel (CAPYX_TEST_KIT) the headless runner uses, this is
// the "select a component, test it in full isolation" surface for humans —
// non-devs included. It reads __CAPYX_TEST__.meta to render: a component
// picker, a live isolated preview, an initial-state editor, event buttons,
// capability mock editors, and a one-click assertion + scenario recorder that
// produces a runnable .capytest you can save. No browser automation, no
// Selenium — every interaction runs in-process through the kernel.
//
// DOM usage is deliberately limited to createElement/createTextNode/appendChild
// (+ classList/value/addEventListener) so the bench runs unchanged under the
// Node DOM shim used by its own smoke test, and in the real webview.
(function (global) {
  "use strict";
  var K = global.CAPYX_TEST_KIT;
  var R = global.__CAPYX_TEST__;
  var doc = global.document;
  if (!K || !R || !doc) return;
  var meta = R.meta || {};

  var bench = {
    component: null,
    handler: null,
    overrides: {}, // field -> value
    mocks: {},     // cap -> method -> returnSpec
    steps: [],     // recorded .capytest steps
    recording: true,
  };

  // ── tiny DOM helpers (shim-safe) ───────────────────────────────────────────
  function el(tag, cls, txt) {
    var n = doc.createElement(tag);
    if (cls) n.className = cls;
    if (txt != null) n.appendChild(doc.createTextNode(String(txt)));
    return n;
  }
  function clear(n) { while (n.childNodes.length) n.removeChild(n.childNodes[0]); }
  function on(n, ev, fn) { n.addEventListener(ev, fn); return n; }
  function add(parent) {
    for (var i = 1; i < arguments.length; i++) if (arguments[i]) parent.appendChild(arguments[i]);
    return parent;
  }
  function setText(n, s) { clear(n); n.appendChild(doc.createTextNode(String(s))); return n; }

  // ── panes (built once) ─────────────────────────────────────────────────────
  var root, preview, stateReadout, domReadout, controls, scenarioPre, runStatus, stepList, compList;

  function build() {
    root = doc.getElementById("capyx-bench") || doc.body;
    clear(root);

    var header = el("div", "cb-header");
    add(header, el("strong", null, "🧪 capyx Test Bench"),
      el("span", "cb-sub", "— " + ((meta.app && meta.app.title) || "app") + " · components tested in full isolation"));
    add(root, header);

    var main = el("div", "cb-main");

    // left: component picker
    var left = el("div", "cb-left");
    add(left, el("div", "cb-h", "Components"));
    compList = el("div", "cb-list");
    add(left, compList);

    // center: preview + readouts
    var center = el("div", "cb-center");
    add(center, el("div", "cb-h", "Live preview (isolated)"));
    preview = el("div", "cb-preview");
    add(center, preview);
    add(center, el("div", "cb-h", "Rendered text"));
    domReadout = el("pre", "cb-readout");
    add(center, domReadout);
    add(center, el("div", "cb-h", "Handler state"));
    stateReadout = el("pre", "cb-readout");
    add(center, stateReadout);

    // right: controls
    var right = el("div", "cb-right");
    controls = el("div");
    add(right, controls);

    add(main, left, center, right);
    add(root, main);

    // bottom: scenario recorder + runner
    var bottom = el("div", "cb-bottom");
    var bar = el("div", "cb-row");
    add(bar, el("div", "cb-h", "Recorded scenario (.capytest)"));
    var runBtn = on(el("button", "cb-btn cb-primary", "▶ Run scenario"), "click", runScenario);
    var clearBtn = on(el("button", "cb-btn", "Clear"), "click", function () { resetSteps(); });
    add(bar, runBtn, clearBtn);
    add(bottom, bar);
    scenarioPre = el("pre", "cb-scenario");
    add(bottom, scenarioPre);
    runStatus = el("div", "cb-status");
    add(bottom, runStatus);
    stepList = el("div", "cb-steps");
    add(bottom, stepList);
    add(root, bottom);

    renderComponents();
    var names = Object.keys(R.components || {});
    if (names.length) select(names[0]);
  }

  function renderComponents() {
    clear(compList);
    Object.keys(R.components || {}).forEach(function (name) {
      var b = on(el("button", "cb-comp", name), "click", function () { select(name); });
      if (name === bench.component) b.className = "cb-comp cb-active";
      add(compList, b);
    });
  }

  // ── selection / mount ───────────────────────────────────────────────────────
  function defaultHandler(name) {
    var cm = (meta.components || {})[name] || {};
    return cm.defaultHandler || name;
  }

  function select(name) {
    bench.component = name;
    bench.overrides = {};
    bench.mocks = {};
    resetSteps();
    bench.steps.push({ op: "mount", component: name });
    remount();
    renderComponents();
    renderControls();
    renderScenario();
  }

  function remount() {
    clear(preview);
    K.reset();
    var m = K.mount(bench.component, { state: bench.overrides, mocks: bench.mocks, root: preview });
    bench.handler = m.handler;
    wireRecording();
    refreshReadouts();
  }

  // record real clicks in the preview as steps (so a non-dev just clicks).
  function wireRecording() {
    on(preview, "click", function (e) {
      if (!bench.recording) return;
      var t = e.target;
      var tag = String((t && (t._tag || t.tagName)) || "").toLowerCase();
      if (tag === "button" || tag === "a") {
        var label = (t.textContent || "").replace(/\s+/g, " ").trim();
        if (label) {
          bench.steps.push({ op: "click", text: label });
          // let the real handler run first, then refresh
          setTimeout(refreshReadouts, 0);
          renderScenario();
        }
      }
    });
  }

  function refreshReadouts() {
    try { setText(domReadout, K.text() || "(empty)"); } catch (e) { setText(domReadout, "—"); }
    var snap = {};
    var hm = (meta.handlers || {})[defaultHandler(bench.component)] || {};
    (hm.state || []).forEach(function (s) {
      try { snap[s.name] = bench.handler ? bench.handler[s.name] : undefined; } catch (e) {}
    });
    setText(stateReadout, JSON.stringify(snap, null, 2));
  }

  // ── right-hand controls ──────────────────────────────────────────────────────
  function renderControls() {
    clear(controls);
    var hname = defaultHandler(bench.component);
    var hm = (meta.handlers || {})[hname] || {};

    // initial state editor
    if ((hm.state || []).length) {
      add(controls, el("div", "cb-h", "Initial state"));
      hm.state.forEach(function (s) {
        var rowEl = el("div", "cb-field");
        add(rowEl, el("label", "cb-label", s.name));
        var inp = el("input", "cb-input");
        inp.value = jsonOr(bench.overrides[s.name], s.expr);
        var apply = on(el("button", "cb-btn", "set"), "click", function () {
          var v = parseLoose(inp.value);
          bench.overrides[s.name] = v;
          bench.steps.push({ op: "set", field: s.name, value: v });
          remount();
          renderScenario();
        });
        add(rowEl, inp, apply);
        add(controls, rowEl);
      });
    }

    // event buttons
    if ((hm.methods || []).length) {
      add(controls, el("div", "cb-h", "Events"));
      var evWrap = el("div", "cb-row");
      hm.methods.forEach(function (mth) {
        var label = mth.name + (mth.arity ? "(…)" : "()");
        var btn = on(el("button", "cb-btn", label), "click", function () {
          try {
            K.call(mth.name, []);
            bench.steps.push({ op: "call", method: mth.name });
          } catch (e) {}
          refreshReadouts();
          renderScenario();
        });
        add(evWrap, btn);
      });
      add(controls, evWrap);
    }

    // capability mocks
    if ((hm.ports || []).length) {
      add(controls, el("div", "cb-h", "Mock capabilities"));
      hm.ports.forEach(function (p) {
        add(controls, el("div", "cb-sub", p.cap + " (as " + p.name + ")"));
        var methods = (meta.capabilities || {})[p.cap] || [];
        methods.forEach(function (mthName) {
          var rowEl = el("div", "cb-field");
          add(rowEl, el("label", "cb-label", mthName + " →"));
          var inp = el("input", "cb-input");
          inp.value = (bench.mocks[p.cap] && JSON.stringify(bench.mocks[p.cap][mthName])) || "";
          var apply = on(el("button", "cb-btn", "mock"), "click", function () {
            var v = parseLoose(inp.value);
            bench.mocks[p.cap] = bench.mocks[p.cap] || {};
            bench.mocks[p.cap][mthName] = v;
            bench.steps.push({ op: "mock", cap: p.cap, method: mthName, returns: v });
            remount();
            renderScenario();
          });
          add(rowEl, inp, apply);
          add(controls, rowEl);
        });
      });
    }

    // assertions
    add(controls, el("div", "cb-h", "Assert (records expectation)"));
    var aRow = el("div", "cb-field");
    var aInp = el("input", "cb-input");
    aInp.value = "";
    var aBtn = on(el("button", "cb-btn", "text contains"), "click", function () {
      bench.steps.push({ op: "expectText", value: aInp.value });
      renderScenario();
    });
    add(aRow, el("label", "cb-label", "text"), aInp, aBtn);
    add(controls, aRow);
    var sBtn = on(el("button", "cb-btn", "snapshot state as expectations"), "click", function () {
      var hm2 = (meta.handlers || {})[defaultHandler(bench.component)] || {};
      (hm2.state || []).forEach(function (s) {
        bench.steps.push({ op: "expectState", field: s.name, value: bench.handler ? bench.handler[s.name] : null });
      });
      renderScenario();
    });
    add(controls, sBtn);
  }

  // ── scenario serialisation + running ─────────────────────────────────────────
  function resetSteps() {
    bench.steps = [];
    if (scenarioPre) renderScenario();
    if (runStatus) setText(runStatus, "");
    if (stepList) clear(stepList);
  }

  function stepToText(s) {
    switch (s.op) {
      case "mount": return "    mount " + s.component + (s.handler ? " use " + s.handler : "");
      case "set": return "    set " + s.field + " " + JSON.stringify(s.value);
      case "mock": return "    mock " + s.cap + "." + s.method + " returns " + JSON.stringify(s.returns);
      case "click": return "    click " + (s.selector || JSON.stringify(s.text));
      case "input": return "    input " + (s.selector || JSON.stringify(s.text)) + " " + JSON.stringify(s.value);
      case "fire": return "    fire " + s.event + " " + (s.selector || JSON.stringify(s.text));
      case "call": return "    call " + s.method + (s.args ? " " + JSON.stringify(s.args) : "");
      case "expectText": return "    expect " + (s.negate ? "no text " : "text ") + JSON.stringify(s.value);
      case "expectState": return "    expect state " + s.field + " " + JSON.stringify(s.value);
      case "expectCount": return "    expect count " + s.selector + " " + s.count;
      case "expectCalled": return "    expect called " + s.cap + "." + s.method + (s.count != null ? " " + s.count : "");
      case "expectClass": return "    expect class " + s.selector + " " + JSON.stringify(s["class"]);
    }
    return "    # " + s.op;
  }

  function renderScenario() {
    var lines = ['suite "' + ((meta.app && meta.app.title) || "app") + '"'];
    lines.push("use ./<app>.capyx");
    lines.push("");
    lines.push('scenario "' + (bench.component || "?") + ' — recorded"');
    bench.steps.forEach(function (s) { lines.push(stepToText(s)); });
    setText(scenarioPre, lines.join("\n"));
  }

  function runScenario() {
    var sc = { name: (bench.component || "?") + " — recorded", steps: bench.steps.slice() };
    var res = K.runScenario(sc);
    clear(stepList);
    res.steps.forEach(function (st) {
      var row = el("div", "cb-step " + (st.ok ? "cb-ok" : "cb-fail"));
      add(row, el("span", "cb-mark", st.ok ? "✓" : "✗"), el("span", "cb-op", st.op),
        st.message ? el("span", "cb-msg", st.message) : null);
      add(stepList, row);
    });
    setText(runStatus, res.ok ? "PASS — all steps green" : "FAIL — " +
      res.steps.filter(function (s) { return !s.ok; }).length + " step(s) failed" +
      (res.error ? " (" + res.error + ")" : ""));
    runStatus.className = "cb-status " + (res.ok ? "cb-ok" : "cb-fail");
    // re-mount to a clean state so the live preview matches a fresh run
    remount();
  }

  // ── value parsing for editor inputs ──────────────────────────────────────────
  function parseLoose(s) {
    s = String(s == null ? "" : s).trim();
    if (s === "") return "";
    try { return JSON.parse(s); } catch (e) { return s; }
  }
  function jsonOr(v, fallbackExpr) {
    if (v !== undefined) { try { return JSON.stringify(v); } catch (e) {} }
    return fallbackExpr || "";
  }

  global.CAPYX_BENCH = { build: build, state: bench };
  if (doc.readyState !== "loading") build();
  else doc.addEventListener("DOMContentLoaded", build);
})(typeof window !== "undefined" ? window : globalThis);
