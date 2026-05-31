// dom_shim.js — a minimal DOM used to execute compiled .capyx apps under Node
// in tests (no browser). Implements just enough of the DOM for the capyx
// runtime: element/text/comment/fragment nodes, insertBefore/appendChild/
// removeChild, attributes, classList, style, and event dispatch.
(function (g) {
  function Node(kind) {
    this.kind = kind;
    this.childNodes = [];
    this.parentNode = null;
    this.listeners = {};
    this.attributes = {};
    this._classes = new Set();
    this.style = {
      _p: {},
      setProperty: function (k, v) { this._p[k] = v; },
    };
    this.data = "";
    this.value = "";
    this.checked = false;
    this.type = "";
  }
  Object.defineProperty(Node.prototype, "className", {
    get: function () { return Array.from(this._classes).join(" "); },
    set: function (s) {
      this._classes = new Set(String(s).split(/\s+/).filter(Boolean));
      this.attributes["class"] = s;
    },
  });
  Object.defineProperty(Node.prototype, "classList", {
    get: function () {
      var self = this;
      return {
        add: function (c) { self._classes.add(c); },
        remove: function (c) { self._classes.delete(c); },
        contains: function (c) { return self._classes.has(c); },
        toggle: function (c, force) {
          var has = self._classes.has(c);
          var want = force === undefined ? !has : !!force;
          if (want) self._classes.add(c); else self._classes.delete(c);
          return want;
        },
      };
    },
  });
  Object.defineProperty(Node.prototype, "tagName", {
    get: function () { return (this._tag || "").toUpperCase(); },
  });
  Object.defineProperty(Node.prototype, "textContent", {
    get: function () {
      if (this.kind === "text") return this.data;
      var out = "";
      for (var i = 0; i < this.childNodes.length; i++) out += this.childNodes[i].textContent;
      return out;
    },
  });
  Node.prototype.setAttribute = function (k, v) { this.attributes[k] = String(v); if (k === "class") this.className = v; };
  Node.prototype.removeAttribute = function (k) { delete this.attributes[k]; };
  Node.prototype.getAttribute = function (k) { return this.attributes[k]; };
  Node.prototype.addEventListener = function (ev, fn) { (this.listeners[ev] = this.listeners[ev] || []).push(fn); };
  Node.prototype.removeEventListener = function () {};
  Node.prototype.dispatch = function (ev) {
    var ls = this.listeners[ev] || [];
    for (var i = 0; i < ls.length; i++) ls[i].call(this, { type: ev, target: this });
  };
  Node.prototype.insertBefore = function (node, ref) {
    if (node.kind === "fragment") {
      var kids = node.childNodes.slice();
      node.childNodes = [];
      for (var i = 0; i < kids.length; i++) this.insertBefore(kids[i], ref);
      return node;
    }
    if (node.parentNode) node.parentNode.removeChild(node);
    if (ref == null) {
      this.childNodes.push(node);
    } else {
      var idx = this.childNodes.indexOf(ref);
      if (idx < 0) this.childNodes.push(node);
      else this.childNodes.splice(idx, 0, node);
    }
    node.parentNode = this;
    return node;
  };
  Node.prototype.appendChild = function (node) { return this.insertBefore(node, null); };
  Node.prototype.removeChild = function (node) {
    var idx = this.childNodes.indexOf(node);
    if (idx >= 0) this.childNodes.splice(idx, 1);
    node.parentNode = null;
    return node;
  };
  Node.prototype.remove = function () { if (this.parentNode) this.parentNode.removeChild(this); };
  Node.prototype.querySelectorAll = function (sel) {
    var out = [];
    walk(this, function (n) {
      if (n.kind !== "element") return;
      if (sel === "*") { out.push(n); return; }
      if (sel[0] === ".") { if (n._classes.has(sel.slice(1))) out.push(n); return; }
      if (n._tag === sel.toLowerCase()) out.push(n);
    });
    return out;
  };
  Node.prototype.querySelector = function (sel) { return this.querySelectorAll(sel)[0] || null; };

  function walk(node, fn) {
    fn(node);
    for (var i = 0; i < node.childNodes.length; i++) walk(node.childNodes[i], fn);
  }

  function mkEl(tag) { var n = new Node("element"); n._tag = tag; return n; }

  var root = mkEl("div");
  root.attributes.id = "app";

  g.document = {
    readyState: "complete",
    createElement: function (tag) { return mkEl(tag); },
    createTextNode: function (s) { var n = new Node("text"); n.data = String(s); return n; },
    createComment: function (s) { var n = new Node("comment"); n.data = String(s); return n; },
    createDocumentFragment: function () { return new Node("fragment"); },
    getElementById: function (id) { return id === "app" ? root : null; },
    addEventListener: function () {},
    body: root,
  };
  g.window = g;
  g.__APP_ROOT__ = root;
  g.__walk__ = walk;
})(globalThis);
