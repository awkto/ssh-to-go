const Dashboard = ({
  store,
  setView,
  openSession,
  openNewSession
}) => {
  const HOSTS = store.hosts;
  const SESSIONS = store.sessions;
  const KEYPAIRS = store.keypairs;
  const activeCount = SESSIONS.filter(s => s.activity === 'active').length;
  const attached = SESSIONS.filter(s => s.clients.length > 0).length;
  const totalHostLoad = HOSTS.length ? HOSTS.reduce((sum, h) => sum.map((v, i) => v + (h.load[i] || 0)), Array(20).fill(0)).map(v => v / HOSTS.length) : Array(20).fill(0);
  return React.createElement("div", null, React.createElement("div", {
    className: "page-head"
  }, React.createElement("div", {
    className: "page-title-block"
  }, React.createElement("h1", null, "Dashboard"), React.createElement("p", null, HOSTS.length, " hosts online \xB7 ", activeCount, " sessions active \xB7 ", attached, " with clients attached")), React.createElement("div", {
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
    className: "stats-grid"
  }, React.createElement(StatCard, {
    label: "Hosts",
    value: HOSTS.length,
    sub: React.createElement(React.Fragment, null, React.createElement("span", {
      className: "dot ok",
      style: {
        width: 6,
        height: 6,
        boxShadow: 'none',
        display: 'inline-block',
        marginRight: 5
      }
    }), "All online"),
    delta: null,
    spark: React.createElement(Sparkline, {
      data: totalHostLoad,
      width: 70,
      height: 20
    })
  }), React.createElement(StatCard, {
    label: "Active sessions",
    value: SESSIONS.length,
    sub: React.createElement("span", null, activeCount, " active \xB7 ", SESSIONS.length - activeCount, " idle"),
    delta: {
      dir: 'up',
      val: '+3'
    },
    spark: React.createElement(Sparkline, {
      data: [8, 9, 10, 9, 10, 11, 12, 12, 11, 12],
      width: 70,
      height: 20
    })
  }), React.createElement(StatCard, {
    label: "Attached clients",
    value: React.createElement(React.Fragment, null, SESSIONS.reduce((n, s) => n + s.clients.length, 0), React.createElement("span", {
      style: {
        fontSize: 14,
        color: 'var(--fg-subtle)',
        fontWeight: 500,
        marginLeft: 6
      }
    }, "across ", attached)),
    sub: React.createElement(Presence, {
      clients: [{
        name: 'AC',
        color: 'indigo'
      }, {
        name: 'MB',
        color: 'teal'
      }, {
        name: 'JP',
        color: 'amber'
      }, {
        name: 'SR',
        color: 'violet'
      }],
      max: 4
    }),
    delta: null
  }), React.createElement(StatCard, {
    label: "SSH keypairs",
    value: KEYPAIRS.length,
    sub: (() => {
      const def = KEYPAIRS.find(k => k.isDefault) || KEYPAIRS[0];
      return React.createElement("span", {
        className: "mono",
        style: {
          fontSize: 11.5
        }
      }, def ? `default: ${def.name} · ${def.type}` : 'no keypairs yet');
    })(),
    delta: null,
    icon: React.createElement(IconKey, {
      size: 13
    })
  })), React.createElement("div", {
    className: "grid-2"
  }, React.createElement("div", {
    className: "panel"
  }, React.createElement("div", {
    className: "panel-head"
  }, React.createElement("div", {
    className: "row gap-3"
  }, React.createElement("h2", null, "Recent sessions"), React.createElement("div", {
    className: "seg"
  }, React.createElement("span", {
    className: "seg-btn active"
  }, "All ", React.createElement("span", {
    className: "count"
  }, SESSIONS.length)), React.createElement("span", {
    className: "seg-btn"
  }, "Active ", React.createElement("span", {
    className: "count"
  }, activeCount)), React.createElement("span", {
    className: "seg-btn"
  }, "Attached ", React.createElement("span", {
    className: "count"
  }, attached)))), React.createElement("button", {
    className: "btn btn-ghost btn-sm",
    onClick: () => setView('sessions')
  }, "View all ", React.createElement(IconArrowRight, {
    size: 12
  }))), React.createElement("div", {
    className: "tbl-scroll",
    style: {
      maxHeight: 640,
      overflowY: 'auto'
    }
  }, React.createElement("table", {
    className: "tbl tbl-fixed"
  }, React.createElement("thead", null, React.createElement("tr", null, React.createElement("th", {
    style: {
      width: '30%'
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
  }, "Actions"))), React.createElement("tbody", null, SESSIONS.slice().sort((a, b) => {
    if (a.status === 'offloaded' !== (b.status === 'offloaded')) {
      return a.status === 'offloaded' ? 1 : -1;
    }
    if (a.starred !== b.starred) return a.starred ? -1 : 1;
    const aRecent = Math.max(a.activityMs || 0, a.lastAccessedMs || 0);
    const bRecent = Math.max(b.activityMs || 0, b.lastAccessedMs || 0);
    if (bRecent !== aRecent) return bRecent - aRecent;
    return b.createdMs - a.createdMs;
  }).slice(0, 20).map(s => React.createElement(SessionRow, {
    key: `${s.hostName}:${s.id}`,
    session: s,
    onOpen: () => openSession(s)
  })))))), React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 16
    }
  }, React.createElement("div", {
    className: "panel"
  }, React.createElement("div", {
    className: "panel-head"
  }, React.createElement("h2", null, "Host load"), React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 11.5,
      fontFamily: 'var(--font-mono)'
    }
  }, "cpu \xB7 mem")), React.createElement("div", {
    className: "host-mini-grid"
  }, HOSTS.map(h => React.createElement("div", {
    key: h.id,
    className: "host-mini"
  }, React.createElement("div", {
    className: "host-mini-head"
  }, React.createElement(StatusDot, {
    status: h.status === 'online' ? 'active' : 'offline'
  }), React.createElement("span", {
    className: "host-mini-name truncate"
  }, h.fqdn)), React.createElement("div", {
    className: "host-bar-row"
  }, React.createElement(HostBar, {
    label: "CPU",
    value: h.cpu
  }), React.createElement(HostBar, {
    label: "MEM",
    value: h.mem
  })))))), React.createElement("div", {
    className: "panel"
  }, React.createElement("div", {
    className: "panel-head"
  }, React.createElement("h2", null, "Activity"), React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 11.5
    }
  }, "coming soon")), React.createElement("div", {
    style: {
      padding: '28px 16px',
      color: 'var(--fg-subtle)',
      fontSize: 12.5,
      textAlign: 'center'
    }
  }, "Live activity feed lands with backend event log (issue #20).")))));
};
const HostBar = ({
  label,
  value
}) => {
  const has = value != null && !isNaN(value);
  const pct = has ? Math.max(0, Math.min(100, Number(value))) : 0;
  const color = !has ? 'transparent' : pct < 60 ? 'var(--ok)' : pct < 85 ? 'var(--warn)' : 'var(--err)';
  return React.createElement("div", {
    className: "host-bar"
  }, React.createElement("div", {
    className: "host-bar-track"
  }, React.createElement("div", {
    className: "host-bar-fill",
    style: {
      height: pct + '%',
      background: color
    }
  })), React.createElement("div", {
    className: "host-bar-label"
  }, label), React.createElement("div", {
    className: "host-bar-value mono",
    style: {
      color: has ? 'var(--fg)' : 'var(--fg-faint)'
    }
  }, has ? `${Math.round(pct)}%` : '—'));
};
const StatCard = ({
  label,
  value,
  sub,
  delta,
  spark,
  icon
}) => React.createElement("div", {
  className: "stat"
}, React.createElement("div", {
  className: "stat-label"
}, icon, label), React.createElement("div", {
  className: "stat-value"
}, React.createElement("span", null, value), delta && React.createElement("span", {
  className: `stat-delta ${delta.dir}`
}, delta.val)), React.createElement("div", {
  className: "stat-sub"
}, sub), spark && React.createElement("div", {
  className: "stat-spark"
}, spark));
const MissingSessionsPanel = ({
  missing
}) => {
  const [busy, setBusy] = React.useState({});
  const action = async (fn, m, label) => {
    const key = `${m.hostName}:${m.name}`;
    setBusy(b => ({
      ...b,
      [key]: label
    }));
    try {
      await fn(m.hostName, m.name);
    } catch (err) {
      alert(`${label} failed: ${err.message}`);
    } finally {
      setBusy(b => {
        const n = {
          ...b
        };
        delete n[key];
        return n;
      });
    }
  };
  return React.createElement("div", {
    className: "panel",
    style: {
      marginBottom: 16,
      borderColor: 'var(--warn, #c89b3c)'
    }
  }, React.createElement("div", {
    className: "panel-head"
  }, React.createElement("div", {
    className: "row gap-3"
  }, React.createElement("h2", {
    style: {
      color: 'var(--warn, #c89b3c)'
    }
  }, "Resumable sessions"), React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 12
    }
  }, "tracked but not running \u2014 either you offloaded them or the host rebooted")), React.createElement("span", {
    className: "muted mono",
    style: {
      fontSize: 12
    }
  }, missing.length)), React.createElement("table", {
    className: "tbl"
  }, React.createElement("thead", null, React.createElement("tr", null, React.createElement("th", {
    style: {
      width: '24%'
    }
  }, "Session"), React.createElement("th", null, "Host"), React.createElement("th", {
    className: "hide-mobile"
  }, "Last working dir"), React.createElement("th", {
    style: {
      textAlign: 'right'
    }
  }, "Actions"))), React.createElement("tbody", null, missing.map(m => {
    const key = `${m.hostName}:${m.name}`;
    const b = busy[key];
    return React.createElement("tr", {
      key: key
    }, React.createElement("td", null, React.createElement("span", {
      className: "mono"
    }, m.name)), React.createElement("td", {
      className: "muted mono",
      style: {
        fontSize: 12.5
      }
    }, m.host), React.createElement("td", {
      className: "mono muted hide-mobile",
      style: {
        fontSize: 12
      }
    }, m.workingDir || '—'), React.createElement("td", null, React.createElement("div", {
      className: "actions-cell"
    }, React.createElement("button", {
      className: "action-btn primary",
      disabled: !!b,
      onClick: () => action(recreateSession, m, 'recreate')
    }, b === 'recreate' ? '…' : 'Recreate'), React.createElement("button", {
      className: "action-btn",
      disabled: !!b,
      onClick: () => action(forgetSession, m, 'forget')
    }, b === 'forget' ? '…' : 'Forget'))));
  }))));
};
const SessionRow = ({
  session: s,
  onOpen
}) => {
  const [starred, setStarred] = React.useState(s.starred);
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
  }, "offloaded")), offloaded && s.workingDir && React.createElement("div", {
    className: "muted mono",
    style: {
      fontSize: 11,
      marginTop: 2,
      paddingLeft: 28
    }
  }, "resume in ", s.workingDir)), React.createElement("td", {
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
  }, "\u2014") : React.createElement(Presence, {
    clients: s.clients
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
  })), React.createElement("button", {
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
  Dashboard,
  StatCard,
  SessionRow,
  HostBar,
  MissingSessionsPanel
});