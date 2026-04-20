// Dashboard

const Dashboard = ({ store, setView, openSession, openNewSession }) => {
  const HOSTS = store.hosts;
  const SESSIONS = store.sessions;
  const KEYPAIRS = store.keypairs;
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
          <div style={{maxHeight: 440, overflowY:'auto'}}>
            <table className="tbl">
              <thead>
                <tr>
                  <th style={{width: '30%'}}>Session</th>
                  <th>Host</th>
                  <th>Activity</th>
                  <th>Clients</th>
                  <th>Uptime</th>
                  <th style={{textAlign:'right'}}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {SESSIONS.slice(0, 8).map(s => (
                  <SessionRow key={s.id} session={s} onOpen={() => openSession(s)} />
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
              <span className="muted" style={{fontSize:11.5, fontFamily:'var(--font-mono)'}}>last 60s</span>
            </div>
            <div className="host-mini-grid">
              {HOSTS.map(h => (
                <div key={h.id} className="host-mini">
                  <div className="host-mini-head">
                    <StatusDot status={h.status === 'online' ? 'active' : 'offline'} />
                    <span className="host-mini-name truncate">{h.fqdn}</span>
                  </div>
                  <Sparkline data={h.load} width={120} height={28} />
                  <div className="host-mini-meta mt-2">
                    <span>CPU {h.cpu}%</span>
                    <span>MEM {h.mem}%</span>
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

const SessionRow = ({ session: s, onOpen }) => {
  const [starred, setStarred] = React.useState(s.starred);
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
    if (!confirm(`End session "${s.id}"?`)) return;
    try { await killSession(s.hostName, s.id); } catch(err) { alert('end failed: ' + err.message); }
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
  return (
    <tr>
      <td>
        <div className="cell-session">
          <button className="sess-icon-btn" onClick={onPickIcon} title="Change icon">
            <SessIcon kind={s.iconKind} color={s.iconColor} />
          </button>
          <span className="mono name" onClick={onOpen} style={{cursor:'pointer'}}>{s.id}</span>
          <button className="rename-btn" onClick={onRename} title="Rename"><IconEdit size={12}/></button>
        </div>
      </td>
      <td className="muted mono" style={{fontSize:12.5}}>{s.host}</td>
      <td><ActivityCell session={s} /></td>
      <td><Presence clients={s.clients} /></td>
      <td className="muted num">{s.uptime}</td>
      <td>
        <div className="actions-cell">
          <button className={`action-btn star ${starred ? 'starred' : ''}`} onClick={toggleStar}>
            <IconStar size={13} fill={starred ? 'currentColor' : 'none'} />
          </button>
          <button className="action-btn primary" onClick={onOpen}>Open</button>
          <button className="action-btn" onClick={onHandoff} title="Copy SSH handoff command">Handoff</button>
          <button className="action-btn danger" onClick={onKill}>End</button>
        </div>
      </td>
    </tr>
  );
};

Object.assign(window, { Dashboard, StatCard, SessionRow });
