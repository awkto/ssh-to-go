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
  }, React.createElement("h4", null, "Show public key on dashboard"), React.createElement("p", null, "Display your default public key for easy copy.")), React.createElement("div", null, React.createElement("label", {
    className: "checkbox"
  }, React.createElement("input", {
    type: "checkbox",
    checked: !!draft.show_pub_key,
    onChange: e => set({
      show_pub_key: e.target.checked
    })
  }), " Show on dashboard")))), React.createElement("div", {
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
  }, saving ? 'Saving…' : 'Save changes'))), React.createElement(McpPanel, null), React.createElement("div", {
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
Object.assign(window, {
  Settings,
  McpPanel
});