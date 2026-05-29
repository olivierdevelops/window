(function () {

    const levels = ["log", "warn", "error", "info"];
    levels.forEach(level => {
        const orig = console[level];
        console[level] = function (...args) {
            logFromJS(level, args.map(JSON.stringify).join(" "));
            orig.apply(console, args);
        };
    });

    window.onerror = function (msg, src, line, col) {
        logFromJS("error", msg + " @" + src + ":" + line + ":" + col);
    };

    window.onunhandledrejection = function (e) {
        logFromJS("error", "UnhandledPromise: " + e.reason);
    };
})();

class Backend {
    constructor() {
    }

    /**
    * @param {string} name
    * @param {Map<string, any>} params
    * @returns {Promise<[Map<string, any>, error]>}
    */
    call(name, params, onReply) {
        return new Promise(async (resolve, reject) => {
         try {
            const {eventId, err, result} = await __CALL_BACKEND(name, params || {});
            if (err) {
                onReply(null, err);
                resolve();
                return
            }

            // In-process Go handler: result is returned synchronously, no
            // streaming event. Resolve immediately (avoids the listener race).
            if (typeof result !== "undefined") {
                onReply({data: result});
                resolve();
                return
            }

            // Subprocess/socket backend: replies stream in as events keyed by eventId.
            window.addEventListener(eventId, ({detail})=>{
                const {name, code, eventId, data, function: functionName, done} = detail;
                if (code == "eval") {
                    eval(code)
                }

                try {

                    onReply({data, err})
                    
                } catch (error) {
                    console.log("[unexpected error]", `${error}`)
                }

                if (done){

                    resolve()

                }
            })


        } catch (error) {
            console.log("[unexpected error]", `${error}`)
            onReply(null, error);
        }
       })
    }

    onEvent(eventId, onReply) {
        window.addEventListener(eventId, (e)=>{
            const {detail} = e;
            onReply(detail)
        })
    }
}

const BACKEND = new Backend()
