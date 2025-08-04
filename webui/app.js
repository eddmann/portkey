const token =
  localStorage.getItem('portkeyToken') || prompt('Auth token (admin):');
localStorage.setItem('portkeyToken', token);

const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
const ws = new WebSocket(`${protocol}//${location.host}/api/ws?token=${token}`);
const tbody = document.querySelector('#log-table tbody');
const filterInput = document.getElementById('filter');
const tunnelSpan = document.getElementById('tunnel-list');
const loadMoreBtn = document.getElementById('load-more');
const toggleModeBtn = document.getElementById('toggle-mode');

let filterText = '';
let showCount = 100; // pagination size

filterInput.addEventListener('input', () => {
  filterText = filterInput.value.toLowerCase();
  applyFilter();
});

toggleModeBtn.addEventListener('click', () => {
  document.body.classList.toggle('dark');
});

loadMoreBtn.addEventListener('click', () => {
  showCount += 100;
  applyFilter();
});

function applyFilter() {
  let shown = 0;
  [...tbody.querySelectorAll('tr.main')].forEach(row => {
    const path = row.dataset.path.toLowerCase();
    const match = path.includes(filterText);
    if (match && shown < showCount) {
      row.style.display = '';
      shown++;
    } else {
      row.style.display = 'none';
      if (row.nextSibling && row.nextSibling.classList.contains('details'))
        row.nextSibling.style.display = 'none';
    }
  });
}

function addRow(e) {
  const tr = document.createElement('tr');
  tr.className = 'main';
  tr.dataset.path = e.path;
  tr.innerHTML =
    `<td>${new Date(e.timestamp).toLocaleTimeString()}</td>` +
    `<td>${e.subdomain}</td>` +
    `<td>${e.method}</td>` +
    `<td>${e.path}</td>` +
    `<td>${e.status}</td>`;
  const arrowTd = document.createElement('td');
  arrowTd.className = 'arrow';
  arrowTd.textContent = '▶';
  tr.appendChild(arrowTd);
  tr.addEventListener('click', () => toggleDetails(tr, e));
  tbody.prepend(tr);
  applyFilter();
}

function toggleDetails(row, entry) {
  if (row.nextSibling && row.nextSibling.classList.contains('details')) {
    row.nextSibling.remove();
    row.querySelector('.arrow').textContent = '▶';
    return;
  }
  const detail = document.createElement('tr');
  detail.className = 'details';
  const cell = document.createElement('td');
  cell.colSpan = 6;
  row.querySelector('.arrow').textContent = '▼';
  let bodyVal = entry.body;
  try {
    bodyVal = JSON.parse(entry.body);
  } catch {}
  const pre = document.createElement('pre');
  pre.textContent = JSON.stringify(
    { headers: entry.headers, body: bodyVal },
    null,
    2
  );
  cell.appendChild(pre);
  detail.appendChild(cell);
  row.after(detail);
}

// initial load (fetch latest logs)
fetch(`/api/requests?token=${token}`)
  .then(r => r.json())
  .then(arr => {
    arr.slice(-showCount).forEach(addRow);
  });

// live updates
ws.onmessage = evt => addRow(JSON.parse(evt.data));
ws.onerror = () => console.error('WebSocket error');

// tunnel list poll
setInterval(() => {
  fetch(`/api/tunnels?token=${token}`)
    .then(r => r.json())
    .then(arr => {
      tunnelSpan.textContent = `Tunnels: ${arr.join(', ')}`;
    });
}, 5000);
