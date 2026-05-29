(function () {
    window.NATIVE = window.NATIVE || {};
    window.NATIVE.fs = {
        readFile: function (path) {
            return __native_fs_readFile(path);
        },
        writeFile: function (path, data, perm) {
            return __native_fs_writeFile(path, data, perm == null ? 420 : perm);
        },
        readDir: function (path) {
            return __native_fs_readDir(path);
        },
        watchFile: function (path, callback) {
            window.addEventListener('native:fs:watch:' + path, function (e) {
                if (callback) callback(e.detail.content);
            });
            return __native_fs_watchFile(path);
        },
    };
})();
