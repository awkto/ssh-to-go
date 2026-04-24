const Sessions = ({
  store,
  openSession,
  openNewSession,
  initialFilter
}) => {
  const SESSIONS = store.sessions;
  const [filter, setFilter] = React.useState(initialFilter || 'all');
  const [search, setSearch] = React.useState('');
  React.useEffect(() => {
    if (initialFilter) setFilter(initialFilter);
  }, [initialFilter]);
  const filtered = SESSIONS.filter(s => {
    if (filter === 'active' && s.activity !== 'active') return false;
    if (filter === 'attached' && s.clients.length === 0) return false;
    if (filter === 'favorites' && !s.starred) return false;
    if (search && !s.id.toLowerCase().includes(search.toLowerCase()) && !s.host.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
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
  }), React.createElement("span", {
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
    if (!confirm(`End session "${s.id}"?`)) return;
    try {
      await killSession(s.hostName, s.id);
    } catch (err) {
      alert('end failed: ' + err.message);
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
  return React.createElement("tr", null, React.createElement("td", null, React.createElement("div", {
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
    onClick: onOpen,
    style: {
      cursor: 'pointer'
    }
  }, s.id), React.createElement("button", {
    className: "rename-btn",
    onClick: onRename,
    title: "Rename"
  }, React.createElement(IconEdit, {
    size: 12
  })))), React.createElement("td", {
    className: "muted mono hide-mobile",
    style: {
      fontSize: 12.5
    }
  }, s.host), React.createElement("td", {
    className: "hide-mobile"
  }, React.createElement(ActivityCell, {
    session: s
  })), React.createElement("td", {
    className: "hide-mobile"
  }, React.createElement(Presence, {
    clients: s.clients
  })), React.createElement("td", {
    className: "mono num muted hide-mobile",
    style: {
      fontSize: 12
    }
  }, s.win), React.createElement("td", {
    className: "mono num muted hide-mobile",
    style: {
      fontSize: 12
    }
  }, s.pid), React.createElement("td", {
    className: "muted num hide-mobile"
  }, s.uptime), React.createElement("td", null, React.createElement("div", {
    className: "actions-cell"
  }, React.createElement("button", {
    className: `action-btn star ${starred ? 'starred' : ''}`,
    onClick: toggleStar
  }, React.createElement(IconStar, {
    size: 13,
    fill: starred ? 'currentColor' : 'none'
  })), React.createElement("button", {
    className: "action-btn primary",
    onClick: onOpen
  }, "Open"), React.createElement("button", {
    className: "action-btn",
    onClick: onHandoff,
    title: "Copy SSH handoff command"
  }, "Handoff"), React.createElement("button", {
    className: "action-btn danger",
    onClick: onKill
  }, "End"))));
};
Object.assign(window, {
  Sessions,
  FullSessionRow
});