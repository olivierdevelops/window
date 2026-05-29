(function () {
    window.NATIVE = window.NATIVE || {};
    window.NATIVE.speech = {
        // Text to speech. opts: {voice, rate, pitch, volume, lang}. Resolves when finished speaking.
        say: function (text, opts) {
            opts = opts || {};
            return new Promise(function (resolve, reject) {
                if (!('speechSynthesis' in window)) { reject(new Error('speech synthesis unavailable')); return; }
                var u = new SpeechSynthesisUtterance(text);
                if (opts.lang) u.lang = opts.lang;
                if (opts.rate != null) u.rate = opts.rate;
                if (opts.pitch != null) u.pitch = opts.pitch;
                if (opts.volume != null) u.volume = opts.volume;
                if (opts.voice) {
                    var v = speechSynthesis.getVoices().find(function (x) { return x.name === opts.voice; });
                    if (v) u.voice = v;
                }
                u.onend = function () { resolve(); };
                u.onerror = function (e) { reject(e.error || new Error('speech error')); };
                speechSynthesis.speak(u);
            });
        },
        // Available voices: [{name, lang, default}].
        voices: function () {
            return (window.speechSynthesis ? speechSynthesis.getVoices() : [])
                .map(function (v) { return { name: v.name, lang: v.lang, default: v.default }; });
        },
        // Cancel any in-progress or queued speech.
        stop: function () { if (window.speechSynthesis) speechSynthesis.cancel(); },
        // Speech to text. callback({text, isFinal}). opts: {lang, continuous, interim}. Returns { stop() }.
        listen: function (callback, opts) {
            opts = opts || {};
            var SR = window.SpeechRecognition || window.webkitSpeechRecognition;
            if (!SR) throw new Error('speech recognition unavailable');
            var rec = new SR();
            rec.lang = opts.lang || 'en-US';
            rec.continuous = opts.continuous !== false;
            rec.interimResults = opts.interim !== false;
            rec.onresult = function (e) {
                for (var i = e.resultIndex; i < e.results.length; i++) {
                    callback({ text: e.results[i][0].transcript, isFinal: e.results[i].isFinal });
                }
            };
            rec.start();
            return { stop: function () { rec.stop(); } };
        },
    };
})();
