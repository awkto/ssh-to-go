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
  React.useEffect(() => { localStorage.setItem('sshtogo.sessionSort', sortBy); }, [sortBy]);
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
    if (filter === 'attached' && s.clients.length === 0) return false;
    if (filter === 'favorites' && !s.starred) return false;
    if (search && !s.id.toLowerCase().includes(search.toLowerCase()) && !s.host.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  }).sort((a, b) => {
    // Offloaded sessions always sink to the bottom of the table.
    if ((a.status === 'offloaded') !== (b.status === 'offloaded')) {
      return a.status === 'offloaded' ? 1 : -1;
    }
    if (a.starred !== b.starred) return a.starred ? -1 : 1;
    return cmp(a, b);
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
  }, SESSIONS.filter(s => s.clients.length > 0).length)), React.createElement("button", {
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
  }), React.createElement("label", {
    className: "muted",
    style: { fontSize: 12, display: 'flex', alignItems: 'center', gap: 6 }
  }, "Sort:", React.createElement("select", {
    className: "select",
    style: { padding: '2px 6px', fontSize: 12 },
    value: sortBy,
    onChange: e => setSortBy(e.target.value)
  }, React.createElement("option", { value: "activity" }, "Recent activity"),
     React.createElement("option", { value: "opened" }, "Recently opened"),
     React.createElement("option", { value: "created" }, "Newest created"),
     React.createElement("option", { value: "name" }, "Name (A–Z)"))),
  React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 12
    }
  }, filtered.length, " shown")), React.createElement("div", {
    className: "panel"
  }, React.createElement("table", {
    className: "tbl"
  }, React.createElement("thead", null, React.createElement("tr", null, React.createElement("th", {
    style: {
      width: '26%'
    }
  }, "Session"), React.createElement("th", {
    className: "hide-mobile"
  }, "Host"), React.createElement("th", {
    className: "hide-mobile"
  }, "Activity"), React.createElement("th", {
    className: "hide-mobile"
  }, "Clients"), React.createElement("th", {
    className: "hide-mobile"
  }, "Window"), React.createElement("th", {
    className: "hide-mobile"
  }, "PID"), React.createElement("th", {
    className: "hide-mobile"
  }, "Uptime"), React.createElement("th", {
    style: {
      textAlign: 'right'
    }
  }, "Actions"))), React.createElement("tbody", null, filtered.map(s => React.createElement(FullSessionRow, {
    key: s.id,
    session: s,
    onOpen: () => openSession(s)
  }))))));
};
const FullSessionRow = ({
  session: s,
  onOpen
}) => {
  const [starred, setStarred] = React.useState(s.starred);
  const [recreating, setRecreating] = React.useState(false);
  // null | 'offloading' | 'ending'
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
    try { await forgetSession(s.hostName, s.id); } catch (err) { alert('forget failed: ' + err.message); }
  };
  const offloaded = s.status === 'offloaded';
  return React.createElement("tr", { style: offloaded ? { opacity: 0.65 } : null },
    React.createElement("td", null,
      React.createElement("div", { className: "cell-session" },
        React.createElement("button", { className: "sess-icon-btn", onClick: onPickIcon, title: "Change icon" },
          React.createElement(SessIcon, { kind: s.iconKind, color: s.iconColor })),
        React.createElement("span", { className: "mono name", onClick: offloaded ? onRecreate : onOpen, style: { cursor: 'pointer' } }, s.id),
        !offloaded && React.createElement("button", { className: "rename-btn", onClick: onRename, title: "Rename" },
          React.createElement(IconEdit, { size: 12 })),
        offloaded && React.createElement(Pill, { variant: "muted" }, "offloaded")),
      offloaded && s.workingDir && React.createElement("div", {
        className: "muted mono", style: { fontSize: 11, marginTop: 2, paddingLeft: 28 }
      }, "resume in ", s.workingDir)),
    React.createElement("td", { className: "muted mono hide-mobile", style: { fontSize: 12.5 } }, s.host),
    React.createElement("td", { className: "hide-mobile" },
      offloaded ? React.createElement("span", { className: "muted", style: { fontSize: 12 } }, "—")
                : React.createElement(ActivityCell, { session: s })),
    React.createElement("td", { className: "hide-mobile" },
      offloaded ? React.createElement("span", { className: "muted", style: { fontSize: 12 } }, "—")
                : React.createElement(Presence, { clients: s.clients })),
    React.createElement("td", { className: "mono num muted hide-mobile", style: { fontSize: 12 } }, offloaded ? "—" : s.win),
    React.createElement("td", { className: "mono num muted hide-mobile", style: { fontSize: 12 } }, offloaded ? "—" : s.pid),
    React.createElement("td", { className: "muted num hide-mobile" }, s.uptime),
    React.createElement("td", null, React.createElement("div", { className: "actions-cell" },
      offloaded
        ? [
            React.createElement("button", { key: "rec", className: "action-btn primary", onClick: onRecreate, disabled: recreating,
                title: "Bring the tmux session back at its saved working directory and open it in a new tab" },
              recreating ? "Recreating…" : "Recreate"),
            React.createElement("button", { key: "fgt", className: "action-btn", onClick: onForget, disabled: recreating, title: "Forget the saved working directory" }, "Forget"),
          ]
        : [
            React.createElement("button", { key: "star", className: `action-btn star ${starred ? 'starred' : ''}`, onClick: toggleStar, disabled: !!busy },
              React.createElement(IconStar, { size: 13, fill: starred ? 'currentColor' : 'none' })),
            React.createElement("button", { key: "open", className: "action-btn primary", onClick: onOpen, disabled: !!busy }, "Open"),
            React.createElement("button", { key: "h", className: "action-btn", onClick: onHandoff, disabled: !!busy, title: "Copy SSH handoff command" }, "Handoff"),
            React.createElement("button", { key: "off", className: "action-btn", onClick: onOffload, disabled: !!busy, title: "Stop tmux but keep tracked so you can resume from the same directory" },
              busy === 'offloading' ? "Offloading…" : "Offload"),
            React.createElement("button", { key: "end", className: "action-btn danger", onClick: onKill, disabled: !!busy },
              busy === 'ending' ? "Ending…" : "End"),
          ])));
};
Object.assign(window, {
  Sessions,
  FullSessionRow
});