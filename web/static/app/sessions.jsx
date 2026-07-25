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
        <SortMenu sortBy={sortBy} setSortBy={setSortBy} />
        <span className="muted" title={`${filtered.length} shown`} style={{fontSize:12, whiteSpace:'nowrap', flexShrink:0}}>{filtered.length}</span>
      </div>

      <div className="panel">
        <table className="tbl tbl-fixed">
          <thead>
            <tr>
              <th style={{width:'26%'}}>Session</th>
              <th className="col-h3">Host</th>
              <th className="col-act">Activity</th>
              <th className="col-h2">Clients</th>
              <th className="col-h1">Window</th>
              <th className="col-h1">PID</th>
              <th className="col-h4">Up</th>
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

const SORT_OPTS = [
  { value: 'activity', label: 'Recent activity' },
  { value: 'opened', label: 'Recently opened' },
  { value: 'created', label: 'Newest created' },
  { value: 'name', label: 'Name (A–Z)' },
];

// Compact sort control: the trigger button always reads "Sort" so it never
// gets truncated in a tight filter bar; the current selection is shown (with a
// check) inside the popover instead.
const SortMenu = ({ sortBy, setSortBy }) => {
  const [open, setOpen] = React.useState(false);
  const ref = React.useRef(null);
  React.useEffect(() => {
    if (!open) return;
    const onDoc = (e) => { if (ref.current && !ref.current.contains(e.target)) setOpen(false); };
    const onKey = (e) => { if (e.key === 'Escape') setOpen(false); };
    document.addEventListener('mousedown', onDoc);
    document.addEventListener('keydown', onKey);
    return () => { document.removeEventListener('mousedown', onDoc); document.removeEventListener('keydown', onKey); };
  }, [open]);
  return (
    <div className="sort-menu" ref={ref}>
      <button className={`btn btn-secondary btn-sm ${open ? 'open' : ''}`} onClick={() => setOpen(o => !o)}
              title="Sort sessions" aria-haspopup="listbox" aria-expanded={open}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <line x1="4" y1="6" x2="13" y2="6"/><line x1="4" y1="12" x2="11" y2="12"/><line x1="4" y1="18" x2="9" y2="18"/>
          <polyline points="17 8 20 5 23 8"/><line x1="20" y1="5" x2="20" y2="19"/><polyline points="17 16 20 19 23 16"/>
        </svg>
        <span>Sort</span>
        <svg className="caret" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
      </button>
      {open && (
        <div className="sort-pop" role="listbox">
          <div className="sort-pop-label">Sort by</div>
          {SORT_OPTS.map(o => (
            <button key={o.value} className={`sort-opt ${o.value === sortBy ? 'active' : ''}`} role="option"
                    aria-selected={o.value === sortBy} onClick={() => { setSortBy(o.value); setOpen(false); }}>
              <span>{o.label}</span>
              {o.value === sortBy && <IconCheck size={14} />}
            </button>
          ))}
        </div>
      )}
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
      <td className="mono num muted col-h1" style={{fontSize:12}}>{offloaded ? '—' : s.win}</td>
      <td className="mono num muted col-h1" style={{fontSize:12}}>{offloaded ? '—' : s.pid}</td>
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
          ) : (<React.Fragment>
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
          </React.Fragment>)}
        </div>
      </td>
    </tr>
  );
};

Object.assign(window, { Sessions, FullSessionRow });
