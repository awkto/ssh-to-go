const Sessions = ({
  store,
  openSession,
  openNewSession,
  initialFilter
}) => {
  const SESSIONS = store.sessions;
  const [filter, setFilter] = React.useState(initialFilter || 'all');
  const [search, setSearch] = React.useState('');
  const [sortBy, setSortBy] = React.useState(() => localStorage.getItem('sshtogo.sessionSort') || 'activity');
  React.useEffect(() => {
    localStorage.setItem('sshtogo.sessionSort', sortBy);
  }, [sortBy]);
  const [tagFilter, setTagFilter] = React.useState(() => localStorage.getItem('sshtogo.sessionTag') || '');
  React.useEffect(() => {
    localStorage.setItem('sshtogo.sessionTag', tagFilter);
  }, [tagFilter]);
  const tags = tagCounts(SESSIONS);
  const tagGone = tagFilter && tagFilter !== UNTAGGED && !tags.some(t => t.tag === tagFilter);
  React.useEffect(() => {
    if (tagGone && !store.loading) setTagFilter('');
  }, [tagGone, store.loading]);
  React.useEffect(() => {
    if (initialFilter) setFilter(initialFilter);
  }, [initialFilter]);
  const sortComparators = {
    activity: (a, b) => {
      const aR = Math.max(a.activityMs || 0, a.lastAccessedMs || 0);
      const bR = Math.max(b.activityMs || 0, b.lastAccessedMs || 0);
      if (bR !== aR) return bR - aR;
      return b.createdMs - a.createdMs;
    },
    opened: (a, b) => (b.lastAccessedMs || 0) - (a.lastAccessedMs || 0) || b.createdMs - a.createdMs,
    created: (a, b) => b.createdMs - a.createdMs,
    name: (a, b) => a.id.localeCompare(b.id)
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
    if (sortBy === 'tag') {
      const g = groupOrder(a, b);
      if (g) return g;
    }
    if (a.status === 'offloaded' !== (b.status === 'offloaded')) {
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
      rows.push(React.createElement(TagGroupRow, {
        key: 'group:' + group,
        tag: group,
        count: filtered.filter(x => primaryTag(x) === group).length
      }));
    }
    rows.push(React.createElement(FullSessionRow, {
      key: s.hostName + ':' + s.id,
      session: s,
      onOpen: () => openSession(s),
      allTags: tags,
      onTagClick: setTagFilter
    }));
  });
  return React.createElement("div", null, React.createElement("div", {
    className: "page-head"
  }, React.createElement("div", {
    className: "page-title-block"
  }, React.createElement("h1", null, "Sessions"), React.createElement("p", null, SESSIONS.length, " tmux sessions across ", new Set(SESSIONS.map(s => s.host)).size, " hosts")), React.createElement("div", {
    className: "page-actions"
  }, React.createElement(Button, {
    variant: "secondary",
    size: "sm",
    icon: IconRefresh,
    onClick: () => store.refresh()
  }, "Refresh"), React.createElement(Button, {
    variant: "primary",
    size: "sm",
    icon: IconPlus,
    onClick: openNewSession
  }, "New session"))), React.createElement("div", {
    className: "filter-bar"
  }, React.createElement("div", {
    className: "seg"
  }, React.createElement("button", {
    className: `seg-btn ${filter === 'all' ? 'active' : ''}`,
    onClick: () => setFilter('all')
  }, "All ", React.createElement("span", {
    className: "count"
  }, SESSIONS.length)), React.createElement("button", {
    className: `seg-btn ${filter === 'active' ? 'active' : ''}`,
    onClick: () => setFilter('active')
  }, "Active ", React.createElement("span", {
    className: "count"
  }, SESSIONS.filter(s => s.activity === 'active').length)), React.createElement("button", {
    className: `seg-btn ${filter === 'attached' ? 'active' : ''}`,
    onClick: () => setFilter('attached')
  }, "Attached ", React.createElement("span", {
    className: "count"
  }, SESSIONS.filter(s => s.clients > 0).length)), React.createElement("button", {
    className: `seg-btn ${filter === 'favorites' ? 'active' : ''}`,
    onClick: () => setFilter('favorites')
  }, "Favorites ", React.createElement("span", {
    className: "count"
  }, SESSIONS.filter(s => s.starred).length))), React.createElement("div", {
    className: "search-sm"
  }, React.createElement(IconSearch, {
    size: 13
  }), React.createElement("input", {
    placeholder: "Filter sessions\u2026",
    value: search,
    onChange: e => setSearch(e.target.value)
  })), React.createElement("div", {
    style: {
      flex: 1
    }
  }), React.createElement(TagFilterMenu, {
    tags: tags,
    tagFilter: tagGone ? '' : tagFilter,
    setTagFilter: setTagFilter,
    untaggedCount: SESSIONS.filter(s => !s.tags.length).length
  }), React.createElement(SortMenu, {
    sortBy: sortBy,
    setSortBy: setSortBy
  }), React.createElement("span", {
    className: "muted",
    title: `${filtered.length} shown`,
    style: {
      fontSize: 12,
      whiteSpace: 'nowrap',
      flexShrink: 0
    }
  }, filtered.length)), React.createElement("div", {
    className: "panel"
  }, React.createElement("table", {
    className: "tbl tbl-fixed"
  }, React.createElement("thead", null, React.createElement("tr", null, React.createElement("th", {
    style: {
      width: '26%'
    }
  }, "Session"), React.createElement("th", {
    className: "col-h3"
  }, "Host"), React.createElement("th", {
    className: "col-act"
  }, "Activity"), React.createElement("th", {
    className: "col-h2"
  }, "Clients"), React.createElement("th", {
    className: "col-h4"
  }, "Up"), React.createElement("th", {
    style: {
      textAlign: 'right'
    }
  }, "Actions"))), React.createElement("tbody", null, rows))));
};
const SORT_OPTS = [{
  value: 'activity',
  label: 'Recent activity'
}, {
  value: 'opened',
  label: 'Recently opened'
}, {
  value: 'created',
  label: 'Newest created'
}, {
  value: 'name',
  label: 'Name (A–Z)'
}, {
  value: 'tag',
  label: 'Tag (grouped)'
}];
const UNTAGGED = ':untagged';
const primaryTag = s => s.tags && s.tags[0] || '';
const groupOrder = (a, b) => {
  const ga = primaryTag(a),
    gb = primaryTag(b);
  if (ga === gb) return 0;
  if (!ga) return 1;
  if (!gb) return -1;
  return ga.localeCompare(gb);
};
const TAG_HUES = [25, 55, 75, 95, 125, 150, 180, 210, 245, 275, 305, 345];
const tagHue = tag => {
  const stored = STORE.tagColors && STORE.tagColors[tag];
  if (typeof stored === 'number') return stored;
  let h = 0;
  for (let i = 0; i < tag.length; i++) h = h * 31 + tag.charCodeAt(i) >>> 0;
  return h % 360;
};
const TagChip = ({
  tag,
  onClick,
  title
}) => React.createElement("span", {
  className: `tag-chip ${onClick ? 'clickable' : ''}`,
  style: {
    '--tag-h': tagHue(tag)
  },
  onClick: onClick,
  title: title
}, tag);
const TagGroupRow = ({
  tag,
  count
}) => React.createElement("tr", {
  className: "tag-group-row"
}, React.createElement("td", null, tag ? React.createElement(TagChip, {
  tag: tag
}) : React.createElement("span", {
  className: "muted"
}, "Untagged"), React.createElement("span", {
  className: "count"
}, count)), React.createElement("td", {
  className: "col-h3"
}), React.createElement("td", {
  className: "col-act"
}), React.createElement("td", {
  className: "col-h2"
}), React.createElement("td", {
  className: "col-h4"
}), React.createElement("td", null));
const useDismiss = (open, setOpen, refs) => {
  React.useEffect(() => {
    if (!open) return;
    const onDoc = e => {
      if (!refs.some(r => r.current && r.current.contains(e.target))) setOpen(false);
    };
    const onKey = e => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('mousedown', onDoc);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDoc);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);
};
const TagFilterMenu = ({
  tags,
  tagFilter,
  setTagFilter,
  untaggedCount
}) => {
  const [open, setOpen] = React.useState(false);
  const ref = React.useRef(null);
  useDismiss(open, setOpen, [ref]);
  const pick = v => {
    setTagFilter(v);
    setOpen(false);
  };
  const label = tagFilter === UNTAGGED ? 'Untagged' : tagFilter || 'Tags';
  return React.createElement("div", {
    className: "sort-menu",
    ref: ref
  }, React.createElement("button", {
    className: `btn btn-secondary btn-sm ${open ? 'open' : ''} ${tagFilter ? 'tag-filter-on' : ''}`,
    onClick: () => setOpen(o => !o),
    title: "Filter by tag",
    "aria-haspopup": "listbox",
    "aria-expanded": open
  }, React.createElement(IconTag, {
    size: 14
  }), React.createElement("span", {
    className: "tag-filter-label"
  }, label), React.createElement("svg", {
    className: "caret",
    width: "12",
    height: "12",
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: "2",
    strokeLinecap: "round",
    strokeLinejoin: "round"
  }, React.createElement("polyline", {
    points: "6 9 12 15 18 9"
  }))), open && React.createElement("div", {
    className: "sort-pop",
    role: "listbox"
  }, React.createElement("div", {
    className: "sort-pop-label"
  }, "Show"), React.createElement("button", {
    className: `sort-opt ${!tagFilter ? 'active' : ''}`,
    role: "option",
    "aria-selected": !tagFilter,
    onClick: () => pick('')
  }, React.createElement("span", null, "All tags"), !tagFilter && React.createElement(IconCheck, {
    size: 14
  })), tags.map(t => React.createElement("button", {
    key: t.tag,
    className: `sort-opt ${tagFilter === t.tag ? 'active' : ''}`,
    role: "option",
    "aria-selected": tagFilter === t.tag,
    onClick: () => pick(t.tag)
  }, React.createElement("span", {
    className: "tag-opt"
  }, React.createElement(TagChip, {
    tag: t.tag
  }), React.createElement("span", {
    className: "count"
  }, t.count)), tagFilter === t.tag && React.createElement(IconCheck, {
    size: 14
  }))), React.createElement("button", {
    className: `sort-opt ${tagFilter === UNTAGGED ? 'active' : ''}`,
    role: "option",
    "aria-selected": tagFilter === UNTAGGED,
    onClick: () => pick(UNTAGGED)
  }, React.createElement("span", {
    className: "tag-opt"
  }, React.createElement("span", null, "Untagged"), React.createElement("span", {
    className: "count"
  }, untaggedCount)), tagFilter === UNTAGGED && React.createElement(IconCheck, {
    size: 14
  })), !tags.length && React.createElement("div", {
    className: "tag-pop-hint"
  }, "Tag a session with the ", React.createElement(IconTag, {
    size: 11
  }), " button next to its name.")));
};
const TagEditor = ({
  anchor,
  session: s,
  allTags,
  onClose
}) => {
  const [draft, setDraft] = React.useState('');
  const [coloring, setColoring] = React.useState(null);
  const [current, setCurrent] = React.useState(s.tags);
  const [err, setErr] = React.useState('');
  const ref = React.useRef(null);
  const anchorRef = React.useRef(anchor);
  useDismiss(true, onClose, [ref, anchorRef]);
  const r = anchor.getBoundingClientRect();
  const width = 240;
  const left = Math.max(8, Math.min(r.left, window.innerWidth - width - 8));
  const save = async next => {
    setCurrent(next);
    setErr('');
    try {
      await setSessionTags(s.hostName, s.id, next);
    } catch (e) {
      setErr('save failed: ' + e.message);
    }
  };
  const toggle = tag => save(current.includes(tag) ? current.filter(t => t !== tag) : [...current, tag].sort());
  const add = e => {
    e.preventDefault();
    const t = normalizeTag(draft);
    setDraft('');
    if (t && !current.includes(t)) save([...current, t].sort());
  };
  const suggestions = [...new Set([...allTags.map(t => t.tag), ...current])].sort();
  return ReactDOM.createPortal(React.createElement("div", {
    className: "tag-editor",
    ref: ref,
    style: {
      top: r.bottom + 6,
      left,
      width
    }
  }, React.createElement("div", {
    className: "sort-pop-label"
  }, "Tags for ", s.id), React.createElement("form", {
    onSubmit: add,
    className: "tag-editor-add"
  }, React.createElement("input", {
    autoFocus: true,
    placeholder: "New tag, then Enter",
    value: draft,
    maxLength: 32,
    onChange: e => setDraft(e.target.value)
  })), suggestions.map(tag => React.createElement(React.Fragment, {
    key: tag
  }, React.createElement("div", {
    className: "tag-editor-row"
  }, React.createElement("button", {
    className: `tag-swatch-btn ${coloring === tag ? 'open' : ''}`,
    style: {
      '--tag-h': tagHue(tag)
    },
    title: `Change the colour of "${tag}"`,
    "aria-label": `Change colour of ${tag}`,
    onClick: () => setColoring(coloring === tag ? null : tag)
  }), React.createElement("button", {
    className: `sort-opt ${current.includes(tag) ? 'active' : ''}`,
    onClick: () => toggle(tag)
  }, React.createElement(TagChip, {
    tag: tag
  }), current.includes(tag) && React.createElement(IconCheck, {
    size: 14
  }))), coloring === tag && React.createElement("div", {
    className: "tag-swatches"
  }, TAG_HUES.map(h => React.createElement("button", {
    key: h,
    className: `tag-swatch ${tagHue(tag) === h ? 'selected' : ''}`,
    style: {
      '--tag-h': h
    },
    "aria-label": `Hue ${h}`,
    onClick: () => setTagColor(tag, h).catch(e => setErr('colour save failed: ' + e.message))
  }))))), !suggestions.length && React.createElement("div", {
    className: "tag-pop-hint"
  }, "Try \"work\" or \"personal\"."), err && React.createElement("div", {
    className: "tag-pop-hint",
    style: {
      color: 'var(--err)'
    }
  }, err)), document.body);
};
const SortMenu = ({
  sortBy,
  setSortBy
}) => {
  const [open, setOpen] = React.useState(false);
  const ref = React.useRef(null);
  React.useEffect(() => {
    if (!open) return;
    const onDoc = e => {
      if (ref.current && !ref.current.contains(e.target)) setOpen(false);
    };
    const onKey = e => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('mousedown', onDoc);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onDoc);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);
  return React.createElement("div", {
    className: "sort-menu",
    ref: ref
  }, React.createElement("button", {
    className: `btn btn-secondary btn-sm ${open ? 'open' : ''}`,
    onClick: () => setOpen(o => !o),
    title: "Sort sessions",
    "aria-haspopup": "listbox",
    "aria-expanded": open
  }, React.createElement("svg", {
    width: "14",
    height: "14",
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: "2",
    strokeLinecap: "round",
    strokeLinejoin: "round"
  }, React.createElement("line", {
    x1: "4",
    y1: "6",
    x2: "13",
    y2: "6"
  }), React.createElement("line", {
    x1: "4",
    y1: "12",
    x2: "11",
    y2: "12"
  }), React.createElement("line", {
    x1: "4",
    y1: "18",
    x2: "9",
    y2: "18"
  }), React.createElement("polyline", {
    points: "17 8 20 5 23 8"
  }), React.createElement("line", {
    x1: "20",
    y1: "5",
    x2: "20",
    y2: "19"
  }), React.createElement("polyline", {
    points: "17 16 20 19 23 16"
  })), React.createElement("span", null, "Sort"), React.createElement("svg", {
    className: "caret",
    width: "12",
    height: "12",
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: "2",
    strokeLinecap: "round",
    strokeLinejoin: "round"
  }, React.createElement("polyline", {
    points: "6 9 12 15 18 9"
  }))), open && React.createElement("div", {
    className: "sort-pop",
    role: "listbox"
  }, React.createElement("div", {
    className: "sort-pop-label"
  }, "Sort by"), SORT_OPTS.map(o => React.createElement("button", {
    key: o.value,
    className: `sort-opt ${o.value === sortBy ? 'active' : ''}`,
    role: "option",
    "aria-selected": o.value === sortBy,
    onClick: () => {
      setSortBy(o.value);
      setOpen(false);
    }
  }, React.createElement("span", null, o.label), o.value === sortBy && React.createElement(IconCheck, {
    size: 14
  })))));
};
const autoSleepOn = () => STORE.settings && STORE.settings.idle_offload_hours > 0;
const KeepAwakeButton = ({
  session: s,
  disabled
}) => {
  const [keep, setKeep] = React.useState(!!s.keepAwake);
  if (!autoSleepOn()) return null;
  const toggle = e => {
    e.stopPropagation();
    const next = !keep;
    setKeep(next);
    setSessionIconPatch(s.hostName, s.id, {
      keep_awake: next
    });
  };
  return React.createElement("button", {
    className: `action-btn icon ${keep ? 'starred' : ''}`,
    onClick: toggle,
    disabled: disabled,
    "data-tip": keep ? 'Keep awake: exempt from idle auto-offload' : 'Let this session auto-offload when idle',
    "aria-label": keep ? 'Allow auto-offload' : 'Keep awake'
  }, React.createElement(IconSun, {
    size: 14
  }));
};
const OffloadedNote = ({
  session: s
}) => {
  if (!s.workingDir && !s.command) return null;
  return React.createElement("div", {
    className: "muted mono",
    style: {
      fontSize: 11,
      marginTop: 2,
      paddingLeft: 28
    }
  }, s.workingDir ? `resume in ${s.workingDir}` : 'resume', s.command ? ` · runs ${s.command}` : '');
};
const FullSessionRow = ({
  session: s,
  onOpen,
  allTags,
  onTagClick
}) => {
  const [starred, setStarred] = React.useState(s.starred);
  const [tagAnchor, setTagAnchor] = React.useState(null);
  const [recreating, setRecreating] = React.useState(false);
  const [busy, setBusy] = React.useState(null);
  const toggleStar = e => {
    e.stopPropagation();
    const next = !starred;
    setStarred(next);
    setSessionIconPatch(s.hostName, s.id, {
      starred: next
    });
  };
  const onHandoff = async e => {
    e.stopPropagation();
    try {
      const cmd = await getHandoff(s.hostName, s.id);
      await navigator.clipboard.writeText(cmd);
    } catch (err) {
      alert('handoff failed: ' + err.message);
    }
  };
  const onKill = async e => {
    e.stopPropagation();
    if (busy) return;
    if (!confirm(`End session "${s.id}"? This forgets it entirely.`)) return;
    setBusy('ending');
    try {
      await killSession(s.hostName, s.id);
    } catch (err) {
      alert('end failed: ' + err.message);
    } finally {
      setBusy(null);
    }
  };
  const onOffload = async e => {
    e.stopPropagation();
    if (busy) return;
    if (!confirm(`Offload session "${s.id}"? The tmux session is killed but ssh-to-go keeps the working directory so you can resume it from the table below.`)) return;
    setBusy('offloading');
    try {
      await offloadSession(s.hostName, s.id);
    } catch (err) {
      alert('offload failed: ' + err.message);
    } finally {
      setBusy(null);
    }
  };
  const onPickIcon = e => {
    e.stopPropagation();
    if (!window.showIconPicker) return;
    window.showIconPicker(e.currentTarget, s.iconKind || 'terminal', (iconName, colorName) => {
      setSessionIconPatch(s.hostName, s.id, {
        icon: iconName,
        color: colorName
      });
    }, s.iconColor || 'default');
  };
  const onRename = async e => {
    e.stopPropagation();
    const next = prompt(`Rename session "${s.id}" to:`, s.id);
    if (!next || next === s.id) return;
    try {
      await renameSession(s.hostName, s.id, next);
    } catch (err) {
      alert('rename failed: ' + err.message);
    }
  };
  const onRecreate = async e => {
    e.stopPropagation();
    if (recreating) return;
    setRecreating(true);
    try {
      await recreateSession(s.hostName, s.id);
      openTerminal(s.hostName, s.id);
    } catch (err) {
      alert('recreate failed: ' + err.message);
    } finally {
      setRecreating(false);
    }
  };
  const onForget = async e => {
    e.stopPropagation();
    if (!confirm(`Forget session "${s.id}"? ssh-to-go drops the saved working directory; it can't be resumed afterwards.`)) return;
    try {
      await forgetSession(s.hostName, s.id);
    } catch (err) {
      alert('forget failed: ' + err.message);
    }
  };
  const offloaded = s.status === 'offloaded';
  return React.createElement("tr", {
    style: offloaded ? {
      opacity: 0.65
    } : null
  }, React.createElement("td", null, React.createElement("div", {
    className: "cell-session"
  }, React.createElement("button", {
    className: "sess-icon-btn",
    onClick: onPickIcon,
    title: "Change icon"
  }, React.createElement(SessIcon, {
    kind: s.iconKind,
    color: s.iconColor
  })), React.createElement("span", {
    className: "mono name",
    onClick: offloaded ? onRecreate : onOpen,
    style: {
      cursor: 'pointer'
    }
  }, s.id), !offloaded && React.createElement("button", {
    className: "rename-btn",
    onClick: onRename,
    "data-tip": "Rename this session",
    "aria-label": "Rename session"
  }, React.createElement(IconEdit, {
    size: 12
  })), offloaded && React.createElement(Pill, {
    variant: "muted"
  }, s.autoOffloaded ? 'slept' : 'offloaded'), s.tags.map(t => React.createElement(TagChip, {
    key: t,
    tag: t,
    title: `Show only "${t}" sessions`,
    onClick: onTagClick ? e => {
      e.stopPropagation();
      onTagClick(t);
    } : undefined
  })), React.createElement("button", {
    className: `rename-btn tag-btn ${s.tags.length ? '' : 'untagged'}`,
    "data-tip": "Edit tags",
    "aria-label": "Edit tags",
    onClick: e => {
      e.stopPropagation();
      setTagAnchor(tagAnchor ? null : e.currentTarget);
    }
  }, React.createElement(IconTag, {
    size: 12
  })), tagAnchor && React.createElement(TagEditor, {
    anchor: tagAnchor,
    session: s,
    allTags: allTags || [],
    onClose: () => setTagAnchor(null)
  })), offloaded && React.createElement(OffloadedNote, {
    session: s
  })), React.createElement("td", {
    className: "muted mono col-h3",
    style: {
      fontSize: 12.5
    }
  }, s.host), React.createElement("td", {
    className: "col-act"
  }, offloaded ? React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 12
    }
  }, "\u2014") : React.createElement(ActivityCell, {
    session: s
  })), React.createElement("td", {
    className: "col-h2"
  }, offloaded ? React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 12
    }
  }, "\u2014") : React.createElement(ClientCount, {
    count: s.clients
  })), React.createElement("td", {
    className: "muted num col-h4"
  }, s.uptime), React.createElement("td", null, React.createElement("div", {
    className: "actions-cell"
  }, offloaded ? React.createElement(React.Fragment, null, React.createElement("button", {
    className: "action-btn primary",
    onClick: onRecreate,
    disabled: recreating,
    title: "Bring the tmux session back at its saved working directory and open it in a new tab"
  }, recreating ? 'Recreating…' : 'Recreate'), React.createElement("button", {
    className: "action-btn",
    onClick: onForget,
    disabled: recreating,
    title: "Forget the saved working directory"
  }, "Forget")) : React.createElement(React.Fragment, null, React.createElement("button", {
    className: `action-btn icon star ${starred ? 'starred' : ''}`,
    onClick: toggleStar,
    disabled: !!busy,
    "data-tip": starred ? 'Remove from favorites' : 'Add to favorites',
    "aria-label": starred ? 'Remove from favorites' : 'Add to favorites'
  }, React.createElement(IconStar, {
    size: 14,
    fill: starred ? 'currentColor' : 'none'
  })), React.createElement("button", {
    className: "action-btn icon",
    onClick: onHandoff,
    disabled: !!busy,
    "data-tip": "Copy the SSH command to attach from your own terminal",
    "aria-label": "Copy SSH command"
  }, React.createElement(IconCopy, {
    size: 14
  })), React.createElement(KeepAwakeButton, {
    session: s,
    disabled: !!busy
  }), React.createElement("button", {
    className: `action-btn icon ${busy === 'offloading' ? 'busy' : ''}`,
    onClick: onOffload,
    disabled: !!busy,
    "data-tip": "Offload: stop it now, resume later in the same directory",
    "aria-label": "Offload session"
  }, React.createElement(IconMoon, {
    size: 14
  })), React.createElement("button", {
    className: `action-btn icon danger ${busy === 'ending' ? 'busy' : ''}`,
    onClick: onKill,
    disabled: !!busy,
    "data-tip": "Kill the session and stop tracking it",
    "aria-label": "Kill session"
  }, React.createElement(IconClose, {
    size: 14
  }))))));
};
Object.assign(window, {
  Sessions,
  FullSessionRow,
  KeepAwakeButton,
  OffloadedNote,
  TagChip
});
