// Dashboard

const Dashboard = ({ store, setView, openSession, openNewSession }) => {
  const HOSTS = store.hosts;
  const SESSIONS = store.sessions;
  const KEYPAIRS = store.keypairs;
  // Offloaded sessions are merged into store.sessions with status:'offloaded'
  // and rendered at the bottom of the Recent sessions table.
  const activeCount = SESSIONS.filter(s => s.activity === 'active').length;
  const attached = SESSIONS.filter(s => s.clients.length > 0).length;
  const totalHostLoad = HOSTS.length
    ? HOSTS.reduce((sum, h) => sum.map((v, i) => v + (h.load[i] || 0)), Array(20).fill(0)).map(v => v / HOSTS.length)
    : Array(20).fill(0);

  return (
    <div>
      <div className="page-head">
        <div className="page-title-block">
          <h1>Dashboard</h1>
          <p>{HOSTS.length} hosts online · {activeCount} sessions active · {attached} with clients attached</p>
        </div>
        <div className="page-actions">
          <Button variant="secondary" size="sm" icon={IconRefresh} onClick={() => store.refresh()}>Refresh</Button>
          <Button variant="primary" size="sm" icon={IconPlus} onClick={openNewSession}>New session</Button>
        </div>
      </div>

      {/* Stat cards */}
      <div className="stats-grid">
        <StatCard
          label="Hosts"
          value={HOSTS.length}
          sub={<><span className="dot ok" style={{width:6,height:6,boxShadow:'none', display:'inline-block', marginRight:5}}></span>All online</>}
          delta={null}
          spark={<Sparkline data={totalHostLoad} width={70} height={20} />}
        />
        <StatCard
          label="Active sessions"
          value={SESSIONS.length}
          sub={<span>{activeCount} active · {SESSIONS.length - activeCount} idle</span>}
          delta={{dir:'up', val:'+3'}}
          spark={<Sparkline data={[8,9,10,9,10,11,12,12,11,12]} width={70} height={20} />}
        />
        <StatCard
          label="Attached clients"
          value={<>{SESSIONS.reduce((n,s)=>n+s.clients.length,0)}<span style={{fontSize:14, color:'var(--fg-subtle)', fontWeight:500, marginLeft:6}}>across {attached}</span></>}
          sub={<Presence clients={[{name:'AC',color:'indigo'},{name:'MB',color:'teal'},{name:'JP',color:'amber'},{name:'SR',color:'violet'}]} max={4} />}
          delta={null}
        />
        <StatCard
          label="SSH keypairs"
          value={KEYPAIRS.length}
          sub={(() => {
            const def = KEYPAIRS.find(k => k.isDefault) || KEYPAIRS[0];
            return <span className="mono" style={{fontSize:11.5}}>{def ? `default: ${def.name} · ${def.type}` : 'no keypairs yet'}</span>;
          })()}
          delta={null}
          icon={<IconKey size={13} />}
        />
      </div>

      <div className="grid-2">
        {/* Recent sessions */}
        <div className="panel">
          <div className="panel-head">
            <div className="row gap-3">
              <h2>Recent sessions</h2>
              <div className="seg">
                <span className="seg-btn active">All <span className="count">{SESSIONS.length}</span></span>
                <span className="seg-btn">Active <span className="count">{activeCount}</span></span>
                <span className="seg-btn">Attached <span className="count">{attached}</span></span>
              </div>
            </div>
            <button className="btn btn-ghost btn-sm" onClick={() => setView('sessions')}>
              View all <IconArrowRight size={12} />
            </button>
          </div>
          <div className="tbl-scroll" style={{maxHeight: 640, overflowY:'auto'}}>
            <table className="tbl tbl-fixed">
              <thead>
                <tr>
                  <th style={{width: '30%'}}>Session</th>
                  <th className="col-h3">Host</th>
                  <th className="col-act">Activity</th>
                  <th className="col-h2">Clients</th>
                  <th className="col-h4">Up</th>
                  <th style={{textAlign:'right'}}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {SESSIONS.slice().sort((a, b) => {
                  // Starred pinned to top, then most-recently-used. "Recently used"
                  // = max(tmux session_activity, our own click-to-open timestamp),
                  // so opening a session via this UI and bash typing on the remote
                  // both bubble the session to the top.
                  // Offloaded sessions always sink to the bottom of the table.
                  if ((a.status === 'offloaded') !== (b.status === 'offloaded')) {
                    return a.status === 'offloaded' ? 1 : -1;
                  }
                  if (a.starred !== b.starred) return a.starred ? -1 : 1;
                  const aRecent = Math.max(a.activityMs || 0, a.lastAccessedMs || 0);
                  const bRecent = Math.max(b.activityMs || 0, b.lastAccessedMs || 0);
                  if (bRecent !== aRecent) return bRecent - aRecent;
                  return b.createdMs - a.createdMs;
                }).slice(0, 20).map(s => (
                  <SessionRow key={`${s.hostName}:${s.id}`} session={s} onOpen={() => openSession(s)} />
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Right column: host load + activity */}
        <div style={{display:'flex', flexDirection:'column', gap:16}}>
          <div className="panel">
            <div className="panel-head">
              <h2>Host load</h2>
              <span className="muted" style={{fontSize:11.5, fontFamily:'var(--font-mono)'}}>cpu · mem</span>
            </div>
            <div className="host-mini-grid">
              {HOSTS.map(h => (
                <div key={h.id} className="host-mini">
                  <div className="host-mini-head">
                    <StatusDot status={h.status === 'online' ? 'active' : 'offline'} />
                    <span className="host-mini-name truncate">{h.fqdn}</span>
                  </div>
                  <div className="host-bar-row">
                    <HostBar label="CPU" value={h.cpu} />
                    <HostBar label="MEM" value={h.mem} />
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="panel">
            <div className="panel-head">
              <h2>Activity</h2>
              <span className="muted" style={{fontSize:11.5}}>coming soon</span>
            </div>
            <div style={{padding:'28px 16px', color:'var(--fg-subtle)', fontSize:12.5, textAlign:'center'}}>
              Live activity feed lands with backend event log (issue #20).
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

// Vertical bar for CPU/MEM. value is 0-100 or null. When null we render an
// empty track + dash so the layout stays put until the metrics endpoint lands.
const HostBar = ({ label, value }) => {
  const has = value != null && !isNaN(value);
  const pct = has ? Math.max(0, Math.min(100, Number(value))) : 0;
  const color = !has ? 'transparent'
    : pct < 60 ? 'var(--ok)'
    : pct < 85 ? 'var(--warn)'
    : 'var(--err)';
  return (
    <div className="host-bar">
      <div className="host-bar-track">
        <div className="host-bar-fill" style={{ height: pct + '%', background: color }} />
      </div>
      <div className="host-bar-label">{label}</div>
      <div className="host-bar-value mono" style={{ color: has ? 'var(--fg)' : 'var(--fg-faint)' }}>
        {has ? `${Math.round(pct)}%` : '—'}
      </div>
    </div>
  );
};

const StatCard = ({ label, value, sub, delta, spark, icon }) => (
  <div className="stat">
    <div className="stat-label">
      {icon}{label}
    </div>
    <div className="stat-value">
      <span>{value}</span>
      {delta && <span className={`stat-delta ${delta.dir}`}>{delta.val}</span>}
    </div>
    <div className="stat-sub">{sub}</div>
    {spark && <div className="stat-spark">{spark}</div>}
  </div>
);

const MissingSessionsPanel = ({ missing }) => {
  const [busy, setBusy] = React.useState({});
  const action = async (fn, m, label) => {
    const key = `${m.hostName}:${m.name}`;
    setBusy(b => ({ ...b, [key]: label }));
    try { await fn(m.hostName, m.name); }
    catch (err) { alert(`${label} failed: ${err.message}`); }
    finally { setBusy(b => { const n = { ...b }; delete n[key]; return n; }); }
  };
  return (
    <div className="panel" style={{marginBottom:16, borderColor:'var(--warn, #c89b3c)'}}>
      <div className="panel-head">
        <div className="row gap-3">
          <h2 style={{color:'var(--warn, #c89b3c)'}}>Resumable sessions</h2>
          <span className="muted" style={{fontSize:12}}>tracked but not running — either you offloaded them or the host rebooted</span>
        </div>
        <span className="muted mono" style={{fontSize:12}}>{missing.length}</span>
      </div>
      <table className="tbl">
        <thead>
          <tr>
            <th style={{width:'24%'}}>Session</th>
            <th>Host</th>
            <th className="hide-mobile">Last working dir</th>
            <th style={{textAlign:'right'}}>Actions</th>
          </tr>
        </thead>
        <tbody>
          {missing.map(m => {
            const key = `${m.hostName}:${m.name}`;
            const b = busy[key];
            return (
              <tr key={key}>
                <td><span className="mono">{m.name}</span></td>
                <td className="muted mono" style={{fontSize:12.5}}>{m.host}</td>
                <td className="mono muted hide-mobile" style={{fontSize:12}}>{m.workingDir || '—'}</td>
                <td>
                  <div className="actions-cell">
                    <button className="action-btn primary" disabled={!!b} onClick={() => action(recreateSession, m, 'recreate')}>
                      {b === 'recreate' ? '…' : 'Recreate'}
                    </button>
                    <button className="action-btn" disabled={!!b} onClick={() => action(forgetSession, m, 'forget')}>
                      {b === 'forget' ? '…' : 'Forget'}
                    </button>
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

const SessionRow = ({ session: s, onOpen }) => {
  const [starred, setStarred] = React.useState(s.starred);
  const [recreating, setRecreating] = React.useState(false);
  // null | 'offloading' | 'ending' — shared so both destructive actions
  // disable each other while one is in flight.
  const [busy, setBusy] = React.useState(null);
  const toggleStar = (e) => {
    e.stopPropagation();
    const next = !starred;
    setStarred(next);
    setSessionIconPatch(s.hostName, s.id, { starred: next });
  };
  const onHandoff = async (e) => {
    e.stopPropagation();
    try { const cmd = await getHandoff(s.hostName, s.id); await navigator.clipboard.writeText(cmd); } catch(err) { alert('handoff failed: ' + err.message); }
  };
  const onKill = async (e) => {
    e.stopPropagation();
    if (busy) return;
    if (!confirm(`End session "${s.id}"? This forgets it entirely.`)) return;
    setBusy('ending');
    try { await killSession(s.hostName, s.id); }
    catch(err) { alert('end failed: ' + err.message); }
    finally { setBusy(null); }
  };
  const onOffload = async (e) => {
    e.stopPropagation();
    if (busy) return;
    if (!confirm(`Offload session "${s.id}"? The tmux session is killed but ssh-to-go keeps the working directory so you can resume it from the table below.`)) return;
    setBusy('offloading');
    try { await offloadSession(s.hostName, s.id); }
    catch(err) { alert('offload failed: ' + err.message); }
    finally { setBusy(null); }
  };
  const onPickIcon = (e) => {
    e.stopPropagation();
    if (!window.showIconPicker) return;
    window.showIconPicker(e.currentTarget, s.iconKind || 'terminal', (iconName, colorName) => {
      setSessionIconPatch(s.hostName, s.id, { icon: iconName, color: colorName });
    }, s.iconColor || 'default');
  };
  const onRename = async (e) => {
    e.stopPropagation();
    const next = prompt(`Rename session "${s.id}" to:`, s.id);
    if (!next || next === s.id) return;
    try { await renameSession(s.hostName, s.id, next); } catch(err) { alert('rename failed: ' + err.message); }
  };
  const onRecreate = async (e) => {
    e.stopPropagation();
    if (recreating) return;
    setRecreating(true);
    try {
      await recreateSession(s.hostName, s.id);
      // Jump straight into the freshly-spawned tmux session in a new tab.
      openTerminal(s.hostName, s.id);
    } catch(err) {
      alert('recreate failed: ' + err.message);
    } finally {
      setRecreating(false);
    }
  };
  const onForget = async (e) => {
    e.stopPropagation();
    if (!confirm(`Forget session "${s.id}"? ssh-to-go drops the saved working directory; it can't be resumed afterwards.`)) return;
    try { await forgetSession(s.hostName, s.id); } catch(err) { alert('forget failed: ' + err.message); }
  };
  const offloaded = s.status === 'offloaded';
  return (
    <tr style={offloaded ? {opacity: 0.65} : null}>
      <td>
        <div className="cell-session">
          <button className="sess-icon-btn" onClick={onPickIcon} title="Change icon">
            <SessIcon kind={s.iconKind} color={s.iconColor} />
          </button>
          <span className="mono name" onClick={offloaded ? onRecreate : onOpen} style={{cursor:'pointer'}}>{s.id}</span>
          {!offloaded && <button className="rename-btn" onClick={onRename} data-tip="Rename this session" aria-label="Rename session"><IconEdit size={12}/></button>}
          {offloaded && <Pill variant="muted">offloaded</Pill>}
        </div>
        {offloaded && s.workingDir && (
          <div className="muted mono" style={{fontSize:11, marginTop:2, paddingLeft:28}}>resume in {s.workingDir}</div>
        )}
      </td>
      <td className="muted mono col-h3" style={{fontSize:12.5}}>{s.host}</td>
      <td className="col-act">{offloaded ? <span className="muted" style={{fontSize:12}}>—</span> : <ActivityCell session={s} />}</td>
      <td className="col-h2">{offloaded ? <span className="muted" style={{fontSize:12}}>—</span> : <Presence clients={s.clients} />}</td>
      <td className="muted num col-h4">{s.uptime}</td>
      <td>
        <div className="actions-cell">
          {offloaded ? (
            <React.Fragment>
              <button className="action-btn primary" onClick={onRecreate} disabled={recreating}
                      title="Bring the tmux session back at its saved working directory and open it in a new tab">
                {recreating ? 'Recreating…' : 'Recreate'}
              </button>
              <button className="action-btn" onClick={onForget} disabled={recreating} title="Forget the saved working directory">Forget</button>
            </React.Fragment>
          ) : (
            <React.Fragment>
              <button className={`action-btn icon star ${starred ? 'starred' : ''}`} onClick={toggleStar} disabled={!!busy} data-tip={starred ? 'Remove from favorites' : 'Add to favorites'} aria-label={starred ? 'Remove from favorites' : 'Add to favorites'}>
                <IconStar size={14} fill={starred ? 'currentColor' : 'none'} />
              </button>
              <button className="action-btn icon" onClick={onHandoff} disabled={!!busy} data-tip="Copy the SSH command to attach from your own terminal" aria-label="Copy SSH command">
                <IconCopy size={14} />
              </button>
              <button className={`action-btn icon ${busy === 'offloading' ? 'busy' : ''}`} onClick={onOffload} disabled={!!busy}
                      data-tip="Offload: stop it now, resume later in the same directory" aria-label="Offload session">
                <IconMoon size={14} />
              </button>
              <button className={`action-btn icon danger ${busy === 'ending' ? 'busy' : ''}`} onClick={onKill} disabled={!!busy} data-tip="Kill the session and stop tracking it" aria-label="Kill session">
                <IconClose size={14} />
              </button>
            </React.Fragment>
          )}
        </div>
      </td>
    </tr>
  );
};

Object.assign(window, { Dashboard, StatCard, SessionRow, HostBar, MissingSessionsPanel });
