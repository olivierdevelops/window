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

    window.NATIVE.mic = {
        // Record audio. opts: {ms, blob, mimeType}. Returns { stop() -> Promise<dataURL> }.
        record: async function (opts) {
            var stream = await navigator.mediaDevices.getUserMedia({ audio: true });
            active.push(stream);
            return record(stream, opts);
        },
        // Raw microphone stream.
        stream: async function () {
            var stream = await navigator.mediaDevices.getUserMedia({ audio: true });
            active.push(stream);
            return stream;
        },
        // Live input level metering. callback(level) where level is 0..1. Returns { stop() }.
        meter: async function (callback) {
            var stream = await navigator.mediaDevices.getUserMedia({ audio: true });
            active.push(stream);
            var ctx = new (window.AudioContext || window.webkitAudioContext)();
            var analyser = ctx.createAnalyser();
            analyser.fftSize = 512;
            ctx.createMediaStreamSource(stream).connect(analyser);
            var data = new Uint8Array(analyser.frequencyBinCount);
            var running = true;
            (function loop() {
                if (!running) return;
                analyser.getByteTimeDomainData(data);
                var sum = 0;
                for (var i = 0; i < data.length; i++) {
                    var v = (data[i] - 128) / 128;
                    sum += v * v;
                }
                callback(Math.sqrt(sum / data.length));
                requestAnimationFrame(loop);
            })();
            return { stop: function () {
                running = false;
                ctx.close();
                stream.getTracks().forEach(function (t) { t.stop(); });
            } };
        },
        // List available microphones: [{deviceId, label}].
        list: async function () {
            var devices = await navigator.mediaDevices.enumerateDevices();
            return devices.filter(function (d) { return d.kind === 'audioinput'; })
                .map(function (d) { return { deviceId: d.deviceId, label: d.label }; });
        },
        stop: function () {
            active.forEach(function (s) { s.getTracks().forEach(function (t) { t.stop(); }); });
            active = [];
        },
    };
})();
