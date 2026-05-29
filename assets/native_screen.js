(function () {
    window.NATIVE = window.NATIVE || {};
    var active = [];

    function record(stream, opts) {
        opts = opts || {};
        var chunks = [];
        var mr = new MediaRecorder(stream, opts.mimeType ? { mimeType: opts.mimeType } : undefined);
        mr.ondataavailable = function (e) { if (e.data.size) chunks.push(e.data); };
        var done = new Promise(function (resolve) {
            mr.onstop = function () {
                stream.getTracks().forEach(function (t) { t.stop(); });
                var blob = new Blob(chunks, { type: mr.mimeType });
                if (opts.blob) { resolve(blob); return; }
                var reader = new FileReader();
                reader.onloadend = function () { resolve(reader.result); };
                reader.readAsDataURL(blob);
            };
        });
        mr.start();
        if (opts.ms) setTimeout(function () { if (mr.state !== 'inactive') mr.stop(); }, opts.ms);
        return { stop: function () { if (mr.state !== 'inactive') mr.stop(); return done; }, done: done };
    }

    window.NATIVE.screen = {
        // Capture a still screenshot of a chosen screen/window. Returns a data URL.
        snapshot: async function (opts) {
            opts = opts || {};
            var stream = await navigator.mediaDevices.getDisplayMedia({ video: true });
            var video = document.createElement('video');
            video.srcObject = stream;
            await video.play();
            var canvas = document.createElement('canvas');
            canvas.width = video.videoWidth;
            canvas.height = video.videoHeight;
            canvas.getContext('2d').drawImage(video, 0, 0);
            stream.getTracks().forEach(function (t) { t.stop(); });
            return canvas.toDataURL(opts.type || 'image/png', opts.quality);
        },
        // Record the screen. opts: {audio, ms, blob, mimeType}. Returns { stop() -> Promise<dataURL> }.
        record: async function (opts) {
            opts = opts || {};
            var stream = await navigator.mediaDevices.getDisplayMedia({ video: true, audio: !!opts.audio });
            active.push(stream);
            return record(stream, opts);
        },
        // Raw display stream. opts: {audio, into}. `into` attaches it to a <video>.
        stream: async function (opts) {
            opts = opts || {};
            var stream = await navigator.mediaDevices.getDisplayMedia({ video: true, audio: !!opts.audio });
            active.push(stream);
            if (opts.into) {
                var el = typeof opts.into === 'string' ? document.querySelector(opts.into) : opts.into;
                if (el) { el.srcObject = stream; el.play && el.play().catch(function () {}); }
            }
            return stream;
        },
        stop: function () {
            active.forEach(function (s) { s.getTracks().forEach(function (t) { t.stop(); }); });
            active = [];
        },
    };
})();
