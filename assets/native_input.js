(function () {
    window.NATIVE = window.NATIVE || {};
    var pos = { x: 0, y: 0 };
    window.addEventListener('mousemove', function (e) { pos.x = e.clientX; pos.y = e.clientY; }, true);

    function key(e) {
        return {
            key: e.key, code: e.code, type: e.type,
            ctrl: e.ctrlKey, shift: e.shiftKey, alt: e.altKey, meta: e.metaKey,
            repeat: e.repeat,
        };
    }

    window.NATIVE.input = {
        // Listen for key events. opts: {up} to also fire on keyup. Returns { stop() }.
        onKey: function (callback, opts) {
            opts = opts || {};
            function handler(e) { callback(key(e)); }
            window.addEventListener('keydown', handler, true);
            if (opts.up) window.addEventListener('keyup', handler, true);
            return { stop: function () {
                window.removeEventListener('keydown', handler, true);
                window.removeEventListener('keyup', handler, true);
            } };
        },
        // Bind a hotkey combo like "ctrl+s" or "ctrl+shift+k". Returns { stop() }.
        hotkey: function (combo, callback) {
            var parts = combo.toLowerCase().split('+').map(function (p) { return p.trim(); });
            var mods = ['ctrl', 'shift', 'alt', 'meta', 'cmd'];
            var want = {
                ctrl: parts.indexOf('ctrl') >= 0,
                shift: parts.indexOf('shift') >= 0,
                alt: parts.indexOf('alt') >= 0,
                meta: parts.indexOf('meta') >= 0 || parts.indexOf('cmd') >= 0,
            };
            var target = parts.filter(function (p) { return mods.indexOf(p) < 0; })[0];
            function handler(e) {
                if (e.ctrlKey === want.ctrl && e.shiftKey === want.shift &&
                    e.altKey === want.alt && e.metaKey === want.meta &&
                    e.key.toLowerCase() === target) {
                    e.preventDefault();
                    callback(key(e));
                }
            }
            window.addEventListener('keydown', handler, true);
            return { stop: function () { window.removeEventListener('keydown', handler, true); } };
        },
        // Listen for mouse events. opts: {events: [...]}. Default: mousedown, mouseup, click. Returns { stop() }.
        onMouse: function (callback, opts) {
            opts = opts || {};
            var events = opts.events || ['mousedown', 'mouseup', 'click'];
            function handler(e) {
                callback({
                    type: e.type, x: e.clientX, y: e.clientY, button: e.button,
                    dx: e.movementX, dy: e.movementY, deltaY: e.deltaY,
                });
            }
            events.forEach(function (ev) { window.addEventListener(ev, handler, true); });
            return { stop: function () {
                events.forEach(function (ev) { window.removeEventListener(ev, handler, true); });
            } };
        },
        // Current pointer position within the window: {x, y}.
        pointer: function () { return { x: pos.x, y: pos.y }; },
    };
})();
