document.getElementById('pw-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const msg = document.getElementById('pw-msg');
  msg.textContent = '';
  try {
    await api.send('POST', '/api/account/password', {
      old: fd.get('old'),
      new: fd.get('new'),
    });
    msg.style.color = 'var(--success)';
    msg.textContent = 'Password updated';
    e.target.reset();
  } catch (err) {
    msg.style.color = 'var(--danger)';
    msg.textContent = err.message;
  }
});

(async () => {
  try {
    const { host } = await api.get('/api/settings/host');
    document.querySelector('#host-form [name=host]').value = host || '';
  } catch {}
})();

document.getElementById('host-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const fd = new FormData(e.target);
  const msg = document.getElementById('host-msg');
  msg.textContent = '';
  try {
    await api.send('POST', '/api/settings/host', { host: fd.get('host') });
    msg.style.color = 'var(--success)';
    msg.textContent = 'Saved';
  } catch (err) {
    msg.style.color = 'var(--danger)';
    msg.textContent = err.message;
  }
});

document.getElementById('restart-xray').addEventListener('click', async () => {
  try { await api.send('POST', '/api/xray/restart'); alert('Xray restarted'); }
  catch (e) { alert('Failed: ' + e.message); }
});

(async () => {
  try {
    const cfg = await api.get('/api/xray/config');
    document.getElementById('xray-config').textContent = JSON.stringify(cfg, null, 2);
  } catch (e) {
    document.getElementById('xray-config').textContent = 'Failed: ' + e.message;
  }
})();
