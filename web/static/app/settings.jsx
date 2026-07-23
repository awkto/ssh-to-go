// Settings

const Settings = ({ store }) => {
  const [draft, setDraft] = React.useState(store.settings || {});
  const [saving, setSaving] = React.useState(false);
  const [saveMsg, setSaveMsg] = React.useState('');
  const [curPw, setCurPw] = React.useState('');
  const [newPw, setNewPw] = React.useState('');
  const [pwMsg, setPwMsg] = React.useState('');

  React.useEffect(() => { setDraft(store.settings || {}); }, [JSON.stringify(store.settings)]);

  const set = (patch) => setDraft(d => ({ ...d, ...patch }));

  const save = async () => {
    setSaving(true); setSaveMsg('');
    try {
      const r = await fetch('/api/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(draft),
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

  const changePassword = async (e) => {
    e.preventDefault();
    setPwMsg('');
    try {
      const r = await fetch('/api/auth/password', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ current_password: curPw, new_password: newPw }),
      });
      if (!r.ok) throw new Error(await r.text());
      setPwMsg('Password updated.');
      setCurPw(''); setNewPw('');
    } catch (err) {
      setPwMsg('Failed: ' + err.message);
    }
  };

  return (
    <div>
      <div className="page-head">
        <div className="page-title-block">
          <h1>Settings</h1>
          <p>Defaults, integrations, SSH keypairs, and account preferences.</p>
        </div>
      </div>

      <div className="settings-grid">
        <div className="panel">
          <div className="panel-head"><h2>Defaults</h2><span className="muted" style={{fontSize:12}}>Applied to new sessions</span></div>
          <div className="panel-body">
            <div className="setting-row">
              <div className="setting-label">
                <h4>Default username</h4>
                <p>Used when creating new sessions without an explicit user.</p>
              </div>
              <div>
                <input className="input mono" value={draft.default_username || ''} onChange={e => set({default_username: e.target.value})} placeholder="altanc" />
              </div>
            </div>
            <div className="setting-row">
              <div className="setting-label">
                <h4>Default keypair</h4>
                <p>Which key is offered first when authenticating to new hosts.</p>
              </div>
              <div>
                <select className="select" value={draft.default_keypair || ''} onChange={e => set({default_keypair: e.target.value})}>
                  {store.keypairs.map(k => <option key={k.name} value={k.name}>{k.name} ({k.type})</option>)}
                </select>
              </div>
            </div>
            <div className="setting-row">
              <div className="setting-label">
                <h4>Multi-client window size</h4>
                <p>How tmux sizes the session when multiple clients connect.</p>
              </div>
              <div>
                <select className="select" value={draft.tmux_window_size || 'latest'} onChange={e => set({tmux_window_size: e.target.value})}>
                  <option value="latest">Latest resize</option>
                  <option value="aggressive">Aggressive (smallest)</option>
                  <option value="manual">Manual</option>
                </select>
              </div>
            </div>
            <div className="setting-row">
              <div className="setting-label">
                <h4>Terminal tab title</h4>
                <p>How the browser tab title appears for terminal sessions.</p>
              </div>
              <div>
                <select className="select" value={draft.tab_title_format || 'session'} onChange={e => set({tab_title_format: e.target.value})}>
                  <option value="session">session only</option>
                  <option value="session_at_host">session @ host</option>
                  <option value="host">host only</option>
                </select>
              </div>
            </div>
            <div className="setting-row">
              <div className="setting-label">
                <h4>Show public key on dashboard</h4>
                <p>Display your default public key for easy copy.</p>
              </div>
              <div>
                <label className="checkbox">
                  <input type="checkbox" checked={!!draft.show_pub_key} onChange={e => set({show_pub_key: e.target.checked})} /> Show on dashboard
                </label>
              </div>
            </div>
            <div className="setting-row">
              <div className="setting-label">
                <h4>New session icon</h4>
                <p>The icon &amp; color a session gets when it's created.</p>
              </div>
              <div>
                <select className="select" value={draft.session_icon_mode || 'random'} onChange={e => set({session_icon_mode: e.target.value})}>
                  <option value="random">Random each time</option>
                  <option value="fixed">Fixed icon &amp; color</option>
                </select>
              </div>
            </div>
            {(draft.session_icon_mode || 'random') === 'fixed' && (
              <div className="setting-row">
                <div className="setting-label">
                  <h4>Fixed icon &amp; color</h4>
                  <p>Applied to every new session. Click to choose.</p>
                </div>
                <div>
                  <button
                    className="sess-icon-btn"
                    title="Choose icon &amp; color"
                    onClick={e => window.showIconPicker(
                      e.currentTarget,
                      draft.session_icon_name || 'terminal',
                      (icon, color) => set({session_icon_name: icon, session_icon_color: color}),
                      draft.session_icon_color || 'default'
                    )}
                  >
                    <SessIcon kind={draft.session_icon_name || 'terminal'} color={draft.session_icon_color || 'default'} />
                  </button>
                </div>
              </div>
            )}
          </div>
          <div className="panel-head" style={{borderTop:'1px solid var(--hairline)', borderBottom:0, background:'var(--bg-elev-2)', justifyContent:'flex-end', gap:10}}>
            <span className="muted" style={{fontSize:12}}>{saveMsg}</span>
            <Button variant="ghost" size="sm" onClick={() => setDraft(store.settings || {})}>Reset</Button>
            <Button variant="primary" size="sm" onClick={save} disabled={saving}>{saving ? 'Saving…' : 'Save changes'}</Button>
          </div>
        </div>

        <McpPanel />

        <ApiTokensPanel />

        <div className="panel">
          <div className="panel-head">
            <h2>SSH keypairs</h2>
            <span className="muted" style={{fontSize:12}}>Read-only · management UI coming soon</span>
          </div>
          <div className="panel-body">
            {store.keypairs.length === 0 && <div className="muted" style={{fontSize:13}}>No keypairs yet.</div>}
            {store.keypairs.map(k => (
              <div key={k.name} className="key-card">
                <span className="icon-bg" style={{width:28, height:28, borderRadius:6, background:'var(--accent-soft)', color:'var(--accent)', display:'grid', placeItems:'center'}}>
                  <IconKey size={14}/>
                </span>
                <div>
                  <div className="row gap-2">
                    <span className="key-name">{k.name}</span>
                    {k.isDefault && <Pill variant="ok">default</Pill>}
                    <span className="muted mono" style={{fontSize:11}}>{k.type}</span>
                    {k.imported && <span className="muted mono" style={{fontSize:11}}>imported</span>}
                  </div>
                  <div className="key-fp">{k.fp}</div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="panel">
          <div className="panel-head"><h2>Account</h2></div>
          <div className="panel-body">
            <form className="setting-row" onSubmit={changePassword}>
              <div className="setting-label">
                <h4>Change password</h4>
                <p>Session tokens are reissued after a password change.</p>
              </div>
              <div style={{display:'flex', flexDirection:'column', gap:8}}>
                <input className="input" type="password" placeholder="Current password" value={curPw} onChange={e => setCurPw(e.target.value)} />
                <input className="input" type="password" placeholder="New password" value={newPw} onChange={e => setNewPw(e.target.value)} />
                <Button variant="secondary" size="sm" type="submit" style={{alignSelf:'flex-start'}}>Update password</Button>
                {pwMsg && <span className="muted" style={{fontSize:12}}>{pwMsg}</span>}
              </div>
            </form>
            <div className="setting-row">
              <div className="setting-label">
                <h4 style={{color:'var(--err)'}}>Sign out</h4>
                <p>End this browser session.</p>
              </div>
              <div>
                <Button variant="secondary" size="sm" onClick={async () => { await fetch('/api/auth/logout', {method:'POST'}); location.href = '/login'; }}>Sign out</Button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
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
    const next = { ...cfg, enabled: !cfg.enabled };
    const r = await fetch('/api/settings/mcp', { method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify(next) });
    if (r.ok) setCfg(next);
    setBusy(false);
  };
  const copyEndpoint = () => {
    const url = location.origin + '/mcp/sse';
    navigator.clipboard?.writeText(url);
  };
  return (
    <div className="panel">
      <div className="panel-head"><h2>Integrations</h2></div>
      <div className="panel-body">
        <div className="setting-row" style={{borderBottom:0, padding:0}}>
          <div className="setting-label">
            <h4 className="row gap-2">
              MCP (Model Context Protocol)
              {cfg?.enabled && <Pill variant="ok">Live</Pill>}
            </h4>
            <p>Exposes an MCP server at <span className="mono">/mcp/sse</span> for AI tool integrations (e.g. Claude Code). Uses the same API token authentication.</p>
          </div>
          <div>
            <label className="checkbox" style={{marginBottom:10}}>
              <input type="checkbox" checked={!!cfg?.enabled} onChange={toggle} disabled={busy || !cfg} />
              Enable MCP server
            </label>
            <div style={{display:'flex', gap:6}}>
              <Button variant="secondary" size="sm" icon={IconExternalLink} onClick={() => window.open('/mcpdocs', '_blank')}>View MCP docs</Button>
              <Button variant="secondary" size="sm" icon={IconCopy} onClick={copyEndpoint}>Copy endpoint</Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

const ApiTokensPanel = () => {
  const [tokens, setTokens] = React.useState(null);
  const [name, setName] = React.useState('');
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');
  const [justCreated, setJustCreated] = React.useState(null); // { name, token }
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

  React.useEffect(() => { load(); }, [load]);

  const create = async (e) => {
    e.preventDefault();
    if (!name.trim()) return;
    setBusy(true); setErr('');
    try {
      const r = await fetch('/api/auth/tokens', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: name.trim() }),
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

  const remove = async (tokenName) => {
    if (!confirm(`Revoke token "${tokenName}"? Clients using it will stop working.`)) return;
    try {
      const r = await fetch(`/api/auth/tokens/${encodeURIComponent(tokenName)}`, { method: 'DELETE' });
      if (!r.ok) throw new Error(await r.text());
      if (justCreated && justCreated.name === tokenName) setJustCreated(null);
      load();
    } catch (ex) {
      alert('revoke failed: ' + ex.message);
    }
  };

  const copy = async () => {
    if (!justCreated) return;
    try { await navigator.clipboard.writeText(justCreated.token); setCopied(true); }
    catch (ex) { alert('copy failed: ' + ex.message); }
  };

  return (
    <div className="panel">
      <div className="panel-head">
        <h2>API tokens</h2>
        <span className="muted" style={{fontSize:12}}>Bearer tokens for native clients (Android app, MCP)</span>
      </div>
      <div className="panel-body">
        {justCreated && (
          <div className="setting-row" style={{background:'var(--bg-elev-2)', padding:12, borderRadius:8, alignItems:'flex-start'}}>
            <div className="setting-label">
              <h4 style={{color:'var(--ok)'}}>New token: {justCreated.name}</h4>
              <p>Copy it now — it won't be shown again. Paste this into the Android app's "API token" field.</p>
            </div>
            <div style={{display:'flex', flexDirection:'column', gap:8, minWidth:0}}>
              <code className="mono" style={{
                padding:'8px 10px',
                background:'var(--bg)',
                border:'1px solid var(--hairline)',
                borderRadius:6,
                fontSize:12,
                wordBreak:'break-all',
                userSelect:'all'
              }}>{justCreated.token}</code>
              <div style={{display:'flex', gap:6}}>
                <Button variant="primary" size="sm" icon={IconCopy} onClick={copy}>{copied ? 'Copied!' : 'Copy'}</Button>
                <Button variant="ghost" size="sm" onClick={() => setJustCreated(null)}>Dismiss</Button>
              </div>
            </div>
          </div>
        )}

        <form className="setting-row" onSubmit={create}>
          <div className="setting-label">
            <h4>Create token</h4>
            <p>Pick a memorable name (e.g. "phone", "android"). The token itself will be generated server-side.</p>
          </div>
          <div style={{display:'flex', gap:6}}>
            <input className="input mono" placeholder="name" value={name} onChange={e => setName(e.target.value)} />
            <Button variant="primary" size="sm" type="submit" disabled={busy || !name.trim()}>{busy ? '…' : 'Create'}</Button>
          </div>
        </form>

        {err && <div style={{color:'var(--err)', fontSize:12.5, padding:'0 0 8px'}}>{err}</div>}

        <div style={{padding:'4px 0 0'}}>
          {tokens === null && <div className="muted" style={{fontSize:13}}>Loading…</div>}
          {tokens && tokens.length === 0 && <div className="muted" style={{fontSize:13}}>No tokens yet.</div>}
          {tokens && tokens.map(t => (
            <div key={t.name} className="key-card">
              <span className="icon-bg" style={{width:28, height:28, borderRadius:6, background:'var(--accent-soft)', color:'var(--accent)', display:'grid', placeItems:'center'}}>
                <IconKey size={14}/>
              </span>
              <div style={{flex:1, minWidth:0}}>
                <div className="row gap-2">
                  <span className="key-name">{t.name}</span>
                  <span className="muted mono" style={{fontSize:11}}>created {timeAgo(new Date(t.created))}</span>
                </div>
              </div>
              <Button variant="ghost" size="sm" onClick={() => remove(t.name)}>Revoke</Button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

Object.assign(window, { Settings, McpPanel, ApiTokensPanel });
