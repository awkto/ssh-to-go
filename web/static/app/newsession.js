const NewSession = ({
  store,
  onClose
}) => {
  const HOSTS = store.hosts;
  const [host, setHost] = React.useState(HOSTS[0] ? HOSTS[0].id : '');
  const [name, setName] = React.useState('');
  const [cwd, setCwd] = React.useState('');
  const [attach, setAttach] = React.useState(true);
  const [showAdvanced, setShowAdvanced] = React.useState(false);
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');
  React.useEffect(() => {
    if (!host && HOSTS[0]) setHost(HOSTS[0].id);
  }, [HOSTS.length]);
  const submit = async e => {
    if (e && e.preventDefault) e.preventDefault();
    if (!host) {
      setErr('Pick a host first.');
      return;
    }
    setErr('');
    setBusy(true);
    try {
      const finalName = name.trim() || `session-${Math.random().toString(36).slice(2, 7)}`;
      await createSession(host, finalName, cwd.trim() || '');
      onClose();
      if (attach) openTerminal(host, finalName);
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
  }, React.createElement("form", {
    onSubmit: submit
  }, React.createElement("div", {
    className: "modal-head"
  }, React.createElement("div", {
    style: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      gap: 12
    }
  }, React.createElement("div", null, React.createElement("h3", null, "New session"), React.createElement("p", null, "Press Create to spin up a detached tmux session. All fields are optional.")), React.createElement("button", {
    type: "button",
    className: "icon-btn",
    onClick: onClose
  }, React.createElement(IconClose, {
    size: 15
  })))), React.createElement("div", {
    className: "modal-body"
  }, React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Session name ", React.createElement("span", {
    className: "muted",
    style: {
      fontWeight: 400,
      fontSize: 11.5
    }
  }, "(optional \u2014 auto-generated if empty)")), React.createElement("input", {
    className: "input mono",
    placeholder: "e.g. claude-code",
    value: name,
    onChange: e => setName(e.target.value),
    autoFocus: true
  })), HOSTS.length === 0 && React.createElement("div", {
    className: "muted",
    style: {
      fontSize: 12.5,
      marginBottom: 12
    }
  }, "No hosts registered yet \u2014 add one from the Hosts page first."), HOSTS.length > 1 && React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Target host"), React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 6
    }
  }, HOSTS.map(h => React.createElement("div", {
    key: h.id,
    className: `radio-card ${host === h.id ? 'selected' : ''}`,
    onClick: () => setHost(h.id)
  }, React.createElement(StatusDot, {
    status: h.status === 'online' ? 'active' : 'offline'
  }), React.createElement("div", {
    style: {
      flex: 1
    }
  }, React.createElement("div", {
    className: "radio-title mono"
  }, h.fqdn), React.createElement("div", {
    className: "radio-sub"
  }, h.user, "@", h.fqdn.split(':')[0], " \xB7 ", h.os)), React.createElement(Pill, {
    variant: h.sessions > 0 ? 'accent' : 'default',
    mono: true
  }, h.sessions, " sess"))))), HOSTS.length === 1 && React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Target host"), React.createElement("div", {
    className: "radio-card selected",
    style: {
      cursor: 'default'
    }
  }, React.createElement(StatusDot, {
    status: HOSTS[0].status === 'online' ? 'active' : 'offline'
  }), React.createElement("div", {
    style: {
      flex: 1
    }
  }, React.createElement("div", {
    className: "radio-title mono"
  }, HOSTS[0].fqdn), React.createElement("div", {
    className: "radio-sub"
  }, HOSTS[0].user, "@", HOSTS[0].fqdn.split(':')[0], " \xB7 ", HOSTS[0].os)))), React.createElement("button", {
    type: "button",
    className: "btn btn-ghost btn-sm",
    onClick: () => setShowAdvanced(v => !v),
    style: {
      padding: '4px 0',
      marginTop: 4
    }
  }, showAdvanced ? '▾' : '▸', " Advanced options"), showAdvanced && React.createElement("div", {
    style: {
      marginTop: 10
    }
  }, React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Working directory"), React.createElement("input", {
    className: "input mono",
    value: cwd,
    onChange: e => setCwd(e.target.value),
    placeholder: "~/"
  }), React.createElement("div", {
    className: "hint"
  }, "Defaults to the user's home. The tmux session is blank \u2014 run any command after attach.")), React.createElement("div", {
    className: "field"
  }, React.createElement("label", {
    className: "checkbox"
  }, React.createElement("input", {
    type: "checkbox",
    checked: attach,
    onChange: e => setAttach(e.target.checked)
  }), " Attach immediately"))), err && React.createElement("div", {
    style: {
      color: 'var(--err)',
      fontSize: 12.5,
      marginTop: 10
    }
  }, err)), React.createElement("div", {
    className: "modal-foot"
  }, React.createElement(Button, {
    variant: "ghost",
    type: "button",
    onClick: onClose
  }, "Cancel"), React.createElement("div", {
    style: {
      flex: 1
    }
  }), React.createElement(Button, {
    variant: "primary",
    type: "submit",
    icon: IconPlay,
    disabled: busy || !host
  }, busy ? 'Creating…' : attach ? 'Create & attach' : 'Create')))));
};
Object.assign(window, {
  NewSession
});