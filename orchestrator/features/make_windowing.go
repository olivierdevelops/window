package orchfeatures

import (
	"webview_gui/domain"
	"webview_gui/features"
	"webview_gui/infra"
)

const trustScript = `(function() {

    // ── Clipboard ────────────────────────────────────────────────────────
    const _clip = {
        readText: () => new Promise((resolve, reject) => {
            const ta = document.createElement('textarea')
            ta.style.cssText = 'position:fixed;top:-9999px;opacity:0'
            document.body.appendChild(ta)
            ta.focus()
            const ok = document.execCommand('paste')
            const text = ta.value
            document.body.removeChild(ta)
            ok ? resolve(text) : reject(new Error('paste unavailable'))
        }),
        writeText: (text) => new Promise((resolve, reject) => {
            const ta = document.createElement('textarea')
            ta.style.cssText = 'position:fixed;top:-9999px;opacity:0'
            ta.value = text
            document.body.appendChild(ta)
            ta.select()
            const ok = document.execCommand('copy')
            document.body.removeChild(ta)
            ok ? resolve() : reject(new Error('copy unavailable'))
        }),
        read: () => Promise.reject(new Error('not supported')),
        write: () => Promise.reject(new Error('not supported')),
    }

    try {
        Object.defineProperty(navigator, 'clipboard', {
            value: _clip, writable: false, configurable: true,
        })
    } catch(_) {}

    if (navigator.permissions) {
        const _query = navigator.permissions.query.bind(navigator.permissions)
        navigator.permissions.query = (desc) => {
            if (desc?.name === 'clipboard-read' || desc?.name === 'clipboard-write') {
                return Promise.resolve({ state: 'granted', onchange: null })
            }
            return _query(desc)
        }
    }

    document.addEventListener('keydown', (e) => {
        if (!(e.ctrlKey || e.metaKey) || e.key !== 'v') return
        const el = document.activeElement
        if (!el) return
        _clip.readText().then(text => {
            if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
                const s = el.selectionStart ?? el.value.length
                const end = el.selectionEnd ?? el.value.length
                el.value = el.value.slice(0, s) + text + el.value.slice(end)
                el.selectionStart = el.selectionEnd = s + text.length
                el.dispatchEvent(new Event('input', { bubbles: true }))
                el.dispatchEvent(new Event('change', { bubbles: true }))
            } else if (el.isContentEditable) {
                const sel = window.getSelection()
                if (sel?.rangeCount) {
                    sel.deleteFromDocument()
                    sel.getRangeAt(0).insertNode(document.createTextNode(text))
                    sel.collapseToEnd()
                    el.dispatchEvent(new InputEvent('input', { bubbles: true }))
                }
            }
        }).catch(() => {})
    }, true)

    document.addEventListener('keydown', (e) => {
        if (!(e.ctrlKey || e.metaKey)) return
        if (e.key !== 'c' && e.key !== 'x') return
        const sel = window.getSelection()?.toString()
        if (!sel) return
        _clip.writeText(sel).catch(() => {})
        if (e.key === 'x') document.execCommand('delete')
    }, true)

    try {
        Object.defineProperty(window, 'isSecureContext', {
            value: true, writable: false, configurable: true,
        })
    } catch(_) {}

    if (!crypto.randomUUID) {
        crypto.randomUUID = () => ([1e7]+-1e3+-4e3+-8e3+-1e11)
            .replace(/[018]/g, c =>
                (c ^ crypto.getRandomValues(new Uint8Array(1))[0] & 15 >> c / 4).toString(16)
            )
    }

})()`

// MakeWindowing builds a Windowing feature struct backed by the real webview.
func MakeWindowing() features.Windowing {
	return features.Windowing{
		New: func(title string, size *domain.WindowSize, debug bool) (features.WindowHandle, error) {
			w, h := 800, 800
			if size != nil {
				w, h = size.Width, size.Height
			}
			handle := infra.NewWebviewHandle(title, w, h, debug)
			// Inject trust script on every page load.
			handle.WV.Init(trustScript)
			return handle, nil
		},
		Destroy: func(h features.WindowHandle) {
			infra.DestroyWebview(h.(*infra.WebviewHandle))
		},
		Navigate: func(h features.WindowHandle, url string) {
			infra.NavigateWebview(h.(*infra.WebviewHandle), url)
		},
		Eval: func(h features.WindowHandle, js string) {
			infra.EvalWebview(h.(*infra.WebviewHandle), js)
		},
		Bind: func(h features.WindowHandle, name string, fn any) error {
			return infra.BindWebview(h.(*infra.WebviewHandle), name, fn)
		},
		Init: func(h features.WindowHandle, js string) {
			infra.InitWebview(h.(*infra.WebviewHandle), js)
		},
		Run: func(h features.WindowHandle) {
			infra.RunWebview(h.(*infra.WebviewHandle))
		},
		SetTitle: func(h features.WindowHandle, title string) {
			infra.SetTitleWebview(h.(*infra.WebviewHandle), title)
		},
		SendEvent: func(h features.WindowHandle, eventID string, data any) {
			infra.SendEventWebview(h.(*infra.WebviewHandle), eventID, data)
		},
	}
}
