// token prompt retained
const token = localStorage.getItem('portkeyToken') || prompt('Auth token (admin):');
localStorage.setItem('portkeyToken', token);

const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
const ws = new WebSocket(`${protocol}//${location.host}/api/ws?token=${token}`);

const tbody = document.querySelector('#log-table tbody');
const filterInput = document.getElementById('filter');

let filterText = '';
filterInput.addEventListener('input', () => {
  filterText = filterInput.value.toLowerCase();
  [...tbody.rows].forEach(row => {
    row.style.display = row.dataset.path.toLowerCase().includes(filterText) ? '' : 'none';
  });
});

function addRow(e) {
  const tr = document.createElement('tr');
  tr.dataset.path = e.path;
  tr.innerHTML = `<td>${new Date(e.timestamp).toLocaleTimeString()}</td>` +
                 `<td>${e.subdomain}</td>` +
                 `<td>${e.method}</td>` +
                 `<td>${e.path}</td>` +
                 `<td>${e.status}</td>`;
  tbody.prepend(tr);
  if (filterText && !e.path.toLowerCase().includes(filterText)) tr.style.display = 'none';
  if (tbody.rows.length > 1000) tbody.deleteRow(-1);
}

fetch(`/api/requests?token=${token}`)
  .then(r => r.json())
  .then(arr => arr.forEach(addRow));

ws.onmessage = evt => addRow(JSON.parse(evt.data));
ws.onerror = () => console.error('WebSocket error');
