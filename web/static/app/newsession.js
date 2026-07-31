const NS_LS = {
  createDir: 's2g:newsession:create-dir',
  launch: 's2g:newsession:launch',
  command: 's2g:newsession:command',
  after: 's2g:newsession:after'
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
const nsAutoName = throwaway => (throwaway ? 'tmp-' : 'session-') + Math.random().toString(36).slice(2, 6);
const NS_VARS = ['name', 'date'];
const NS_VARS_HINT = '$name = this session’s name, $date = today as YYYY-MM-DD. ' + '${name} works too. Any other $VAR is left for the shell.';
const nsIdent = c => /[A-Za-z0-9_]/.test(c);
const nsSanitize = s => s.trim().replace(/[ \t]+/g, '-');
const nsValue = (v, name) => name === 'name' ? v : new Date().toLocaleDateString('en-CA');
const nsMatch = s => {
  if (s.length < 2 || s[0] !== '$') return null;
  if (s[1] === '{') {
    const end = s.indexOf('}');
    if (end < 0) return null;
    const name = s.slice(2, end);
    return NS_VARS.includes(name) ? {
      name,
      len: end + 1
    } : null;
  }
  for (const name of NS_VARS) {
    if (!s.startsWith(name, 1)) continue;
    const rest = s.slice(1 + name.length);
    if (rest && nsIdent(rest[0])) return null;
    return {
      name,
      len: 1 + name.length
    };
  }
  return null;
};
const nsExpand = (s, sessionName) => {
  if (!s || s.indexOf('$') < 0) return s;
  let out = '';
  for (let i = 0; i < s.length;) {
    if (s[i] === '\\' && s[i + 1] === '$') {
      const esc = nsMatch(s.slice(i + 1));
      if (esc) {
        out += s.substr(i + 1, esc.len);
        i += 1 + esc.len;
        continue;
      }
    }
    if (s[i] === '$') {
      const m = nsMatch(s.slice(i));
      if (m) {
        out += nsValue(sessionName, m.name);
        i += m.len;
        continue;
      }
    }
    out += s[i];
    i++;
  }
  return out;
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
  const [after, setAfter] = React.useState(nsGet(NS_LS.after, 'attach') === 'handoff' ? 'handoff' : 'attach');
  const [throwaway, setThrowaway] = React.useState(false);
  const [incognito, setIncognito] = React.useState(false);
  const [autoName, setAutoName] = React.useState(() => nsAutoName(false));
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');
  const [copied, setCopied] = React.useState(false);
  const [sshCmd, setSshCmd] = React.useState('');
  const [hostMenu, setHostMenu] = React.useState(false);
  const cwdEdited = React.useRef(false);
  const copyTimer = React.useRef(null);
  const hostObj = HOSTS.find(h => h.id === host) || HOSTS[0] || null;
  const effName = name.trim() || autoName;
  const varName = nsSanitize(effName);
  const isCommand = launch === 'command';
  const isHandoff = after === 'handoff';
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
  React.useEffect(() => {
    nsSet(NS_LS.after, after);
  }, [after]);
  React.useEffect(() => () => clearTimeout(copyTimer.current), []);
  React.useEffect(() => {
    setAutoName(nsAutoName(throwaway));
  }, [throwaway]);
  const fallbackSsh = React.useMemo(() => {
    if (!hostObj) return '';
    const [addr, port] = String(hostObj.fqdn || '').split(':');
    const p = port && port !== '22' ? `-p ${port} ` : '';
    return `ssh -t ${p}${hostObj.user}@${addr} 'tmux set-option -s escape-time 200 \\; set-option -t "${effName}" mouse on 2>/dev/null; exec tmux attach-session -t "${effName}"'`;
  }, [hostObj && hostObj.fqdn, hostObj && hostObj.user, effName]);
  React.useEffect(() => {
    if (!host) return;
    let cancelled = false;
    setSshCmd('');
    const t = setTimeout(() => {
      getHandoff(host, effName).then(cmd => {
        if (!cancelled) setSshCmd(cmd);
      }).catch(() => {});
    }, 250);
    return () => {
      cancelled = true;
      clearTimeout(t);
    };
  }, [host, effName]);
  const shownSsh = sshCmd || fallbackSsh;
  const caretToEnd = e => {
    const el = e.target;
    if (el.selectionStart === 0 && el.selectionEnd === el.value.length) {
      const n = el.value.length;
      try {
        el.setSelectionRange(n, n);
      } catch (_) {}
    }
  };
  const copySsh = async () => {
    try {
      await navigator.clipboard.writeText(shownSsh);
    } catch (_) {}
    setCopied(true);
    clearTimeout(copyTimer.current);
    copyTimer.current = setTimeout(() => setCopied(false), 1600);
  };
  const recents = (store.recentCommands || []).slice(0, 4);
  const pickRecent = cmd => {
    setCommand(cmd);
    setLaunch('command');
  };
  const forgetRecent = async cmd => {
    try {
      await forgetRecentCommand(cmd);
    } catch (ex) {
      setErr(ex.message || 'could not forget that command');
    }
  };
  const shownCwd = nsExpand(cwd.trim(), varName);
  const shownCommand = nsExpand(command.trim(), varName);
  const summary = (() => {
    const head = isCommand ? `Runs ${shownCommand || '…'}` : 'Opens a shell';
    const where = `in ${shownCwd || '~'} on ${hostObj ? hostObj.fqdn : 'this host'}`;
    const tail = [throwaway ? 'ends when you detach' : 'keeps running until killed'];
    if (incognito) tail.push('untracked');
    tail.push(isHandoff ? 'hands off to your terminal' : 'attaches in a web tab');
    return `${head} ${where} · ${tail.join(', ')}.`;
  })();
  const submit = async e => {
    if (e && e.preventDefault) e.preventDefault();
    if (!host) {
      setErr('Pick a host first.');
      return;
    }
    const runCmd = isCommand ? command.trim() : '';
    if (isCommand && !runCmd) {
      setErr('Type a command, or switch back to shell.');
      return;
    }
    setErr('');
    setBusy(true);
    try {
      await createSession(host, effName, cwd.trim() || '', {
        createDir,
        command: runCmd,
        throwaway,
        incognito
      });
      if (isHandoff) {
        try {
          await navigator.clipboard.writeText(shownSsh);
        } catch (_) {}
      }
      onClose();
      if (!isHandoff) openTerminal(host, effName);
    } catch (ex) {
      setErr(ex.message || 'failed');
    } finally {
      setBusy(false);
    }
  };
  const shortHost = (() => {
    if (!hostObj) return 'host';
    const addr = String(hostObj.fqdn).split(':')[0];
    return /^[\d.]+$/.test(addr) ? addr : addr.split('.')[0];
  })();
  const seg = active => `ns-seg${active ? ' active' : ''}`;
  return React.createElement("div", {
    className: "modal-backdrop",
    onClick: onClose
  }, React.createElement("div", {
    className: "modal",
    onClick: e => {
      e.stopPropagation();
      setHostMenu(false);
    }
  }, React.createElement("form", {
    onSubmit: submit
  }, React.createElement("div", {
    className: "ns-head"
  }, React.createElement("h3", null, "New session"), React.createElement("div", {
    className: "ns-hostpick"
  }, React.createElement("button", {
    type: "button",
    className: "ns-hostchip",
    disabled: HOSTS.length <= 1,
    onClick: e => {
      e.stopPropagation();
      setHostMenu(v => !v);
    },
    title: HOSTS.length > 1 ? 'Choose a host' : undefined
  }, React.createElement(StatusDot, {
    status: hostObj && hostObj.status === 'online' ? 'active' : 'offline'
  }), React.createElement("span", {
    className: "mono"
  }, hostObj ? hostObj.fqdn : 'no hosts'), HOSTS.length > 1 && React.createElement(IconChevronDown, {
    size: 12
  })), hostMenu && React.createElement("div", {
    className: "ns-hostmenu",
    onClick: e => e.stopPropagation()
  }, HOSTS.map(h => React.createElement("button", {
    type: "button",
    key: h.id,
    className: `ns-hostopt${h.id === host ? ' active' : ''}`,
    onClick: () => {
      setHost(h.id);
      setHostMenu(false);
    }
  }, React.createElement(StatusDot, {
    status: h.status === 'online' ? 'active' : 'offline'
  }), React.createElement("span", {
    className: "mono",
    style: {
      flex: 1
    }
  }, h.fqdn), React.createElement("span", {
    className: "ns-hostopt-sub"
  }, h.sessions, " sess"))))), React.createElement("div", {
    style: {
      flex: 1
    }
  }), React.createElement("button", {
    type: "button",
    className: "ns-close",
    onClick: onClose,
    "aria-label": "Close"
  }, React.createElement(IconClose, {
    size: 15
  }))), React.createElement("div", {
    className: "ns-body"
  }, React.createElement("div", {
    className: "ns-grid"
  }, React.createElement("div", {
    style: {
      minWidth: 0
    }
  }, React.createElement("label", {
    className: "ns-label",
    htmlFor: "ns-name"
  }, "Name"), React.createElement("input", {
    id: "ns-name",
    className: "ns-input mono",
    placeholder: autoName,
    value: name,
    onChange: e => setName(e.target.value),
    spellCheck: false,
    autoFocus: true
  })), React.createElement("div", {
    style: {
      minWidth: 0
    }
  }, React.createElement("label", {
    className: "ns-label",
    htmlFor: "ns-dir"
  }, "Directory"), React.createElement("div", {
    className: "ns-fieldbox"
  }, React.createElement("input", {
    id: "ns-dir",
    className: "ns-bare mono",
    value: cwd,
    onChange: e => {
      cwdEdited.current = true;
      setCwd(e.target.value);
    },
    onFocus: caretToEnd,
    placeholder: "~/",
    title: NS_VARS_HINT,
    spellCheck: false
  }), React.createElement("button", {
    type: "button",
    className: `ns-mkdir${createDir ? ' on' : ''}`,
    onClick: () => setCreateDir(v => !v),
    "aria-pressed": createDir,
    title: "Create the directory if it doesn't exist yet"
  }, "mkdir")))), React.createElement("div", null, React.createElement("div", {
    className: "ns-labelline"
  }, React.createElement("label", {
    className: "ns-label",
    style: {
      marginRight: 'auto'
    }
  }, "Start in"), recents.map(rc => React.createElement("span", {
    className: "ns-chip",
    key: rc.command,
    title: `${rc.command} — used ${rc.count}×`
  }, React.createElement("button", {
    type: "button",
    className: "ns-chip-pick mono",
    onClick: () => pickRecent(rc.command)
  }, rc.command), React.createElement("button", {
    type: "button",
    className: "ns-chip-forget",
    onClick: () => forgetRecent(rc.command),
    "aria-label": `Forget ${rc.command}`,
    title: `Forget ${rc.command}`
  }, "\xD7")))), React.createElement("div", {
    className: "ns-control"
  }, React.createElement("div", {
    className: "ns-segs"
  }, React.createElement("button", {
    type: "button",
    className: seg(!isCommand),
    onClick: () => setLaunch('shell')
  }, "shell"), React.createElement("button", {
    type: "button",
    className: seg(isCommand),
    onClick: () => setLaunch('command')
  }, "command")), React.createElement("div", {
    className: "ns-slot"
  }, isCommand ? React.createElement(React.Fragment, null, React.createElement("span", {
    className: "ns-dollar mono"
  }, "$"), React.createElement("input", {
    className: "ns-bare mono",
    value: command,
    onChange: e => setCommand(e.target.value),
    onFocus: caretToEnd,
    placeholder: "claude",
    title: NS_VARS_HINT,
    spellCheck: false
  })) : React.createElement("span", {
    className: "ns-ghost mono"
  }, hostObj ? hostObj.user : 'you', "@", shortHost, ":", shownCwd || '~', "$", React.createElement("i", {
    className: "ns-caret"
  }))))), React.createElement("div", null, React.createElement("div", {
    className: "ns-labelline"
  }, React.createElement("label", {
    className: "ns-label",
    style: {
      marginRight: 'auto'
    }
  }, "Then"), React.createElement("button", {
    type: "button",
    className: `ns-flag${throwaway ? ' on' : ''}`,
    "aria-pressed": throwaway,
    onClick: () => setThrowaway(v => !v),
    title: "Removed as soon as you leave it"
  }, React.createElement(IconClock, {
    size: 13
  }), "throwaway"), React.createElement("button", {
    type: "button",
    className: `ns-flag${incognito ? ' on' : ''}`,
    "aria-pressed": incognito,
    onClick: () => setIncognito(v => !v),
    title: "Not tracked in the app"
  }, React.createElement(IconMoon, {
    size: 13
  }), "incognito")), React.createElement("div", {
    className: "ns-control"
  }, React.createElement("div", {
    className: "ns-segs"
  }, React.createElement("button", {
    type: "button",
    className: seg(!isHandoff),
    onClick: () => setAfter('attach')
  }, "attach here"), React.createElement("button", {
    type: "button",
    className: seg(isHandoff),
    onClick: () => setAfter('handoff')
  }, "hand off")), React.createElement("div", {
    className: "ns-slot"
  }, React.createElement("span", {
    className: "ns-dest"
  }, isHandoff ? React.createElement(IconTerminal, {
    size: 13
  }) : React.createElement(IconExternalLink, {
    size: 13
  }), isHandoff ? 'your own terminal' : 'new web-terminal tab'))), React.createElement("div", {
    className: `ns-ssh${isHandoff ? ' lit' : ''}`
  }, React.createElement("span", {
    className: "ns-ssh-cmd mono"
  }, shownSsh), React.createElement("button", {
    type: "button",
    className: "ns-copy mono",
    onClick: copySsh
  }, React.createElement(IconCopy, {
    size: 12
  }), copied ? 'copied' : 'copy')))), React.createElement("div", {
    className: "ns-foot"
  }, React.createElement("div", {
    className: `ns-summary${err ? ' err' : ''}`,
    title: err || summary
  }, React.createElement("span", null, err || summary)), React.createElement("button", {
    type: "button",
    className: "ns-cancel",
    onClick: onClose
  }, "Cancel"), React.createElement("button", {
    type: "submit",
    className: "ns-primary",
    disabled: busy || !host
  }, isHandoff ? React.createElement(IconCopy, {
    size: 14
  }) : React.createElement(IconPlay, {
    size: 14
  }), busy ? 'Creating…' : isHandoff ? 'Create & copy ssh' : 'Create & attach')))));
};
Object.assign(window, {
  NewSession
});
