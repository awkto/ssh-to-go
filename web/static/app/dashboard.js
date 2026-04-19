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
    style: {
      maxHeight: 440,
      overflowY: 'auto'
    }
  }, React.createElement("table", {
    className: "tbl"
  }, React.createElement("thead", null, React.createElement("tr", null, React.createElement("th", {
    style: {
      width: '30%'
    }
  }, "Session"), React.createElement("th", null, "Host"), React.createElement("th", null, "Activity"), React.createElement("th", null, "Clients"), React.createElement("th", null, "Uptime"), React.createElement("th", {
    style: {
      textAlign: 'right'
    }
  }, "Actions"))), React.createElement("tbody", null, SESSIONS.slice(0, 8).map(s => React.createElement(SessionRow, {
    key: s.id,
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
  }, "last 60s")), React.createElement("div", {
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
  }, h.fqdn)), React.createElement(Sparkline, {
    data: h.load,
    width: 120,
    height: 28
  }), React.createElement("div", {
    className: "host-mini-meta mt-2"
  }, React.createElement("span", null, "CPU ", h.cpu, "%"), React.createElement("span", null, "MEM ", h.mem, "%")))))), React.createElement("div", {
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
const SessionRow = ({
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
  return React.createElement("tr", null, React.createElement("td", null, React.createElement("div", {
    className: "cell-session",
    onClick: onOpen,
    style: {
      cursor: 'pointer'
    }
  }, React.createElement(SessIcon, {
    kind: s.iconKind,
    color: s.iconColor
  }), React.createElement("span", {
    className: "mono name"
  }, s.id))), React.createElement("td", {
    className: "muted mono",
    style: {
      fontSize: 12.5
    }
  }, s.host), React.createElement("td", null, React.createElement(ActivityCell, {
    session: s
  })), React.createElement("td", null, React.createElement(Presence, {
    clients: s.clients
  })), React.createElement("td", {
    className: "muted num"
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
  Dashboard,
  StatCard,
  SessionRow
});