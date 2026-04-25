const STORE = {
  hosts: [],
  sessions: [],
  keypairs: [],
  settings: {},
  icons: {},
  pubkey: null,
  version: '',
  loading: true,
  error: null,
  _listeners: new Set(),
  _lastRefresh: 0
};
function notify() {
  STORE._listeners.forEach(fn => {
    try {
      fn();
    } catch (e) {}
  });
}
async function authFetch(url, opts) {
  const res = await fetch(url, opts);
  if (res.status === 401) {
    window.location.href = '/login';
    throw new Error('unauthorized');
  }
  return res;
}
async function refresh() {
  try {
    const [h, s, k, st, ic] = await Promise.all([authFetch('/api/hosts').then(r => r.json()), authFetch('/api/sessions').then(r => r.json()), authFetch('/api/keypairs').then(r => r.json()), authFetch('/api/settings').then(r => r.json()), authFetch('/api/session-icons').then(r => r.json()).catch(() => ({}))]);
    STORE.hosts = h || [];
    STORE.sessions = s || [];
    STORE.keypairs = k || [];
    STORE.settings = st || {};
    STORE.icons = ic || {};
    STORE.loading = false;
    STORE.error = null;
    STORE._lastRefresh = Date.now();
    notify();
  } catch (e) {
    STORE.error = e.message || 'fetch failed';
    STORE.loading = false;
    notify();
  }
}
function iconFor(hostName, sessionName) {
  const key = `${hostName}:${sessionName}`;
  const icon = STORE.icons[key];
  if (icon) return icon;
  const host = STORE.hosts.find(h => (h.config && h.config.name) === hostName);
  if (host && host.config) return {
    icon: host.config.icon || '',
    color: host.config.icon_color || ''
  };
  return {
    icon: '',
    color: ''
  };
}
function adaptHosts() {
  return STORE.hosts.map(h => {
    const cfg = h.config || {};
    const fqdnBase = cfg.address || cfg.name || '';
    const port = cfg.port && cfg.port !== 22 ? `:${cfg.port}` : '';
    const os = h.detected_os || cfg.os || '';
    const osKind = detectOsKind(os);
    const sessionCount = (h.sessions || []).length;
    const metrics = h.metrics || null;
    return {
      id: cfg.name,
      fqdn: fqdnBase + port,
      user: cfg.user || '',
      status: h.online ? 'online' : 'offline',
      os: os || 'unknown',
      osKind,
      sessions: sessionCount,
      lastSync: h.last_poll ? timeAgo(new Date(h.last_poll)) : '—',
      load: h.load_history && h.load_history.length ? h.load_history : Array(20).fill(0),
      cpu: metrics ? metrics.cpu : null,
      mem: metrics ? metrics.mem : null,
      disk: metrics ? metrics.disk : null,
      load1: metrics ? metrics.load1 : null,
      tags: cfg.tags || [],
      _raw: h
    };
  });
}
function adaptSessions() {
  const now = Date.now();
  return STORE.sessions.map(entry => {
    const s = entry.session || {};
    const hostName = entry.host_name;
    const host = STORE.hosts.find(x => (x.config && x.config.name) === hostName);
    const hostAddress = host ? host.config.address || host.config.name : hostName;
    const createdMs = s.created ? new Date(s.created).getTime() : now;
    const uptimeSec = Math.max(0, Math.floor((now - createdMs) / 1000));
    const icon = iconFor(hostName, s.name);
    const activity = s.attached ? 'active' : 'idle';
    const clients = [];
    for (let i = 0; i < (s.attached_clients || 0); i++) {
      clients.push({
        name: '··',
        color: 'indigo'
      });
    }
    const lastAccessedMs = icon.last_accessed ? new Date(icon.last_accessed).getTime() : 0;
    return {
      id: s.name,
      host: hostAddress,
      hostName,
      uptime: formatUptime(uptimeSec),
      activity,
      lastInput: s.attached ? 'attached' : timeAgo(new Date(createdMs)),
      idle: activity === 'active' ? 0 : uptimeSec,
      clients,
      pid: null,
      win: s.windows || 1,
      iconKind: icon.icon || 'terminal',
      iconColor: icon.color || 'indigo',
      starred: !!icon.starred,
      createdMs,
      lastAccessedMs,
      _raw: entry
    };
  });
}
function adaptKeypairs() {
  const defaultName = STORE.settings.default_keypair;
  return STORE.keypairs.map(k => ({
    name: k.name,
    type: k.type,
    imported: k.source === 'imported',
    fp: k.fingerprint,
    isDefault: k.name === defaultName,
    createdAt: k.created_at,
    _raw: k
  }));
}
function activeSessionCount() {
  return STORE.sessions.filter(s => s.session && s.session.attached).length;
}
function timeAgo(d) {
  const diff = Math.max(0, Date.now() - d.getTime());
  const s = Math.floor(diff / 1000);
  if (s < 10) return 'just now';
  if (s < 60) return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 48) return `${h}h ago`;
  const dd = Math.floor(h / 24);
  return `${dd}d ago`;
}
function formatUptime(s) {
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m`;
  const h = Math.floor(m / 60);
  if (h < 48) return `${h}h`;
  const dd = Math.floor(h / 24);
  return `${dd}d`;
}
function detectOsKind(os) {
  const lower = (os || '').toLowerCase();
  if (lower.includes('ubuntu')) return 'ubuntu';
  if (lower.includes('debian')) return 'debian';
  if (lower.includes('alpine')) return 'alpine';
  return 'ubuntu';
}
function useStore() {
  const [, setTick] = React.useState(0);
  React.useEffect(() => {
    const fn = () => setTick(t => t + 1);
    STORE._listeners.add(fn);
    return () => STORE._listeners.delete(fn);
  }, []);
  return {
    hosts: adaptHosts(),
    sessions: adaptSessions(),
    keypairs: adaptKeypairs(),
    settings: STORE.settings,
    icons: STORE.icons,
    loading: STORE.loading,
    error: STORE.error,
    activeSessionCount: activeSessionCount(),
    refresh,
    raw: STORE
  };
}
async function createSession(hostName, name, cwd) {
  const r = await authFetch(`/api/hosts/${encodeURIComponent(hostName)}/sessions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      name,
      cwd: cwd || ''
    })
  });
  if (!r.ok) throw new Error(await r.text());
  await refresh();
  return r.json();
}
async function killSession(hostName, name) {
  const r = await authFetch(`/api/hosts/${encodeURIComponent(hostName)}/sessions/${encodeURIComponent(name)}`, {
    method: 'DELETE'
  });
  if (!r.ok) throw new Error(await r.text());
  await refresh();
}
async function renameSession(hostName, oldName, newName) {
  const r = await authFetch(`/api/hosts/${encodeURIComponent(hostName)}/sessions/${encodeURIComponent(oldName)}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      new_name: newName
    })
  });
  if (!r.ok) throw new Error(await r.text());
  await refresh();
}
async function getHandoff(hostName, sessionName) {
  const r = await authFetch(`/api/hosts/${encodeURIComponent(hostName)}/sessions/${encodeURIComponent(sessionName)}/handoff`);
  if (!r.ok) throw new Error(await r.text());
  const j = await r.json();
  return j.command;
}
async function setSessionIconPatch(hostName, sessionName, patch) {
  const current = STORE.icons[`${hostName}:${sessionName}`] || {};
  const next = {
    ...current,
    ...patch
  };
  const r = await authFetch(`/api/session-icons/${encodeURIComponent(hostName)}/${encodeURIComponent(sessionName)}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(next)
  });
  if (!r.ok) throw new Error(await r.text());
  STORE.icons[`${hostName}:${sessionName}`] = next;
  notify();
}
async function addHost(payload) {
  const r = await authFetch('/api/hosts', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  });
  if (!r.ok) throw new Error(await r.text());
  await refresh();
}
async function scanAll() {
  const r = await authFetch('/api/scan', {
    method: 'POST'
  });
  if (!r.ok) throw new Error(await r.text());
  await refresh();
}
function openTerminal(hostName, sessionName) {
  window.open(`/terminal/${encodeURIComponent(hostName)}/${encodeURIComponent(sessionName)}`, '_blank');
}
refresh();
setInterval(refresh, 10000);
authFetch('/api/version').then(r => r.json()).then(v => {
  STORE.version = v.version || '';
  notify();
}).catch(() => {});
Object.assign(window, {
  STORE,
  useStore,
  refresh,
  createSession,
  killSession,
  renameSession,
  getHandoff,
  setSessionIconPatch,
  addHost,
  scanAll,
  openTerminal,
  timeAgo,
  formatUptime
});