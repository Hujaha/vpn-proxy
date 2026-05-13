async function tick() {
  try {
    const s = await api.get('/api/stats');
    const sys = s.system;
    document.getElementById('stat-cpu').textContent = sys.cpu_percent.toFixed(1) + '%';
    document.getElementById('stat-cpu-sub').textContent = sys.cpu_cores + ' cores';
    document.getElementById('bar-cpu').style.width = Math.min(100, sys.cpu_percent) + '%';

    document.getElementById('stat-mem').textContent = sys.mem_percent.toFixed(1) + '%';
    document.getElementById('stat-mem-sub').textContent = `${fmt.bytes(sys.mem_used)} / ${fmt.bytes(sys.mem_total)}`;
    document.getElementById('bar-mem').style.width = Math.min(100, sys.mem_percent) + '%';

    document.getElementById('stat-disk').textContent = sys.disk_percent.toFixed(1) + '%';
    document.getElementById('stat-disk-sub').textContent = `${fmt.bytes(sys.disk_used)} / ${fmt.bytes(sys.disk_total)}`;
    document.getElementById('bar-disk').style.width = Math.min(100, sys.disk_percent) + '%';

    document.getElementById('stat-inbounds').textContent = s.inbounds;
    document.getElementById('stat-inbounds-sub').textContent = `${s.enabled} enabled`;

    document.getElementById('i-host').textContent = sys.hostname || '—';
    document.getElementById('i-platform').textContent = sys.platform || sys.os || '—';
    document.getElementById('i-uptime').textContent = fmt.duration(sys.uptime);
    document.getElementById('i-load').textContent = `${sys.load1.toFixed(2)} / ${sys.load5.toFixed(2)} / ${sys.load15.toFixed(2)}`;
    document.getElementById('i-net').textContent = `${fmt.bytes(sys.net_up)} / ${fmt.bytes(sys.net_down)}`;

    document.getElementById('i-ufw-avail').textContent = s.ufw_avail ? 'yes' : 'no';
    document.getElementById('i-ufw').textContent = s.ufw || '—';
  } catch (e) {
    console.warn(e);
  }
}

document.getElementById('restart-xray')?.addEventListener('click', async () => {
  try { await api.send('POST', '/api/xray/restart'); alert('Xray restarted'); }
  catch (e) { alert('Failed: ' + e.message); }
});

tick();
setInterval(tick, 4000);
