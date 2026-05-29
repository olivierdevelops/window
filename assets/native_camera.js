(function () {
    window.NATIVE = window.NATIVE || {};
    var active = [];

    // Wrap a MediaStream in a MediaRecorder; resolves stop() to a data URL (or Blob).
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

    window.NATIVE.camera = {
        // Live camera stream. opts: {audio, facingMode, into}. `into` attaches it to a <video>.
        stream: async function (opts) {
            opts = opts || {};
            var stream = await navigator.mediaDevices.getUserMedia({
                video: opts.facingMode ? { facingMode: opts.facingMode } : true,
                audio: !!opts.audio,
            });
            active.push(stream);
            if (opts.into) {
                var el = typeof opts.into === 'string' ? document.querySelector(opts.into) : opts.into;
                if (el) { el.srcObject = stream; el.play && el.play().catch(function () {}); }
            }
            return stream;
        },
        // Capture a single frame. opts: {width, height, type, quality, facingMode}. Returns a data URL.
        snapshot: async function (opts) {
            opts = opts || {};
            var stream = await navigator.mediaDevices.getUserMedia({
                video: opts.facingMode ? { facingMode: opts.facingMode } : true,
            });
            var video = document.createElement('video');
            video.srcObject = stream;
            await video.play();
            var w = opts.width || video.videoWidth;
            var h = opts.height || video.videoHeight;
            var canvas = document.createElement('canvas');
            canvas.width = w; canvas.height = h;
            canvas.getContext('2d').drawImage(video, 0, 0, w, h);
            stream.getTracks().forEach(function (t) { t.stop(); });
            return canvas.toDataURL(opts.type || 'image/png', opts.quality);
        },
        // Record video. opts: {audio, ms, blob, mimeType}. Returns { stop() -> Promise<dataURL> }.
        record: async function (opts) {
            opts = opts || {};
            var stream = await navigator.mediaDevices.getUserMedia({
                video: true,
                audio: opts.audio !== false,
            });
            active.push(stream);
            return record(stream, opts);
        },
        // List available cameras: [{deviceId, label}].
        list: async function () {
            var devices = await navigator.mediaDevices.enumerateDevices();
            return devices.filter(function (d) { return d.kind === 'videoinput'; })
                .map(function (d) { return { deviceId: d.deviceId, label: d.label }; });
        },
        // Stop every active camera stream this page started.
        stop: function () {
            active.forEach(function (s) { s.getTracks().forEach(function (t) { t.stop(); }); });
            active = [];
        },
    };
})();
