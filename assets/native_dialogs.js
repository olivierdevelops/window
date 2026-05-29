(function () {
    window.NATIVE = window.NATIVE || {};
    window.NATIVE.dialogs = {
        openFile: function (opts) {
            return __native_dialogs_openFile(opts || {});
        },
        saveFile: function (opts) {
            return __native_dialogs_saveFile(opts || {});
        },
        showMessage: function (opts) {
            return __native_dialogs_showMessage(opts || {});
        },
    };
})();
