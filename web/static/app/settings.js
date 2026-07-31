const Settings = ({
  store
}) => {
  const [draft, setDraft] = React.useState(store.settings || {});
  const [saving, setSaving] = React.useState(false);
  const [saveMsg, setSaveMsg] = React.useState('');
  const [curPw, setCurPw] = React.useState('');
  const [newPw, setNewPw] = React.useState('');
  const [pwMsg, setPwMsg] = React.useState('');
  React.useEffect(() => {
    setDraft(store.settings || {});
  }, [JSON.stringify(store.settings)]);
  const set = patch => setDraft(d => ({
    ...d,
    ...patch
  }));
  const save = async () => {
    setSaving(true);
    setSaveMsg('');
    try {
      const r = await fetch('/api/settings', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(draft)
      });
      if (!r.ok) throw new Error(await r.text());
      await store.refresh();
      setSaveMsg('Saved.');
    } catch (e) {
      setSaveMsg('Save failed: ' + e.message);
    } finally {
      setSaving(false);
      setTimeout(() => setSaveMsg(''), 3000);
    }
  };
  const changePassword = async e => {
    e.preventDefault();
    setPwMsg('');
    try {
      const r = await fetch('/api/auth/password', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          current_password: curPw,
          new_password: newPw
        })
      });
      if (!r.ok) throw new Error(await r.text());
      setPwMsg('Password updated.');
      setCurPw('');
      setNewPw('');
    } catch (err) {
      setPwMsg('Failed: ' + err.message);
    }
  };
  return React.createElement("div", null, React.createElement("div", {
    className: "page-head"
  }, React.createElement("div", {
    className: "page-title-block"
  }, React.createElement("h1", null, "Settings"), React.createElement("p", null, "Defaults, integrations, SSH keypairs, and account preferences."))), React.createElement("div", {
    className: "settings-grid"
  }, React.createElement("div", {
    className: "panel"
  }, React.createElement("div", {
    className: "panel-head"
  }, React.createElement("h2", null, "Defaults"), React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 12
    }
  }, "Applied to new sessions")), React.createElement("div", {
    className: "panel-body"
  }, React.createElement("div", {
    className: "setting-row"
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", null, "Default username"), React.createElement("p", null, "Used when creating new sessions without an explicit user.")), React.createElement("div", null, React.createElement("input", {
    className: "input mono",
    value: draft.default_username || '',
    onChange: e => set({
      default_username: e.target.value
    }),
    placeholder: "altanc"
  }))), React.createElement("div", {
    className: "setting-row"
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", null, "New session directory"), React.createElement("p", null, "Prefills the working directory on the New Session form. Leave empty for ~/sessions/. Set it to ", React.createElement("code", null, "~/sessions/$name"), " to give every session a directory of its own; ", React.createElement("code", null, "$date"), " is today as YYYY-MM-DD.")), React.createElement("div", null, React.createElement("input", {
    className: "input mono",
    value: draft.new_session_dir || '',
    onChange: e => set({
      new_session_dir: e.target.value
    }),
    placeholder: "~/sessions/"
  }))), React.createElement("div", {
    className: "setting-row"
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", null, "Recent commands"), React.createElement("p", null, "Commands sessions were started with, offered as chips on the New Session form. Removing one applies immediately \u2014 no need to Save.")), React.createElement("div", null, (store.recentCommands || []).length === 0 ? React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 12.5
    }
  }, "Nothing remembered yet. Start a session with a command and it shows up here.") : React.createElement("div", {
    className: "rc-list"
  }, store.recentCommands.map(rc => React.createElement("div", {
    className: "rc-item",
    key: rc.command
  }, React.createElement("span", {
    className: "rc-cmd mono",
    title: rc.command
  }, rc.command), React.createElement("span", {
    className: "rc-count mono"
  }, rc.count, "\xD7"), React.createElement("button", {
    type: "button",
    className: "rc-forget",
    title: `Forget ${rc.command}`,
    "aria-label": `Forget ${rc.command}`,
    onClick: () => forgetRecentCommand(rc.command).catch(() => {})
  }, React.createElement(IconClose, {
    size: 12
  })))), React.createElement("button", {
    type: "button",
    className: "btn btn-ghost btn-sm rc-clear",
    onClick: () => forgetRecentCommand().catch(() => {})
  }, "Clear all")))), React.createElement("div", {
    className: "setting-row"
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", null, "Default keypair"), React.createElement("p", null, "Which key is offered first when authenticating to new hosts.")), React.createElement("div", null, React.createElement("select", {
    className: "select",
    value: draft.default_keypair || '',
    onChange: e => set({
      default_keypair: e.target.value
    })
  }, store.keypairs.map(k => React.createElement("option", {
    key: k.name,
    value: k.name
  }, k.name, " (", k.type, ")"))))), React.createElement("div", {
    className: "setting-row"
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", null, "Multi-client window size"), React.createElement("p", null, "How tmux sizes the session when multiple clients connect.")), React.createElement("div", null, React.createElement("select", {
    className: "select",
    value: draft.tmux_window_size || 'latest',
    onChange: e => set({
      tmux_window_size: e.target.value
    })
  }, React.createElement("option", {
    value: "latest"
  }, "Latest resize"), React.createElement("option", {
    value: "aggressive"
  }, "Aggressive (smallest)"), React.createElement("option", {
    value: "manual"
  }, "Manual")))), React.createElement("div", {
    className: "setting-row"
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", null, "Terminal tab title"), React.createElement("p", null, "How the browser tab title appears for terminal sessions.")), React.createElement("div", null, React.createElement("select", {
    className: "select",
    value: draft.tab_title_format || 'session',
    onChange: e => set({
      tab_title_format: e.target.value
    })
  }, React.createElement("option", {
    value: "session"
  }, "session only"), React.createElement("option", {
    value: "session_at_host"
  }, "session @ host"), React.createElement("option", {
    value: "host"
  }, "host only")))), React.createElement("div", {
    className: "setting-row"
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", null, "Auto-sleep idle sessions"), React.createElement("p", null, "Offload a session once it has had no client attached and nothing running for this long. The tmux session is stopped to free memory; it stays resumable with its working directory and launch command. Sessions marked keep-awake are never slept.")), React.createElement("div", null, React.createElement("select", {
    className: "select",
    value: String(draft.idle_offload_hours || 0),
    onChange: e => set({
      idle_offload_hours: parseInt(e.target.value, 10)
    })
  }, React.createElement("option", {
    value: "0"
  }, "Off"), React.createElement("option", {
    value: "24"
  }, "After 1 day"), React.createElement("option", {
    value: "48"
  }, "After 2 days"), React.createElement("option", {
    value: "168"
  }, "After 7 days"), React.createElement("option", {
    value: "720"
  }, "After 30 days")))), React.createElement("div", {
    className: "setting-row"
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", null, "Show public key on dashboard"), React.createElement("p", null, "Display your default public key for easy copy.")), React.createElement("div", null, React.createElement("label", {
    className: "checkbox"
  }, React.createElement("input", {
    type: "checkbox",
    checked: !!draft.show_pub_key,
    onChange: e => set({
      show_pub_key: e.target.checked
    })
  }), " Show on dashboard"))), React.createElement("div", {
    className: "setting-row"
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", null, "New session icon"), React.createElement("p", null, "The icon & color a session gets when it's created.")), React.createElement("div", null, React.createElement("select", {
    className: "select",
    value: draft.session_icon_mode || 'random',
    onChange: e => set({
      session_icon_mode: e.target.value
    })
  }, React.createElement("option", {
    value: "random"
  }, "Random each time"), React.createElement("option", {
    value: "fixed"
  }, "Fixed icon & color")))), (draft.session_icon_mode || 'random') === 'fixed' && React.createElement("div", {
    className: "setting-row"
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", null, "Fixed icon & color"), React.createElement("p", null, "Applied to every new session. Click to choose.")), React.createElement("div", null, React.createElement("button", {
    className: "sess-icon-btn",
    title: "Choose icon & color",
    onClick: e => window.showIconPicker(e.currentTarget, draft.session_icon_name || 'terminal', (icon, color) => set({
      session_icon_name: icon,
      session_icon_color: color
    }), draft.session_icon_color || 'default')
  }, React.createElement(SessIcon, {
    kind: draft.session_icon_name || 'terminal',
    color: draft.session_icon_color || 'default'
  }))))), React.createElement("div", {
    className: "panel-head",
    style: {
      borderTop: '1px solid var(--hairline)',
      borderBottom: 0,
      background: 'var(--bg-elev-2)',
      justifyContent: 'flex-end',
      gap: 10
    }
  }, React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 12
    }
  }, saveMsg), React.createElement(Button, {
    variant: "ghost",
    size: "sm",
    onClick: () => setDraft(store.settings || {})
  }, "Reset"), React.createElement(Button, {
    variant: "primary",
    size: "sm",
    onClick: save,
    disabled: saving
  }, saving ? 'Saving…' : 'Save changes'))), React.createElement(McpPanel, null), React.createElement(ApiTokensPanel, null), React.createElement("div", {
    className: "panel"
  }, React.createElement("div", {
    className: "panel-head"
  }, React.createElement("h2", null, "SSH keypairs"), React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 12
    }
  }, "Read-only \xB7 management UI coming soon")), React.createElement("div", {
    className: "panel-body"
  }, store.keypairs.length === 0 && React.createElement("div", {
    className: "muted",
    style: {
      fontSize: 13
    }
  }, "No keypairs yet."), store.keypairs.map(k => React.createElement("div", {
    key: k.name,
    className: "key-card"
  }, React.createElement("span", {
    className: "icon-bg",
    style: {
      width: 28,
      height: 28,
      borderRadius: 6,
      background: 'var(--accent-soft)',
      color: 'var(--accent)',
      display: 'grid',
      placeItems: 'center'
    }
  }, React.createElement(IconKey, {
    size: 14
  })), React.createElement("div", null, React.createElement("div", {
    className: "row gap-2"
  }, React.createElement("span", {
    className: "key-name"
  }, k.name), k.isDefault && React.createElement(Pill, {
    variant: "ok"
  }, "default"), React.createElement("span", {
    className: "muted mono",
    style: {
      fontSize: 11
    }
  }, k.type), k.imported && React.createElement("span", {
    className: "muted mono",
    style: {
      fontSize: 11
    }
  }, "imported")), React.createElement("div", {
    className: "key-fp"
  }, k.fp)))))), React.createElement("div", {
    className: "panel"
  }, React.createElement("div", {
    className: "panel-head"
  }, React.createElement("h2", null, "Account")), React.createElement("div", {
    className: "panel-body"
  }, React.createElement("form", {
    className: "setting-row",
    onSubmit: changePassword
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", null, "Change password"), React.createElement("p", null, "Session tokens are reissued after a password change.")), React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 8
    }
  }, React.createElement("input", {
    className: "input",
    type: "password",
    placeholder: "Current password",
    value: curPw,
    onChange: e => setCurPw(e.target.value)
  }), React.createElement("input", {
    className: "input",
    type: "password",
    placeholder: "New password",
    value: newPw,
    onChange: e => setNewPw(e.target.value)
  }), React.createElement(Button, {
    variant: "secondary",
    size: "sm",
    type: "submit",
    style: {
      alignSelf: 'flex-start'
    }
  }, "Update password"), pwMsg && React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 12
    }
  }, pwMsg))), React.createElement("div", {
    className: "setting-row"
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", {
    style: {
      color: 'var(--err)'
    }
  }, "Sign out"), React.createElement("p", null, "End this browser session.")), React.createElement("div", null, React.createElement(Button, {
    variant: "secondary",
    size: "sm",
    onClick: async () => {
      await fetch('/api/auth/logout', {
        method: 'POST'
      });
      location.href = '/login';
    }
  }, "Sign out")))))));
};
const McpPanel = () => {
  const [cfg, setCfg] = React.useState(null);
  const [busy, setBusy] = React.useState(false);
  React.useEffect(() => {
    fetch('/api/settings/mcp').then(r => r.ok ? r.json() : null).then(setCfg).catch(() => setCfg(null));
  }, []);
  const toggle = async () => {
    if (!cfg) return;
    setBusy(true);
    const next = {
      ...cfg,
      enabled: !cfg.enabled
    };
    const r = await fetch('/api/settings/mcp', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(next)
    });
    if (r.ok) setCfg(next);
    setBusy(false);
  };
  const copyEndpoint = () => {
    const url = location.origin + '/mcp/sse';
    navigator.clipboard?.writeText(url);
  };
  return React.createElement("div", {
    className: "panel"
  }, React.createElement("div", {
    className: "panel-head"
  }, React.createElement("h2", null, "Integrations")), React.createElement("div", {
    className: "panel-body"
  }, React.createElement("div", {
    className: "setting-row",
    style: {
      borderBottom: 0,
      padding: 0
    }
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", {
    className: "row gap-2"
  }, "MCP (Model Context Protocol)", cfg?.enabled && React.createElement(Pill, {
    variant: "ok"
  }, "Live")), React.createElement("p", null, "Exposes an MCP server at ", React.createElement("span", {
    className: "mono"
  }, "/mcp/sse"), " for AI tool integrations (e.g. Claude Code). Uses the same API token authentication.")), React.createElement("div", null, React.createElement("label", {
    className: "checkbox",
    style: {
      marginBottom: 10
    }
  }, React.createElement("input", {
    type: "checkbox",
    checked: !!cfg?.enabled,
    onChange: toggle,
    disabled: busy || !cfg
  }), "Enable MCP server"), React.createElement("div", {
    style: {
      display: 'flex',
      gap: 6
    }
  }, React.createElement(Button, {
    variant: "secondary",
    size: "sm",
    icon: IconExternalLink,
    onClick: () => window.open('/mcpdocs', '_blank')
  }, "View MCP docs"), React.createElement(Button, {
    variant: "secondary",
    size: "sm",
    icon: IconCopy,
    onClick: copyEndpoint
  }, "Copy endpoint"))))));
};
const ApiTokensPanel = () => {
  const [tokens, setTokens] = React.useState(null);
  const [name, setName] = React.useState('');
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');
  const [justCreated, setJustCreated] = React.useState(null);
  const [copied, setCopied] = React.useState(false);
  const load = React.useCallback(async () => {
    try {
      const r = await fetch('/api/auth/tokens');
      if (!r.ok) throw new Error(await r.text());
      setTokens(await r.json());
    } catch (e) {
      setErr(e.message || 'load failed');
    }
  }, []);
  React.useEffect(() => {
    load();
  }, [load]);
  const create = async e => {
    e.preventDefault();
    if (!name.trim()) return;
    setBusy(true);
    setErr('');
    try {
      const r = await fetch('/api/auth/tokens', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          name: name.trim()
        })
      });
      if (!r.ok) throw new Error(await r.text());
      const body = await r.json();
      setJustCreated(body);
      setName('');
      setCopied(false);
      load();
    } catch (ex) {
      setErr(ex.message || 'create failed');
    } finally {
      setBusy(false);
    }
  };
  const remove = async tokenName => {
    if (!confirm(`Revoke token "${tokenName}"? Clients using it will stop working.`)) return;
    try {
      const r = await fetch(`/api/auth/tokens/${encodeURIComponent(tokenName)}`, {
        method: 'DELETE'
      });
      if (!r.ok) throw new Error(await r.text());
      if (justCreated && justCreated.name === tokenName) setJustCreated(null);
      load();
    } catch (ex) {
      alert('revoke failed: ' + ex.message);
    }
  };
  const copy = async () => {
    if (!justCreated) return;
    try {
      await navigator.clipboard.writeText(justCreated.token);
      setCopied(true);
    } catch (ex) {
      alert('copy failed: ' + ex.message);
    }
  };
  return React.createElement("div", {
    className: "panel"
  }, React.createElement("div", {
    className: "panel-head"
  }, React.createElement("h2", null, "API tokens"), React.createElement("span", {
    className: "muted",
    style: {
      fontSize: 12
    }
  }, "Bearer tokens for native clients (Android app, MCP)")), React.createElement("div", {
    className: "panel-body"
  }, justCreated && React.createElement("div", {
    className: "setting-row",
    style: {
      background: 'var(--bg-elev-2)',
      padding: 12,
      borderRadius: 8,
      alignItems: 'flex-start'
    }
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", {
    style: {
      color: 'var(--ok)'
    }
  }, "New token: ", justCreated.name), React.createElement("p", null, "Copy it now \u2014 it won't be shown again. Paste this into the Android app's \"API token\" field.")), React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 8,
      minWidth: 0
    }
  }, React.createElement("code", {
    className: "mono",
    style: {
      padding: '8px 10px',
      background: 'var(--bg)',
      border: '1px solid var(--hairline)',
      borderRadius: 6,
      fontSize: 12,
      wordBreak: 'break-all',
      userSelect: 'all'
    }
  }, justCreated.token), React.createElement("div", {
    style: {
      display: 'flex',
      gap: 6
    }
  }, React.createElement(Button, {
    variant: "primary",
    size: "sm",
    icon: IconCopy,
    onClick: copy
  }, copied ? 'Copied!' : 'Copy'), React.createElement(Button, {
    variant: "ghost",
    size: "sm",
    onClick: () => setJustCreated(null)
  }, "Dismiss")))), React.createElement("form", {
    className: "setting-row",
    onSubmit: create
  }, React.createElement("div", {
    className: "setting-label"
  }, React.createElement("h4", null, "Create token"), React.createElement("p", null, "Pick a memorable name (e.g. \"phone\", \"android\"). The token itself will be generated server-side.")), React.createElement("div", {
    style: {
      display: 'flex',
      gap: 6
    }
  }, React.createElement("input", {
    className: "input mono",
    placeholder: "name",
    value: name,
    onChange: e => setName(e.target.value)
  }), React.createElement(Button, {
    variant: "primary",
    size: "sm",
    type: "submit",
    disabled: busy || !name.trim()
  }, busy ? '…' : 'Create'))), err && React.createElement("div", {
    style: {
      color: 'var(--err)',
      fontSize: 12.5,
      padding: '0 0 8px'
    }
  }, err), React.createElement("div", {
    style: {
      padding: '4px 0 0'
    }
  }, tokens === null && React.createElement("div", {
    className: "muted",
    style: {
      fontSize: 13
    }
  }, "Loading\u2026"), tokens && tokens.length === 0 && React.createElement("div", {
    className: "muted",
    style: {
      fontSize: 13
    }
  }, "No tokens yet."), tokens && tokens.map(t => React.createElement("div", {
    key: t.name,
    className: "key-card"
  }, React.createElement("span", {
    className: "icon-bg",
    style: {
      width: 28,
      height: 28,
      borderRadius: 6,
      background: 'var(--accent-soft)',
      color: 'var(--accent)',
      display: 'grid',
      placeItems: 'center'
    }
  }, React.createElement(IconKey, {
    size: 14
  })), React.createElement("div", {
    style: {
      flex: 1,
      minWidth: 0
    }
  }, React.createElement("div", {
    className: "row gap-2"
  }, React.createElement("span", {
    className: "key-name"
  }, t.name), React.createElement("span", {
    className: "muted mono",
    style: {
      fontSize: 11
    }
  }, "created ", timeAgo(new Date(t.created))))), React.createElement(Button, {
    variant: "ghost",
    size: "sm",
    onClick: () => remove(t.name)
  }, "Revoke"))))));
};
Object.assign(window, {
  Settings,
  McpPanel,
  ApiTokensPanel
});
