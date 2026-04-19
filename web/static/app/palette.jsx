// Command palette (Cmd+K)

const Palette = ({ store, onClose, onOpenSession, setView, openNewSession }) => {
  const [q, setQ] = React.useState('');
  const [idx, setIdx] = React.useState(0);
  const HOSTS = store.hosts;
  const SESSIONS = store.sessions;

  const groups = React.useMemo(() => {
    const ql = q.toLowerCase();
    const sessions = SESSIONS.filter(s => !q || s.id.toLowerCase().includes(ql)).slice(0, 6).map(s => ({
      kind: 'session',
      title: s.id,
      sub: s.host,
      icon: () => <SessIcon kind={s.iconKind} color={s.iconColor} />,
      action: () => { onOpenSession(s); onClose(); },
    }));
    const hosts = HOSTS.filter(h => !q || h.fqdn.toLowerCase().includes(ql)).slice(0, 4).map(h => ({
      kind: 'host',
      title: h.fqdn,
      sub: `${h.user} · ${h.sessions} sessions`,
      icon: IconServer,
      action: () => { setView('hosts'); onClose(); },
    }));
    const actions = [
      { kind: 'action', title: 'New session', sub: '⌘N', icon: IconPlus, action: () => { openNewSession(); onClose(); } },
      { kind: 'action', title: 'Go to dashboard', icon: IconDashboard, action: () => { setView('dashboard'); onClose(); } },
      { kind: 'action', title: 'Go to sessions', icon: IconTerminal, action: () => { setView('sessions'); onClose(); } },
      { kind: 'action', title: 'Go to hosts', icon: IconServer, action: () => { setView('hosts'); onClose(); } },
      { kind: 'action', title: 'Go to settings', icon: IconSettings, action: () => { setView('settings'); onClose(); } },
      { kind: 'action', title: 'Toggle theme', sub: '⇧⌘L', icon: IconMoon, action: () => { document.documentElement.dataset.theme = document.documentElement.dataset.theme === 'light' ? 'dark' : 'light'; onClose(); } },
    ].filter(a => !q || a.title.toLowerCase().includes(ql));
    return [
      { label: 'Sessions', items: sessions },
      { label: 'Hosts', items: hosts },
      { label: 'Actions', items: actions },
    ].filter(g => g.items.length);
  }, [q, SESSIONS, HOSTS]);

  const flat = React.useMemo(() => groups.flatMap(g => g.items), [groups]);

  React.useEffect(() => { setIdx(0); }, [q]);

  React.useEffect(() => {
    const onKey = (e) => {
      if (e.key === 'Escape') onClose();
      else if (e.key === 'ArrowDown') { e.preventDefault(); setIdx(i => Math.min(flat.length - 1, i + 1)); }
      else if (e.key === 'ArrowUp') { e.preventDefault(); setIdx(i => Math.max(0, i - 1)); }
      else if (e.key === 'Enter') { e.preventDefault(); flat[idx]?.action?.(); }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [flat, idx]);

  let runningIdx = 0;
  return (
    <div className="palette-backdrop" onClick={onClose}>
      <div className="palette" onClick={(e)=>e.stopPropagation()}>
        <div className="palette-input">
          <IconSearch size={16}/>
          <input autoFocus placeholder="Search sessions, hosts, or run commands…" value={q} onChange={e=>setQ(e.target.value)} />
          <Kbd>esc</Kbd>
        </div>
        <div className="palette-results">
          {groups.map((g, gi) => (
            <div key={gi}>
              <div className="palette-group-label">{g.label}</div>
              {g.items.map((it, i) => {
                const myIdx = runningIdx++;
                const I = it.icon;
                return (
                  <div key={i} className={`palette-item ${myIdx===idx?'active':''}`} onMouseEnter={()=>setIdx(myIdx)} onClick={it.action}>
                    <span className="palette-icon">
                      {typeof I === 'function' && I.length === 0 ? <I/> : <I size={13}/>}
                    </span>
                    <span className="palette-title">{it.title}</span>
                    {it.sub && <span className="palette-sub">{it.sub}</span>}
                  </div>
                );
              })}
            </div>
          ))}
          {groups.length === 0 && (
            <div style={{padding:'24px 18px', color:'var(--fg-subtle)', fontSize:13, textAlign:'center'}}>
              No results for "{q}"
            </div>
          )}
        </div>
        <div className="palette-footer">
          <span className="hint"><Kbd>↵</Kbd> open</span>
          <span className="hint"><Kbd>↑</Kbd><Kbd>↓</Kbd> navigate</span>
          <span className="hint"><Kbd>⌘</Kbd><Kbd>K</Kbd> toggle</span>
          <span style={{marginLeft:'auto'}} className="mono">{flat.length} results</span>
        </div>
      </div>
    </div>
  );
};

Object.assign(window, { Palette });
