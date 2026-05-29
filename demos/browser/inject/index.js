// Remove any existing custom UI
const existing = document.getElementById('custom-browser-ui-host');
if (existing) existing.remove();

// Create host element
const host = document.createElement('div');
host.id = 'custom-browser-ui-host';
host.style.cssText = 'position: fixed; top: 0; left: 0; right: 0; z-index: 2147483647; pointer-events: none;';

// Attach CLOSED shadow DOM (completely isolated)
const shadow = host.attachShadow({ mode: 'closed' });

// Inject your UI
shadow.innerHTML = "% s";

// Make UI interactive
shadow.querySelectorAll('*').forEach(el => {
    el.style.pointerEvents = 'auto';
});

// // Set up communication with Go
// const setupHandlers = shadow.querySelector('script[data-setup]');
// if (setupHandlers) {
//     eval(setupHandlers.textContent);
// }

document.documentElement.appendChild(host);