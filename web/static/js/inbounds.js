const dialog = document.getElementById('inbound-dialog');
const form = document.getElementById('inbound-form');
const protoSel = document.getElementById('f-protocol');

function applyProtocolVisibility() {
  const p = protoSel.value;
  form.querySelectorAll('label[data-for]').forEach(el => {
    const set = el.dataset.for.split(',');
    el.style.display = set.includes(p) ? '' : 'none';
  });
}
protoSel.addEventListener('change', applyProtocolVisibility);

document.querySelectorAll('[data-close]').forEach(el =>
  el.addEventListener('click', () => dialog.close()));

document.getElementById('new-inbound').addEventListener('click', () => {
  form.reset();
  form.id.value = '';
  document.getElementById('dialog-title').textContent = 'New inbound';
  applyProtocolVisibility();
  dialog.showModal();
});

form.addEventListener('submit', async (e) => {
  e.preventDefault();
  const fd = new FormData(form);
  const obj = Object.fromEntries(fd.entries());
  obj.port = parseInt(obj.port || '0', 10);
  obj.enabled = fd.has('enabled');
  obj.ufw = fd.has('ufw');
  const id = obj.id;
  delete obj.id;
  try {
    if (id) await api.send('PUT', '/api/inbounds/' + id, obj);
    else await api.send('POST', '/api/inbounds', obj);
    dialog.close();
    load();
  } catch (e) {
    alert('Save failed: ' + e.message);
  }
});

function pill(ok, label) {
  return `<span class="pill ${ok ? 'on' : 'off'}">${label}</span>`;
}

async function load() {
  const tbody = document.getElementById('inbound-rows');
  try {
    const { items } = await api.get('/api/inbounds');
    if (!items || !items.length) {
      tbody.innerHTML = `<tr><td colspan="8" class="muted center">No inbounds yet. Click <b>New inbound</b> to add one.</td></tr>`;
      return;
    }
    tbody.innerHTML = items.map(i => `
      <tr data-id="${i.id}">
        <td class="muted">${i.id}</td>
        <td><b>${escapeHtml(i.remark || i.tag || '')}</b><div class="muted small">${escapeHtml(i.tag)}</div></td>
        <td><span class="pill">${i.protocol}</span></td>
        <td>${i.port}</td>
        <td>${i.network}</td>
        <td>${i.security}</td>
        <td>${pill(i.enabled, i.enabled ? 'enabled' : 'disabled')}</td>
        <td><div class="row-actions">
          <button class="btn-ghost" data-edit>Edit</button>
          <button class="btn-danger" data-del>Delete</button>
        </div></td>
      </tr>`).join('');
    tbody.querySelectorAll('[data-edit]').forEach(btn =>
      btn.addEventListener('click', () => openEdit(parseInt(btn.closest('tr').dataset.id, 10))));
    tbody.querySelectorAll('[data-del]').forEach(btn =>
      btn.addEventListener('click', () => del(parseInt(btn.closest('tr').dataset.id, 10))));
    window._inbounds = items;
  } catch (e) {
    tbody.innerHTML = `<tr><td colspan="8" class="muted center">Failed: ${escapeHtml(e.message)}</td></tr>`;
  }
}

function openEdit(id) {
  const item = (window._inbounds || []).find(x => x.id === id);
  if (!item) return;
  form.reset();
  for (const [k, v] of Object.entries(item)) {
    const el = form.elements.namedItem(k);
    if (!el) continue;
    if (el.type === 'checkbox') el.checked = !!v;
    else el.value = v ?? '';
  }
  form.id.value = item.id;
  document.getElementById('dialog-title').textContent = 'Edit inbound #' + item.id;
  applyProtocolVisibility();
  dialog.showModal();
}

async function del(id) {
  if (!confirm('Delete this inbound? The port will also be removed from UFW.')) return;
  try { await api.send('DELETE', '/api/inbounds/' + id); load(); }
  catch (e) { alert('Failed: ' + e.message); }
}

function escapeHtml(s) {
  return String(s ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}

load();
