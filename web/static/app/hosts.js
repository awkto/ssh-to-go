const Hosts = ({
  store,
  openNewSession
}) => {
  const HOSTS = store.hosts;
  const [search, setSearch] = React.useState('');
  const [addOpen, setAddOpen] = React.useState(false);
  const filtered = HOSTS.filter(h => !search || h.fqdn.includes(search));
  return React.createElement("div", null, React.createElement("div", {
    className: "page-head"
  }, React.createElement("div", {
    className: "page-title-block"
  }, React.createElement("h1", null, "Hosts"), React.createElement("p", null, HOSTS.length, " hosts registered \xB7 ", HOSTS.filter(h => h.status === 'online').length, " online")), React.createElement("div", {
    className: "page-actions"
  }, React.createElement(Button, {
    variant: "secondary",
    size: "sm",
    icon: IconRefresh,
    onClick: () => scanAll().catch(err => alert(err.message))
  }, "Sync all"), React.createElement(Button, {
    variant: "primary",
    size: "sm",
    icon: IconPlus,
    onClick: () => setAddOpen(true)
  }, "Add host"))), React.createElement("div", {
    className: "filter-bar"
  }, React.createElement("div", {
    className: "seg"
  }, React.createElement("button", {
    className: "seg-btn active"
  }, "All ", React.createElement("span", {
    className: "count"
  }, HOSTS.length)), React.createElement("button", {
    className: "seg-btn"
  }, "Online ", React.createElement("span", {
    className: "count"
  }, HOSTS.filter(h => h.status === 'online').length)), React.createElement("button", {
    className: "seg-btn"
  }, "With sessions ", React.createElement("span", {
    className: "count"
  }, HOSTS.filter(h => h.sessions > 0).length))), React.createElement("div", {
    className: "search-sm"
  }, React.createElement(IconSearch, {
    size: 13
  }), React.createElement("input", {
    placeholder: "Filter hosts\u2026",
    value: search,
    onChange: e => setSearch(e.target.value)
  }))), React.createElement("div", {
    className: "panel"
  }, React.createElement("table", {
    className: "tbl"
  }, React.createElement("thead", null, React.createElement("tr", null, React.createElement("th", {
    style: {
      width: '28%'
    }
  }, "Host (FQDN)"), React.createElement("th", null, "Sessions"), React.createElement("th", null, "Operating system"), React.createElement("th", null, "Load (60s)"), React.createElement("th", null, "CPU / MEM"), React.createElement("th", null, "Last sync"), React.createElement("th", {
    style: {
      textAlign: 'right'
    }
  }, "Actions"))), React.createElement("tbody", null, filtered.map(h => React.createElement("tr", {
    key: h.id,
    className: "host-row"
  }, React.createElement("td", null, React.createElement("div", {
    className: "row gap-3"
  }, React.createElement(StatusDot, {
    status: h.status === 'online' ? 'active' : 'offline',
    pulse: h.status === 'online'
  }), React.createElement("div", null, React.createElement("div", {
    className: "mono host-name"
  }, h.fqdn), React.createElement("div", {
    className: "host-user"
  }, h.user, "@", h.fqdn.split(':')[0])))), React.createElement("td", null, h.sessions > 0 ? React.createElement(Pill, {
    variant: "accent"
  }, h.sessions, " sessions") : React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 12.5
    }
  }, "No sessions")), React.createElement("td", null, React.createElement(OsChip, {
    os: h.os,
    kind: h.osKind
  })), React.createElement("td", null, h.cpu != null ? React.createElement(Sparkline, {
    data: h.load,
    width: 100,
    height: 22
  }) : React.createElement("span", {
    className: "subtle mono",
    style: {
      fontSize: 11
    }
  }, "\u2014")), React.createElement("td", {
    className: "mono num",
    style: {
      fontSize: 12
    }
  }, h.cpu != null ? React.createElement(React.Fragment, null, React.createElement("span", {
    style: {
      color: h.cpu > 60 ? 'var(--warn)' : 'var(--fg-muted)'
    }
  }, h.cpu, "%"), React.createElement("span", {
    className: "muted"
  }, " / "), React.createElement("span", {
    className: "muted"
  }, h.mem, "%")) : React.createElement("span", {
    className: "subtle",
    style: {
      fontSize: 11
    }
  }, "\u2014")), React.createElement("td", {
    className: "muted mono",
    style: {
      fontSize: 12
    }
  }, h.lastSync), React.createElement("td", null, React.createElement("div", {
    className: "actions-cell"
  }, React.createElement("button", {
    className: "action-btn primary",
    onClick: openNewSession
  }, React.createElement(IconPlus, {
    size: 12
  }), "New session"), React.createElement("button", {
    className: "action-btn",
    onClick: () => scanAll().catch(err => alert(err.message)),
    title: "Re-scan this host"
  }, React.createElement(IconRefresh, {
    size: 12
  }), "Sync")))))))), addOpen && React.createElement(AddHostModal, {
    onClose: () => setAddOpen(false)
  }));
};
const AddHostModal = ({
  onClose
}) => {
  const [name, setName] = React.useState('');
  const [address, setAddress] = React.useState('');
  const [port, setPort] = React.useState(22);
  const [user, setUser] = React.useState('');
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');
  const submit = async e => {
    e.preventDefault();
    setErr('');
    setBusy(true);
    try {
      await addHost({
        name: name || undefined,
        address,
        port: Number(port) || 22,
        user: user || undefined
      });
      onClose();
    } catch (ex) {
      setErr(ex.message || 'failed');
    } finally {
      setBusy(false);
    }
  };
  return React.createElement("div", {
    className: "modal-backdrop",
    onClick: onClose
  }, React.createElement("div", {
    className: "modal",
    onClick: e => e.stopPropagation()
  }, React.createElement("div", {
    className: "modal-head"
  }, React.createElement("div", {
    style: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      gap: 12
    }
  }, React.createElement("div", null, React.createElement("h3", null, "Add host"), React.createElement("p", null, "Register a target host for tmux sessions.")), React.createElement("button", {
    className: "icon-btn",
    onClick: onClose
  }, React.createElement(IconClose, {
    size: 15
  })))), React.createElement("form", {
    onSubmit: submit
  }, React.createElement("div", {
    className: "modal-body"
  }, React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Address (host or IP)"), React.createElement("input", {
    className: "input mono",
    value: address,
    onChange: e => setAddress(e.target.value),
    placeholder: "host.example.com",
    required: true,
    autoFocus: true
  })), React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Port"), React.createElement("input", {
    className: "input mono",
    type: "number",
    value: port,
    onChange: e => setPort(e.target.value)
  })), React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "User"), React.createElement("input", {
    className: "input mono",
    value: user,
    onChange: e => setUser(e.target.value),
    placeholder: "leave empty to use default"
  })), React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Name"), React.createElement("input", {
    className: "input mono",
    value: name,
    onChange: e => setName(e.target.value),
    placeholder: "defaults to address"
  })), err && React.createElement("div", {
    style: {
      color: 'var(--err)',
      fontSize: 12.5
    }
  }, err)), React.createElement("div", {
    className: "modal-foot"
  }, React.createElement(Button, {
    variant: "ghost",
    onClick: onClose
  }, "Cancel"), React.createElement(Button, {
    variant: "primary",
    type: "submit",
    disabled: busy
  }, busy ? 'Adding…' : 'Add host')))));
};
Object.assign(window, {
  Hosts,
  AddHostModal
});