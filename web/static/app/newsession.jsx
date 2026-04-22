// New Session modal — single-form (no stepper). Required: a host (auto-selected
// when only one exists). Name auto-generates if left empty. Everything else is
// optional. Minimum path is one click of Create; Enter also submits.

const NewSession = ({ store, onClose }) => {
  const HOSTS = store.hosts;
  const [host, setHost] = React.useState(HOSTS[0] ? HOSTS[0].id : '');
  const [name, setName] = React.useState('');
  const [cwd, setCwd] = React.useState('');
  const [attach, setAttach] = React.useState(true);
  const [showAdvanced, setShowAdvanced] = React.useState(false);
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');

  React.useEffect(() => { if (!host && HOSTS[0]) setHost(HOSTS[0].id); }, [HOSTS.length]);

  const submit = async (e) => {
    if (e && e.preventDefault) e.preventDefault();
    if (!host) { setErr('Pick a host first.'); return; }
    setErr(''); setBusy(true);
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
                  <label>Working directory</label>
                  <input className="input mono" value={cwd} onChange={e=>setCwd(e.target.value)} placeholder="~/" />
                  <div className="hint">Defaults to the user's home. The tmux session is blank — run any command after attach.</div>
                </div>
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
