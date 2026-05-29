(function () {
    window.NATIVE = window.NATIVE || {};
    window.NATIVE.canvas = {
        drawRect: function (opts) {
            return __native_canvas_drawRect(opts);
        },
        drawText: function (opts) {
            return __native_canvas_drawText(opts);
        },
        clear: function (canvasId) {
            return __native_canvas_clear(canvasId);
        },
        flush: function (canvasId) {
            return __native_canvas_flush(canvasId);
        },
    };
})();
