const token = localStorage.getItem('portkeyToken') || prompt('Auth token (admin):');
localStorage.setItem('portkeyToken', token);

const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
const ws = new WebSocket(`${protocol}//${location.host}/api/ws?token=${token}`);

const tbody = document.querySelector('#log-table tbody');

ws.onmessage = evt => {
  const e = JSON.parse(evt.data);
  const tr = document.createElement('tr');
  tr.innerHTML = `<td>${new Date(e.timestamp).toLocaleTimeString()}</td>` +
                 `<td>${e.subdomain}</td>` +
                 `<td>${e.method}</td>` +
                 `<td>${e.path}</td>` +
                 `<td>${e.status}</td>`;
  tbody.prepend(tr);
  const rows = tbody.querySelectorAll('tr');
  if (rows.length > 1000) rows[rows.length-1].remove();
};

ws.onerror = () => alert('WebSocket error - check console');
