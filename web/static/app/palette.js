const Palette = ({
  store,
  onClose,
  onOpenSession,
  setView,
  openNewSession
}) => {
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
      icon: () => React.createElement(SessIcon, {
        kind: s.iconKind,
        color: s.iconColor
      }),
      action: () => {
        onOpenSession(s);
        onClose();
      }
    }));
    const hosts = HOSTS.filter(h => !q || h.fqdn.toLowerCase().includes(ql)).slice(0, 4).map(h => ({
      kind: 'host',
      title: h.fqdn,
      sub: `${h.user} · ${h.sessions} sessions`,
      icon: IconServer,
      action: () => {
        setView('hosts');
        onClose();
      }
    }));
    const actions = [{
      kind: 'action',
      title: 'New session',
      sub: '⌘N',
      icon: IconPlus,
      action: () => {
        openNewSession();
        onClose();
      }
    }, {
      kind: 'action',
      title: 'Go to dashboard',
      icon: IconDashboard,
      action: () => {
        setView('dashboard');
        onClose();
      }
    }, {
      kind: 'action',
      title: 'Go to sessions',
      icon: IconTerminal,
      action: () => {
        setView('sessions');
        onClose();
      }
    }, {
      kind: 'action',
      title: 'Go to hosts',
      icon: IconServer,
      action: () => {
        setView('hosts');
        onClose();
      }
    }, {
      kind: 'action',
      title: 'Go to settings',
      icon: IconSettings,
      action: () => {
        setView('settings');
        onClose();
      }
    }, {
      kind: 'action',
      title: 'Toggle theme',
      sub: '⇧⌘L',
      icon: IconMoon,
      action: () => {
        document.documentElement.dataset.theme = document.documentElement.dataset.theme === 'light' ? 'dark' : 'light';
        onClose();
      }
    }].filter(a => !q || a.title.toLowerCase().includes(ql));
    return [{
      label: 'Sessions',
      items: sessions
    }, {
      label: 'Hosts',
      items: hosts
    }, {
      label: 'Actions',
      items: actions
    }].filter(g => g.items.length);
  }, [q, SESSIONS, HOSTS]);
  const flat = React.useMemo(() => groups.flatMap(g => g.items), [groups]);
  React.useEffect(() => {
    setIdx(0);
  }, [q]);
  React.useEffect(() => {
    const onKey = e => {
      if (e.key === 'Escape') onClose();else if (e.key === 'ArrowDown') {
        e.preventDefault();
        setIdx(i => Math.min(flat.length - 1, i + 1));
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setIdx(i => Math.max(0, i - 1));
      } else if (e.key === 'Enter') {
        e.preventDefault();
        flat[idx]?.action?.();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [flat, idx]);
  let runningIdx = 0;
  return React.createElement("div", {
    className: "palette-backdrop",
    onClick: onClose
  }, React.createElement("div", {
    className: "palette",
    onClick: e => e.stopPropagation()
  }, React.createElement("div", {
    className: "palette-input"
  }, React.createElement(IconSearch, {
    size: 16
  }), React.createElement("input", {
    autoFocus: true,
    placeholder: "Search sessions, hosts, or run commands\u2026",
    value: q,
    onChange: e => setQ(e.target.value)
  }), React.createElement(Kbd, null, "esc")), React.createElement("div", {
    className: "palette-results"
  }, groups.map((g, gi) => React.createElement("div", {
    key: gi
  }, React.createElement("div", {
    className: "palette-group-label"
  }, g.label), g.items.map((it, i) => {
    const myIdx = runningIdx++;
    const I = it.icon;
    return React.createElement("div", {
      key: i,
      className: `palette-item ${myIdx === idx ? 'active' : ''}`,
      onMouseEnter: () => setIdx(myIdx),
      onClick: it.action
    }, React.createElement("span", {
      className: "palette-icon"
    }, typeof I === 'function' && I.length === 0 ? React.createElement(I, null) : React.createElement(I, {
      size: 13
    })), React.createElement("span", {
      className: "palette-title"
    }, it.title), it.sub && React.createElement("span", {
      className: "palette-sub"
    }, it.sub));
  }))), groups.length === 0 && React.createElement("div", {
    style: {
      padding: '24px 18px',
      color: 'var(--fg-subtle)',
      fontSize: 13,
      textAlign: 'center'
    }
  }, "No results for \"", q, "\"")), React.createElement("div", {
    className: "palette-footer"
  }, React.createElement("span", {
    className: "hint"
  }, React.createElement(Kbd, null, "\u21B5"), " open"), React.createElement("span", {
    className: "hint"
  }, React.createElement(Kbd, null, "\u2191"), React.createElement(Kbd, null, "\u2193"), " navigate"), React.createElement("span", {
    className: "hint"
  }, React.createElement(Kbd, null, "\u2318"), React.createElement(Kbd, null, "K"), " toggle"), React.createElement("span", {
    style: {
      marginLeft: 'auto'
    },
    className: "mono"
  }, flat.length, " results"))));
};
Object.assign(window, {
  Palette
});