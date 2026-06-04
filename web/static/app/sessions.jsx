// Sessions screen

const Sessions = ({ store, openSession, openNewSession, initialFilter }) => {
  const SESSIONS = store.sessions;
  const [filter, setFilter] = React.useState(initialFilter || 'all');
  const [search, setSearch] = React.useState('');
  const [sortBy, setSortBy] = React.useState(
    () => localStorage.getItem('sshtogo.sessionSort') || 'activity'
  );
  React.useEffect(() => { localStorage.setItem('sshtogo.sessionSort', sortBy); }, [sortBy]);
  React.useEffect(() => { if (initialFilter) setFilter(initialFilter); }, [initialFilter]);
  const sortComparators = {
    activity: (a, b) => {
      const aR = Math.max(a.activityMs || 0, a.lastAccessedMs || 0);
      const bR = Math.max(b.activityMs || 0, b.lastAccessedMs || 0);
      if (bR !== aR) return bR - aR;
      return b.createdMs - a.createdMs;
    },
    opened: (a, b) => (b.lastAccessedMs || 0) - (a.lastAccessedMs || 0) || b.createdMs - a.createdMs,
    created: (a, b) => b.createdMs - a.createdMs,
    name: (a, b) => a.id.localeCompare(b.id),
  };
  const cmp = sortComparators[sortBy] || sortComparators.activity;
  const filtered = SESSIONS.filter(s => {
    if (filter === 'active' && s.activity !== 'active') return false;
    if (filter === 'attached' && s.clients.length === 0) return false;
    if (filter === 'favorites' && !s.starred) return false;
    if (search && !s.id.toLowerCase().includes(search.toLowerCase()) && !s.host.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  }).sort((a, b) => {
    // Starred always pinned on top regardless of secondary sort.
    // Offloaded sessions always sink to the bottom of the table.
    if ((a.status === 'offloaded') !== (b.status === 'offloaded')) {
      return a.status === 'offloaded' ? 1 : -1;
    }
    if (a.starred !== b.starred) return a.starred ? -1 : 1;
    return cmp(a, b);
  });
  return (
    <div>
      <div className="page-head">
        <div className="page-title-block">
          <h1>Sessions</h1>
          <p>{SESSIONS.length} tmux sessions across {new Set(SESSIONS.map(s=>s.host)).size} hosts</p>
        </div>
        <div className="page-actions">
          <Button variant="secondary" size="sm" icon={IconRefresh} onClick={() => store.refresh()}>Refresh</Button>
          <Button variant="primary" size="sm" icon={IconPlus} onClick={openNewSession}>New session</Button>
        </div>
      </div>

      <div className="filter-bar">
        <div className="seg">
          <button className={`seg-btn ${filter==='all'?'active':''}`} onClick={()=>setFilter('all')}>All <span className="count">{SESSIONS.length}</span></button>
          <button className={`seg-btn ${filter==='active'?'active':''}`} onClick={()=>setFilter('active')}>Active <span className="count">{SESSIONS.filter(s=>s.activity==='active').length}</span></button>
          <button className={`seg-btn ${filter==='attached'?'active':''}`} onClick={()=>setFilter('attached')}>Attached <span className="count">{SESSIONS.filter(s=>s.clients.length>0).length}</span></button>
          <button className={`seg-btn ${filter==='favorites'?'active':''}`} onClick={()=>setFilter('favorites')}>Favorites <span className="count">{SESSIONS.filter(s=>s.starred).length}</span></button>
        </div>
        <div className="search-sm">
          <IconSearch size={13}/>
          <input placeholder="Filter sessions…" value={search} onChange={e=>setSearch(e.target.value)} />
        </div>
        <div style={{flex:1}}></div>
        <label className="muted" style={{fontSize:12, display:'flex', alignItems:'center', gap:6}}>
          Sort:
          <select className="select" style={{padding:'2px 6px', fontSize:12}} value={sortBy} onChange={e=>setSortBy(e.target.value)}>
            <option value="activity">Recent activity</option>
            <option value="opened">Recently opened</option>
            <option value="created">Newest created</option>
            <option value="name">Name (A–Z)</option>
          </select>
        </label>
        <span className="muted" style={{fontSize:12}}>{filtered.length} shown</span>
      </div>

      <div className="panel">
        <table className="tbl">
          <thead>
            <tr>
              <th style={{width:'26%'}}>Session</th>
              <th className="hide-mobile">Host</th>
              <th className="hide-mobile">Activity</th>
              <th className="hide-mobile">Clients</th>
              <th className="hide-mobile">Window</th>
              <th className="hide-mobile">PID</th>
              <th className="hide-mobile">Uptime</th>
              <th style={{textAlign:'right'}}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map(s => <FullSessionRow key={s.id} session={s} onOpen={() => openSession(s)} />)}
          </tbody>
        </table>
      </div>
    </div>
  );
};

const FullSessionRow = ({ session: s, onOpen }) => {
  const [starred, setStarred] = React.useState(s.starred);
  const [recreating, setRecreating] = React.useState(false);
  // null | 'offloading' | 'ending'
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
          {!offloaded && <button className="rename-btn" onClick={onRename} title="Rename"><IconEdit size={12}/></button>}
          {offloaded && <Pill variant="muted">offloaded</Pill>}
        </div>
        {offloaded && s.workingDir && (
          <div className="muted mono" style={{fontSize:11, marginTop:2, paddingLeft:28}}>resume in {s.workingDir}</div>
        )}
      </td>
      <td className="muted mono hide-mobile" style={{fontSize:12.5}}>{s.host}</td>
      <td className="hide-mobile">{offloaded ? <span className="muted" style={{fontSize:12}}>—</span> : <ActivityCell session={s} />}</td>
      <td className="hide-mobile">{offloaded ? <span className="muted" style={{fontSize:12}}>—</span> : <Presence clients={s.clients} />}</td>
      <td className="mono num muted hide-mobile" style={{fontSize:12}}>{offloaded ? '—' : s.win}</td>
      <td className="mono num muted hide-mobile" style={{fontSize:12}}>{offloaded ? '—' : s.pid}</td>
      <td className="muted num hide-mobile">{s.uptime}</td>
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
          ) : (<React.Fragment>
          <button className={`action-btn star ${starred ? 'starred' : ''}`} onClick={toggleStar} disabled={!!busy}>
            <IconStar size={13} fill={starred ? 'currentColor' : 'none'} />
          </button>
          <button className="action-btn primary" onClick={onOpen} disabled={!!busy}>Open</button>
          <button className="action-btn" onClick={onHandoff} disabled={!!busy} title="Copy SSH handoff command">Handoff</button>
          <button className="action-btn" onClick={onOffload} disabled={!!busy}
                  title="Stop tmux but keep tracked so you can resume from the same directory">
            {busy === 'offloading' ? 'Offloading…' : 'Offload'}
          </button>
          <button className="action-btn danger" onClick={onKill} disabled={!!busy}>
            {busy === 'ending' ? 'Ending…' : 'End'}
          </button>
          </React.Fragment>)}
        </div>
      </td>
    </tr>
  );
};

Object.assign(window, { Sessions, FullSessionRow });
