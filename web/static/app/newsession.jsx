// New Session modal — single-form (no stepper). Required: a host (auto-selected
// when only one exists). Name auto-generates if left empty. Everything else is
// optional. Minimum path is one click of Create; Enter also submits.

// Choices the user is likely to repeat are remembered locally rather than in
// server settings: they're per-browser habits, not deployment config. The
// working directory is the exception — it seeds from the server setting so it
// is the same on every device.
const NS_LS = {
  createDir: 's2g:newsession:create-dir',
  launch: 's2g:newsession:launch',
  command: 's2g:newsession:command',
};
const nsGet = (k, fallback) => {
  try { const v = localStorage.getItem(k); return v === null ? fallback : v; } catch (_) { return fallback; }
};
const nsSet = (k, v) => { try { localStorage.setItem(k, v); } catch (_) {} };

const NewSession = ({ store, onClose }) => {
  const HOSTS = store.hosts;
  const defaultDir = (store.settings && store.settings.new_session_dir) || '~/sessions/';
  const [host, setHost] = React.useState(HOSTS[0] ? HOSTS[0].id : '');
  const [name, setName] = React.useState('');
  const [cwd, setCwd] = React.useState(defaultDir);
  const [createDir, setCreateDir] = React.useState(nsGet(NS_LS.createDir, '1') === '1');
  const [launch, setLaunch] = React.useState(nsGet(NS_LS.launch, 'shell') === 'command' ? 'command' : 'shell');
  const [command, setCommand] = React.useState(nsGet(NS_LS.command, ''));
  // Deliberately NOT persisted, unlike the fields above: both are
  // consequential enough that they should be a per-create decision rather
  // than something yesterday's session left switched on.
  const [throwaway, setThrowaway] = React.useState(false);
  const [incognito, setIncognito] = React.useState(false);
  const [attach, setAttach] = React.useState(true);
  const [showAdvanced, setShowAdvanced] = React.useState(false);
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');
  const cwdEdited = React.useRef(false);

  React.useEffect(() => { if (!host && HOSTS[0]) setHost(HOSTS[0].id); }, [HOSTS.length]);

  // Settings can land after the modal mounts (first paint of a cold load).
  // Adopt the configured default then — but never overwrite typing.
  React.useEffect(() => { if (!cwdEdited.current) setCwd(defaultDir); }, [defaultDir]);

  React.useEffect(() => { nsSet(NS_LS.createDir, createDir ? '1' : '0'); }, [createDir]);
  React.useEffect(() => { nsSet(NS_LS.launch, launch); }, [launch]);
  React.useEffect(() => { nsSet(NS_LS.command, command); }, [command]);

  // Focusing the path field must never wipe it — the point of prefilling
  // "~/sessions/" is that you type the project name onto the end. A click
  // already lands the caret where you click; this only collapses the
  // select-all that tab/programmatic focus produces.
  const caretToEnd = (e) => {
    const el = e.target;
    if (el.selectionStart === 0 && el.selectionEnd === el.value.length) {
      const n = el.value.length;
      try { el.setSelectionRange(n, n); } catch (_) {}
    }
  };

  const submit = async (e) => {
    if (e && e.preventDefault) e.preventDefault();
    if (!host) { setErr('Pick a host first.'); return; }
    const runCmd = launch === 'command' ? command.trim() : '';
    if (launch === 'command' && !runCmd) { setErr('Type a command, or switch back to Start in shell.'); return; }
    setErr(''); setBusy(true);
    try {
      const finalName = name.trim() || `session-${Math.random().toString(36).slice(2, 7)}`;
      await createSession(host, finalName, cwd.trim() || '', { createDir, command: runCmd, throwaway, incognito });
      onClose();
      if (attach) openTerminal(host, finalName);
    } catch (ex) {
      setErr(ex.message || 'failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e)=>e.stopPropagation()}>
        <form onSubmit={submit}>
          <div className="modal-head">
            <div style={{display:'flex', justifyContent:'space-between', alignItems:'center', gap:12}}>
              <div>
                <h3>New session</h3>
                <p>Press Create to spin up a detached tmux session. All fields are optional.</p>
              </div>
              <button type="button" className="icon-btn" onClick={onClose}><IconClose size={15}/></button>
            </div>
          </div>

          <div className="modal-body">
            <div className="field">
              <label>Session name <span className="muted" style={{fontWeight:400, fontSize:11.5}}>(optional — auto-generated if empty)</span></label>
              <input className="input mono" placeholder="e.g. claude-code" value={name} onChange={e=>setName(e.target.value)} autoFocus />
            </div>

            <div className="field">
              <label>Working directory</label>
              <div className="dir-row">
                <input
                  className="input mono"
                  value={cwd}
                  onChange={e=>{ cwdEdited.current = true; setCwd(e.target.value); }}
                  onFocus={caretToEnd}
                  placeholder="~/"
                  spellCheck={false}
                />
                <label className="checkbox dir-create" title="Create the directory if it doesn't exist yet">
                  <input type="checkbox" checked={createDir} onChange={e=>setCreateDir(e.target.checked)} /> Create
                </label>
              </div>
              <div className="hint">The session starts here. Change the default in Settings → Defaults.</div>
            </div>

            <div className="field">
              <label>Launch</label>
              <div className="slide-toggle" data-state={launch}>
                <span className="slide-thumb" aria-hidden="true" />
                <button type="button" className={`slide-opt ${launch==='shell'?'active':''}`} onClick={()=>setLaunch('shell')}>Start in shell</button>
                <button type="button" className={`slide-opt ${launch==='command'?'active':''}`} onClick={()=>setLaunch('command')}>Command</button>
              </div>
              {launch === 'command' && (
                <input
                  className="input mono"
                  style={{marginTop:8}}
                  value={command}
                  onChange={e=>setCommand(e.target.value)}
                  onFocus={caretToEnd}
                  placeholder="e.g. claude"
                  spellCheck={false}
                />
              )}
              <div className="hint">
                {launch === 'command'
                  ? 'Typed into the session once it starts — the shell stays alive when the command exits.'
                  : 'Just a shell, nothing typed for you.'}
              </div>
            </div>

            <div className="field">
              <div className="flavour-row">
                <label className="checkbox flavour" title="Removes session on disconnect">
                  <input type="checkbox" checked={throwaway} onChange={e=>setThrowaway(e.target.checked)} /> Throwaway
                </label>
                <label className="checkbox flavour" title="Hides session from dashboard">
                  <input type="checkbox" checked={incognito} onChange={e=>setIncognito(e.target.checked)} /> Incognito
                </label>
              </div>
              {(throwaway || incognito) && (
                <div className="hint">
                  {throwaway && incognito
                    ? 'Hidden from the dashboard, and deleted once you disconnect or leave it idle 15 minutes.'
                    : throwaway
                      ? 'Killed and forgotten once you disconnect, or after 15 minutes with nothing attached.'
                      : 'Runs normally but never appears in the app — only tmux on the host will show it.'}
                </div>
              )}
            </div>

            {HOSTS.length === 0 && (
              <div className="muted" style={{fontSize:12.5, marginBottom: 12}}>No hosts registered yet — add one from the Hosts page first.</div>
            )}
            {HOSTS.length > 1 && (
              <div className="field">
                <label>Target host</label>
                <div style={{display:'flex', flexDirection:'column', gap:6}}>
                  {HOSTS.map(h => (
                    <div key={h.id} className={`radio-card ${host===h.id?'selected':''}`} onClick={()=>setHost(h.id)}>
                      <StatusDot status={h.status==='online'?'active':'offline'} />
                      <div style={{flex:1}}>
                        <div className="radio-title mono">{h.fqdn}</div>
                        <div className="radio-sub">{h.user}@{h.fqdn.split(':')[0]} · {h.os}</div>
                      </div>
                      <Pill variant={h.sessions>0?'accent':'default'} mono>{h.sessions} sess</Pill>
                    </div>
                  ))}
                </div>
              </div>
            )}
            {HOSTS.length === 1 && (
              <div className="field">
                <label>Target host</label>
                <div className="radio-card selected" style={{cursor:'default'}}>
                  <StatusDot status={HOSTS[0].status==='online'?'active':'offline'} />
                  <div style={{flex:1}}>
                    <div className="radio-title mono">{HOSTS[0].fqdn}</div>
                    <div className="radio-sub">{HOSTS[0].user}@{HOSTS[0].fqdn.split(':')[0]} · {HOSTS[0].os}</div>
                  </div>
                </div>
              </div>
            )}

            <button type="button" className="btn btn-ghost btn-sm" onClick={() => setShowAdvanced(v => !v)} style={{padding:'4px 0', marginTop:4}}>
              {showAdvanced ? '▾' : '▸'} Advanced options
            </button>

            {showAdvanced && (
              <div style={{marginTop: 10}}>
                <div className="field">
                  <label className="checkbox">
                    <input type="checkbox" checked={attach} onChange={e=>setAttach(e.target.checked)} /> Attach immediately
                  </label>
                </div>
              </div>
            )}

            {err && <div style={{color:'var(--err)', fontSize:12.5, marginTop:10}}>{err}</div>}
          </div>

          <div className="modal-foot">
            <Button variant="ghost" type="button" onClick={onClose}>Cancel</Button>
            <div style={{flex:1}}/>
            <Button variant="primary" type="submit" icon={IconPlay} disabled={busy || !host}>
              {busy ? 'Creating…' : (attach ? 'Create & attach' : 'Create')}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};

Object.assign(window, { NewSession });
