let count = 0;

const countEl = document.getElementById('count');
const timerEl = document.getElementById('timer');
const statusEl = document.getElementById('status');

function setCount(n) {
  count = n;
  countEl.textContent = n;
}

function adjust(delta) {
  const fn = delta > 0 ? 'add' : 'sub';
  BACKEND.call(fn, { value: count }, ({ data, err }) => {
    if (err) { console.error(err); return; }
    if (data && data.value !== undefined) setCount(data.value);
  });
}

BACKEND.onEvent('timer', ({ current_time }) => {
  timerEl.textContent = current_time;
});

document.addEventListener('DOMContentLoaded', () => {
  statusEl.textContent = 'connected';
  statusEl.className = 'status connected';
});
