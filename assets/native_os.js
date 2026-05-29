(function () {
    window.NATIVE = window.NATIVE || {};
    window.NATIVE.os = {
        exec: function (cmd, args) {
            return __native_os_exec(cmd, args || []);
        },
        getEnv: function (key) {
            return __native_os_getEnv(key);
        },
        platform: function () {
            return __native_os_platform();
        },
        info: function () {
            return __native_os_info();
        },
    };
})();
