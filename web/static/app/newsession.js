const NS_LS = {
  createDir: 's2g:newsession:create-dir',
  launch: 's2g:newsession:launch',
  command: 's2g:newsession:command'
};
const nsGet = (k, fallback) => {
  try {
    const v = localStorage.getItem(k);
    return v === null ? fallback : v;
  } catch (_) {
    return fallback;
  }
};
const nsSet = (k, v) => {
  try {
    localStorage.setItem(k, v);
  } catch (_) {}
};
const NewSession = ({
  store,
  onClose
}) => {
  const HOSTS = store.hosts;
  const defaultDir = store.settings && store.settings.new_session_dir || '~/sessions/';
  const [host, setHost] = React.useState(HOSTS[0] ? HOSTS[0].id : '');
  const [name, setName] = React.useState('');
  const [cwd, setCwd] = React.useState(defaultDir);
  const [createDir, setCreateDir] = React.useState(nsGet(NS_LS.createDir, '1') === '1');
  const [launch, setLaunch] = React.useState(nsGet(NS_LS.launch, 'shell') === 'command' ? 'command' : 'shell');
  const [command, setCommand] = React.useState(nsGet(NS_LS.command, ''));
  const [attach, setAttach] = React.useState(true);
  const [showAdvanced, setShowAdvanced] = React.useState(false);
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');
  const cwdEdited = React.useRef(false);
  React.useEffect(() => {
    if (!host && HOSTS[0]) setHost(HOSTS[0].id);
  }, [HOSTS.length]);
  React.useEffect(() => {
    if (!cwdEdited.current) setCwd(defaultDir);
  }, [defaultDir]);
  React.useEffect(() => {
    nsSet(NS_LS.createDir, createDir ? '1' : '0');
  }, [createDir]);
  React.useEffect(() => {
    nsSet(NS_LS.launch, launch);
  }, [launch]);
  React.useEffect(() => {
    nsSet(NS_LS.command, command);
  }, [command]);
  const caretToEnd = e => {
    const el = e.target;
    if (el.selectionStart === 0 && el.selectionEnd === el.value.length) {
      const n = el.value.length;
      try {
        el.setSelectionRange(n, n);
      } catch (_) {}
    }
  };
  const submit = async e => {
    if (e && e.preventDefault) e.preventDefault();
    if (!host) {
      setErr('Pick a host first.');
      return;
    }
    const runCmd = launch === 'command' ? command.trim() : '';
    if (launch === 'command' && !runCmd) {
      setErr('Type a command, or switch back to Start in shell.');
      return;
    }
    setErr('');
    setBusy(true);
    try {
      const finalName = name.trim() || `session-${Math.random().toString(36).slice(2, 7)}`;
      await createSession(host, finalName, cwd.trim() || '', {
        createDir,
        command: runCmd
      });
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
  })), React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Working directory"), React.createElement("div", {
    className: "dir-row"
  }, React.createElement("input", {
    className: "input mono",
    value: cwd,
    onChange: e => {
      cwdEdited.current = true;
      setCwd(e.target.value);
    },
    onFocus: caretToEnd,
    placeholder: "~/",
    spellCheck: false
  }), React.createElement("label", {
    className: "checkbox dir-create",
    title: "Create the directory if it doesn't exist yet"
  }, React.createElement("input", {
    type: "checkbox",
    checked: createDir,
    onChange: e => setCreateDir(e.target.checked)
  }), " Create")), React.createElement("div", {
    className: "hint"
  }, "The session starts here. Change the default in Settings \u2192 Defaults.")), React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Launch"), React.createElement("div", {
    className: "slide-toggle",
    "data-state": launch
  }, React.createElement("span", {
    className: "slide-thumb",
    "aria-hidden": "true"
  }), React.createElement("button", {
    type: "button",
    className: `slide-opt ${launch === 'shell' ? 'active' : ''}`,
    onClick: () => setLaunch('shell')
  }, "Start in shell"), React.createElement("button", {
    type: "button",
    className: `slide-opt ${launch === 'command' ? 'active' : ''}`,
    onClick: () => setLaunch('command')
  }, "Command")), launch === 'command' && React.createElement("input", {
    className: "input mono",
    style: {
      marginTop: 8
    },
    value: command,
    onChange: e => setCommand(e.target.value),
    onFocus: caretToEnd,
    placeholder: "e.g. claude",
    spellCheck: false
  }), React.createElement("div", {
    className: "hint"
  }, launch === 'command' ? 'Typed into the session once it starts — the shell stays alive when the command exits.' : 'Just a shell, nothing typed for you.')), HOSTS.length === 0 && React.createElement("div", {
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