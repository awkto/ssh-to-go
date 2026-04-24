// Sessions screen

const Sessions = ({ store, openSession, openNewSession, initialFilter }) => {
  const SESSIONS = store.sessions;
  const [filter, setFilter] = React.useState(initialFilter || 'all');
  const [search, setSearch] = React.useState('');
  React.useEffect(() => { if (initialFilter) setFilter(initialFilter); }, [initialFilter]);
  const filtered = SESSIONS.filter(s => {
    if (filter === 'active' && s.activity !== 'active') return false;
    if (filter === 'attached' && s.clients.length === 0) return false;
    if (filter === 'favorites' && !s.starred) return false;
    if (search && !s.id.toLowerCase().includes(search.toLowerCase()) && !s.host.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
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
      <td className="muted mono hide-mobile" style={{fontSize:12.5}}>{s.host}</td>
      <td className="hide-mobile"><ActivityCell session={s} /></td>
      <td className="hide-mobile"><Presence clients={s.clients} /></td>
      <td className="mono num muted hide-mobile" style={{fontSize:12}}>{s.win}</td>
      <td className="mono num muted hide-mobile" style={{fontSize:12}}>{s.pid}</td>
      <td className="muted num hide-mobile">{s.uptime}</td>
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

Object.assign(window, { Sessions, FullSessionRow });
