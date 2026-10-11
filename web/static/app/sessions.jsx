// Sessions screen

const Sessions = ({ store, openSession, openNewSession, initialFilter }) => {
  const SESSIONS = store.sessions;
  const [filter, setFilter] = React.useState(initialFilter || 'all');
  const [search, setSearch] = React.useState('');
  const [sortBy, setSortBy] = React.useState(
    () => localStorage.getItem('sshtogo.sessionSort') || 'activity'
  );
  React.useEffect(() => { localStorage.setItem('sshtogo.sessionSort', sortBy); }, [sortBy]);
  // '' = every session, UNTAGGED = only sessions without tags, else one tag.
  const [tagFilter, setTagFilter] = React.useState(
    () => localStorage.getItem('sshtogo.sessionTag') || ''
  );
  React.useEffect(() => { localStorage.setItem('sshtogo.sessionTag', tagFilter); }, [tagFilter]);
  const tags = tagCounts(SESSIONS);
  // A remembered tag that no session carries any more would hide everything.
  const tagGone = tagFilter && tagFilter !== UNTAGGED && !tags.some(t => t.tag === tagFilter);
  React.useEffect(() => { if (tagGone && !store.loading) setTagFilter(''); }, [tagGone, store.loading]);
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
    if (filter === 'attached' && !s.clients) return false;
    if (filter === 'favorites' && !s.starred) return false;
    if (tagFilter === UNTAGGED && s.tags.length) return false;
    if (tagFilter && tagFilter !== UNTAGGED && !tagGone && !s.tags.includes(tagFilter)) return false;
    if (search && !s.id.toLowerCase().includes(search.toLowerCase()) && !s.host.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  }).sort((a, b) => {
    // Sorting by tag groups rows under their first tag (untagged last);
    // everything below then orders rows inside each group.
    if (sortBy === 'tag') {
      const g = groupOrder(a, b);
      if (g) return g;
    }
    // Starred always pinned on top regardless of secondary sort.
    // Offloaded sessions always sink to the bottom of the table.
    if ((a.status === 'offloaded') !== (b.status === 'offloaded')) {
      return a.status === 'offloaded' ? 1 : -1;
    }
    if (a.starred !== b.starred) return a.starred ? -1 : 1;
    return (sortBy === 'tag' ? sortComparators.activity : cmp)(a, b);
  });
  const rows = [];
  let group = null;
  filtered.forEach(s => {
    if (sortBy === 'tag' && primaryTag(s) !== group) {
      group = primaryTag(s);
      rows.push(<TagGroupRow key={'group:' + group} tag={group}
                             count={filtered.filter(x => primaryTag(x) === group).length} />);
    }
    rows.push(<FullSessionRow key={s.hostName + ':' + s.id} session={s} onOpen={() => openSession(s)}
                              allTags={tags} onTagClick={setTagFilter} />);
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
          <button className={`seg-btn ${filter==='attached'?'active':''}`} onClick={()=>setFilter('attached')}>Attached <span className="count">{SESSIONS.filter(s=>s.clients>0).length}</span></button>
          <button className={`seg-btn ${filter==='favorites'?'active':''}`} onClick={()=>setFilter('favorites')}>Favorites <span className="count">{SESSIONS.filter(s=>s.starred).length}</span></button>
        </div>
        <div className="search-sm">
          <IconSearch size={13}/>
          <input placeholder="Filter sessions…" value={search} onChange={e=>setSearch(e.target.value)} />
        </div>
        <div style={{flex:1}}></div>
        <TagFilterMenu tags={tags} tagFilter={tagGone ? '' : tagFilter} setTagFilter={setTagFilter}
                       untaggedCount={SESSIONS.filter(s => !s.tags.length).length} />
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
              <th className="col-h4">Up</th>
              <th style={{textAlign:'right'}}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {rows}
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
  { value: 'tag', label: 'Tag (grouped)' },
];

// Sentinel for the "Untagged" filter; can't collide with a real tag because
// normalised tags never contain spaces or start with a colon.
const UNTAGGED = ':untagged';

// The group a session sits in when sorting by tag: its first tag
// alphabetically, or '' for untagged.
const primaryTag = (s) => (s.tags && s.tags[0]) || '';

const groupOrder = (a, b) => {
  const ga = primaryTag(a), gb = primaryTag(b);
  if (ga === gb) return 0;
  if (!ga) return 1;
  if (!gb) return -1;
  return ga.localeCompare(gb);
};

// Swatches for tag colours, as OKLCH hues. Mirrors keystore.TagHues, which
// the server draws a new tag's random colour from.
const TAG_HUES = [25, 55, 75, 95, 125, 150, 180, 210, 245, 275, 305, 345];

// A tag's colour is stored server-side (STORE.tagColors). Tags from before
// colours were stored fall back to a hue hashed from the name, so they stay
// stable until someone picks one.
const tagHue = (tag) => {
  const stored = STORE.tagColors && STORE.tagColors[tag];
  if (typeof stored === 'number') return stored;
  let h = 0;
  for (let i = 0; i < tag.length; i++) h = (h * 31 + tag.charCodeAt(i)) >>> 0;
  return h % 360;
};

const TagChip = ({ tag, onClick, title }) => (
  <span className={`tag-chip ${onClick ? 'clickable' : ''}`} style={{'--tag-h': tagHue(tag)}}
        onClick={onClick} title={title}>{tag}</span>
);

// One cell per column (not a colSpan): the narrow-screen container queries
// hide columns by class, and a spanning cell across hidden columns throws
// off the fixed table layout.
const TagGroupRow = ({ tag, count }) => (
  <tr className="tag-group-row">
    <td>
      {tag ? <TagChip tag={tag} /> : <span className="muted">Untagged</span>}
      <span className="count">{count}</span>
    </td>
    <td className="col-h3"></td><td className="col-act"></td><td className="col-h2"></td>
    <td className="col-h4"></td><td></td>
  </tr>
);

// Closes a popover on an outside click or Escape. `refs` are the elements
// that count as inside (the trigger and the popover itself).
const useDismiss = (open, setOpen, refs) => {
  React.useEffect(() => {
    if (!open) return;
    const onDoc = (e) => { if (!refs.some(r => r.current && r.current.contains(e.target))) setOpen(false); };
    const onKey = (e) => { if (e.key === 'Escape') setOpen(false); };
    document.addEventListener('mousedown', onDoc);
    document.addEventListener('keydown', onKey);
    return () => { document.removeEventListener('mousedown', onDoc); document.removeEventListener('keydown', onKey); };
  }, [open]);
};

// Filter-bar dropdown: show every session, one tag, or the untagged ones.
// The trigger names the active tag so a filtered table is never a mystery.
const TagFilterMenu = ({ tags, tagFilter, setTagFilter, untaggedCount }) => {
  const [open, setOpen] = React.useState(false);
  const ref = React.useRef(null);
  useDismiss(open, setOpen, [ref]);
  const pick = (v) => { setTagFilter(v); setOpen(false); };
  const label = tagFilter === UNTAGGED ? 'Untagged' : (tagFilter || 'Tags');
  return (
    <div className="sort-menu" ref={ref}>
      <button className={`btn btn-secondary btn-sm ${open ? 'open' : ''} ${tagFilter ? 'tag-filter-on' : ''}`}
              onClick={() => setOpen(o => !o)} title="Filter by tag" aria-haspopup="listbox" aria-expanded={open}>
        <IconTag size={14} />
        <span className="tag-filter-label">{label}</span>
        <svg className="caret" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
      </button>
      {open && (
        <div className="sort-pop" role="listbox">
          <div className="sort-pop-label">Show</div>
          <button className={`sort-opt ${!tagFilter ? 'active' : ''}`} role="option" aria-selected={!tagFilter} onClick={() => pick('')}>
            <span>All tags</span>{!tagFilter && <IconCheck size={14} />}
          </button>
          {tags.map(t => (
            <button key={t.tag} className={`sort-opt ${tagFilter === t.tag ? 'active' : ''}`} role="option"
                    aria-selected={tagFilter === t.tag} onClick={() => pick(t.tag)}>
              <span className="tag-opt"><TagChip tag={t.tag} /><span className="count">{t.count}</span></span>
              {tagFilter === t.tag && <IconCheck size={14} />}
            </button>
          ))}
          <button className={`sort-opt ${tagFilter === UNTAGGED ? 'active' : ''}`} role="option"
                  aria-selected={tagFilter === UNTAGGED} onClick={() => pick(UNTAGGED)}>
            <span className="tag-opt"><span>Untagged</span><span className="count">{untaggedCount}</span></span>
            {tagFilter === UNTAGGED && <IconCheck size={14} />}
          </button>
          {!tags.length && <div className="tag-pop-hint">Tag a session with the <IconTag size={11} /> button next to its name.</div>}
        </div>
      )}
    </div>
  );
};

// Popover for adding/removing a session's tags. Rendered into <body> with
// fixed positioning: the table cells clip overflow, so an in-cell popover
// would be cut off.
const TagEditor = ({ anchor, session: s, allTags, onClose }) => {
  const [draft, setDraft] = React.useState('');
  // The tag whose colour swatches are expanded, if any.
  const [coloring, setColoring] = React.useState(null);
  const [current, setCurrent] = React.useState(s.tags);
  const [err, setErr] = React.useState('');
  const ref = React.useRef(null);
  const anchorRef = React.useRef(anchor);
  useDismiss(true, onClose, [ref, anchorRef]);
  const r = anchor.getBoundingClientRect();
  const width = 240;
  const left = Math.max(8, Math.min(r.left, window.innerWidth - width - 8));
  // Open below the button when it fits, else above it; failing both, pin it
  // to the bottom edge. Re-run every render: expanding a tag's swatches
  // changes the height. Set on the node directly so it never re-renders.
  React.useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const h = el.offsetHeight;
    const below = r.bottom + 6;
    const above = r.top - 6 - h;
    el.style.top = (below + h <= window.innerHeight - 8 ? below
      : above >= 8 ? above
      : Math.max(8, window.innerHeight - h - 8)) + 'px';
  });
  const save = async (next) => {
    setCurrent(next);
    setErr('');
    try { await setSessionTags(s.hostName, s.id, next); }
    catch (e) { setErr('save failed: ' + e.message); }
  };
  const toggle = (tag) => save(current.includes(tag) ? current.filter(t => t !== tag) : [...current, tag].sort());
  const add = (e) => {
    e.preventDefault();
    const t = normalizeTag(draft);
    setDraft('');
    if (t && !current.includes(t)) save([...current, t].sort());
  };
  const suggestions = [...new Set([...allTags.map(t => t.tag), ...current])].sort();
  return ReactDOM.createPortal(
    <div className="tag-editor" ref={ref} style={{ top: r.bottom + 6, left, width }}>
      <div className="sort-pop-label">Tags for {s.id}</div>
      <form onSubmit={add} className="tag-editor-add">
        <input autoFocus placeholder="New tag, then Enter" value={draft} maxLength={32}
               onChange={e => setDraft(e.target.value)} />
      </form>
      {suggestions.map(tag => (
        <React.Fragment key={tag}>
          <div className="tag-editor-row">
            <button className={`tag-swatch-btn ${coloring === tag ? 'open' : ''}`} style={{'--tag-h': tagHue(tag)}}
                    title={`Change the colour of "${tag}"`} aria-label={`Change colour of ${tag}`}
                    onClick={() => setColoring(coloring === tag ? null : tag)} />
            <button className={`sort-opt ${current.includes(tag) ? 'active' : ''}`} onClick={() => toggle(tag)}>
              <TagChip tag={tag} />
              {current.includes(tag) && <IconCheck size={14} />}
            </button>
          </div>
          {coloring === tag && (
            <div className="tag-swatches">
              {TAG_HUES.map(h => (
                <button key={h} className={`tag-swatch ${tagHue(tag) === h ? 'selected' : ''}`} style={{'--tag-h': h}}
                        aria-label={`Hue ${h}`}
                        onClick={() => setTagColor(tag, h).catch(e => setErr('colour save failed: ' + e.message))} />
              ))}
            </div>
          )}
        </React.Fragment>
      ))}
      {!suggestions.length && <div className="tag-pop-hint">Try "work" or "personal".</div>}
      {err && <div className="tag-pop-hint" style={{color:'var(--err)'}}>{err}</div>}
    </div>,
    document.body
  );
};

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

// Row bits shared by the Sessions table and the dashboard's Recent sessions.
// Both rows show the same session in the same states, so the "why is this
// offloaded" and "don't sleep this one" affordances live in one place.

// Auto-sleep is off by default, and a keep-awake toggle means nothing until
// it's on — so the button only appears once the setting exists.
const autoSleepOn = () => (STORE.settings && STORE.settings.idle_offload_hours > 0);

const KeepAwakeButton = ({ session: s, disabled }) => {
  const [keep, setKeep] = React.useState(!!s.keepAwake);
  if (!autoSleepOn()) return null;
  const toggle = (e) => {
    e.stopPropagation();
    const next = !keep;
    setKeep(next);
    setSessionIconPatch(s.hostName, s.id, { keep_awake: next });
  };
  return (
    <button className={`action-btn icon ${keep ? 'starred' : ''}`} onClick={toggle} disabled={disabled}
            data-tip={keep ? 'Keep awake: exempt from idle auto-offload' : 'Let this session auto-offload when idle'}
            aria-label={keep ? 'Allow auto-offload' : 'Keep awake'}>
      <IconSun size={14} />
    </button>
  );
};

// Why a session is in the offloaded state, and what comes back with it.
const OffloadedNote = ({ session: s }) => {
  if (!s.workingDir && !s.command) return null;
  return (
    <div className="muted mono" style={{fontSize:11, marginTop:2, paddingLeft:28}}>
      {s.workingDir ? `resume in ${s.workingDir}` : 'resume'}
      {s.command ? ` · runs ${s.command}` : ''}
    </div>
  );
};

const FullSessionRow = ({ session: s, onOpen, allTags, onTagClick }) => {
  const [starred, setStarred] = React.useState(s.starred);
  // The tag button element while the tag editor is open, else null.
  const [tagAnchor, setTagAnchor] = React.useState(null);
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
          {offloaded && <Pill variant="muted">{s.autoOffloaded ? 'slept' : 'offloaded'}</Pill>}
          {s.tags.map(t => <TagChip key={t} tag={t} title={`Show only "${t}" sessions`}
                                    onClick={onTagClick ? (e) => { e.stopPropagation(); onTagClick(t); } : undefined} />)}
          <button className={`rename-btn tag-btn ${s.tags.length ? '' : 'untagged'}`} data-tip="Edit tags" aria-label="Edit tags"
                  onClick={(e) => { e.stopPropagation(); setTagAnchor(tagAnchor ? null : e.currentTarget); }}>
            <IconTag size={12}/>
          </button>
          {tagAnchor && <TagEditor anchor={tagAnchor} session={s} allTags={allTags || []} onClose={() => setTagAnchor(null)} />}
        </div>
        {offloaded && <OffloadedNote session={s} />}
      </td>
      <td className="muted mono col-h3" style={{fontSize:12.5}}>{s.host}</td>
      <td className="col-act">{offloaded ? <span className="muted" style={{fontSize:12}}>—</span> : <ActivityCell session={s} />}</td>
      <td className="col-h2">{offloaded ? <span className="muted" style={{fontSize:12}}>—</span> : <ClientCount count={s.clients} />}</td>
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
          <KeepAwakeButton session={s} disabled={!!busy} />
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

Object.assign(window, { Sessions, FullSessionRow, KeepAwakeButton, OffloadedNote, TagChip });
