// New Session modal

const NewSession = ({ store, onClose }) => {
  const HOSTS = store.hosts;
  const [step, setStep] = React.useState(1);
  const [host, setHost] = React.useState(HOSTS[0] ? HOSTS[0].id : '');
  const [name, setName] = React.useState('');
  const [cwd, setCwd] = React.useState('');
  const [attach, setAttach] = React.useState(true);
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');
  const [kind, setKind] = React.useState('shell');

  React.useEffect(() => { if (!host && HOSTS[0]) setHost(HOSTS[0].id); }, [HOSTS.length]);
  React.useEffect(() => { setCwd(prev => prev || (name ? `~/projects/${name}` : '')); }, [name]);

  const submit = async () => {
    setErr(''); setBusy(true);
    try {
      const finalName = name || `session-${Math.random().toString(36).slice(2, 7)}`;
      await createSession(host, finalName, cwd || '');
      onClose();
      if (attach) openTerminal(host, finalName);
    } catch (e) {
      setErr(e.message || 'failed');
    } finally {
      setBusy(false);
    }
  };

  // The backend creates a blank tmux session; the command dropdown is visual
  // (the user runs their own command after attach). Kept for UX parity.
  const kinds = [
    { id: 'shell', label: 'Blank shell', sub: '$SHELL' },
    { id: 'claude', label: 'Claude Code', sub: 'claude' },
    { id: 'custom', label: 'Custom command', sub: 'run anything' },
  ];

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e)=>e.stopPropagation()}>
        <div className="modal-head">
          <div style={{display:'flex', justifyContent:'space-between', alignItems:'center', gap:12}}>
            <div>
              <h3>New session</h3>
              <p>Spin up a detached tmux session on one of your hosts.</p>
            </div>
            <button className="icon-btn" onClick={onClose}><IconClose size={15}/></button>
          </div>
          <div className="stepper mt-3">
            <div className={`step ${step===1?'active':step>1?'done':''}`}>
              <span className="step-num">{step>1 ? <IconCheck size={12}/> : 1}</span> Host
            </div>
            <span className="step-divider"/>
            <div className={`step ${step===2?'active':step>2?'done':''}`}>
              <span className="step-num">{step>2 ? <IconCheck size={12}/> : 2}</span> Command
            </div>
            <span className="step-divider"/>
            <div className={`step ${step===3?'active':''}`}>
              <span className="step-num">3</span> Review
            </div>
          </div>
        </div>

        <div className="modal-body">
          {step === 1 && (
            <>
              <div className="field">
                <label>Target host</label>
                {HOSTS.length === 0 && <div className="muted" style={{fontSize:12.5}}>No hosts registered yet — add one from the Hosts page first.</div>}
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
              <div className="field">
                <label>Session name</label>
                <input className="input mono" placeholder={kinds.find(k=>k.id===kind)?.id || 'my-session'} value={name} onChange={e=>setName(e.target.value)} />
                <div className="hint">Auto-generated if left empty. Lowercase, no spaces.</div>
              </div>
            </>
          )}

          {step === 2 && (
            <>
              <div className="field">
                <label>What to run</label>
                <div className="radio-grid">
                  {kinds.map(k => (
                    <div key={k.id} className={`radio-card ${kind===k.id?'selected':''}`} onClick={()=>setKind(k.id)}>
                      <div style={{flex:1}}>
                        <div className="radio-title">{k.label}</div>
                        <div className="radio-sub">{k.sub}</div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
              {kind === 'custom' && (
                <div className="field">
                  <label>Command</label>
                  <input className="input mono" placeholder="e.g. npm run dev" />
                </div>
              )}
              <div className="field">
                <label>Working directory</label>
                <input className="input mono" value={cwd} onChange={e => setCwd(e.target.value)} placeholder={`~/projects/${name || 'new'}`} />
              </div>
              <div className="field">
                <label className="checkbox">
                  <input type="checkbox" checked={attach} onChange={e => setAttach(e.target.checked)} /> Attach immediately
                </label>
              </div>
            </>
          )}

          {step === 3 && (
            <div>
              <div style={{background: 'var(--bg-elev-2)', border: '1px solid var(--hairline)', borderRadius: 8, padding: 14, marginBottom: 12}}>
                <div className="row gap-3" style={{marginBottom: 10}}>
                  <SessIcon kind="terminal" color="indigo"/>
                  <span className="mono" style={{fontWeight:500}}>{name || 'new-session'}</span>
                  <Pill variant="default" mono>will be created</Pill>
                </div>
                <div style={{fontFamily:'var(--font-mono)', fontSize:12, color:'var(--fg-muted)', lineHeight:1.8}}>
                  <div><span style={{color:'var(--fg-subtle)'}}>host  </span> {HOSTS.find(h=>h.id===host)?.fqdn}</div>
                  <div><span style={{color:'var(--fg-subtle)'}}>user  </span> {HOSTS.find(h=>h.id===host)?.user}</div>
                  <div><span style={{color:'var(--fg-subtle)'}}>cwd   </span> {cwd || `~/projects/${name || 'new'}`}</div>
                </div>
              </div>
              <div className="hint" style={{fontSize:12, color:'var(--fg-subtle)'}}>
                A detached tmux session will be created on the host. {attach ? 'Your browser will open the terminal tab on success.' : 'You can attach later from any browser.'}
              </div>
              {err && <div style={{color:'var(--err)', fontSize:12.5, marginTop:10}}>{err}</div>}
            </div>
          )}
        </div>

        <div className="modal-foot">
          {step > 1 && <Button variant="ghost" onClick={()=>setStep(step-1)}>Back</Button>}
          <div style={{flex:1}}/>
          <Button variant="ghost" onClick={onClose}>Cancel</Button>
          {step < 3
            ? <Button variant="primary" onClick={()=>setStep(step+1)} disabled={!host}>Continue</Button>
            : <Button variant="primary" icon={IconPlay} onClick={submit} disabled={busy || !host}>{busy ? 'Creating…' : 'Create' + (attach ? ' & attach' : '')}</Button>}
        </div>
      </div>
    </div>
  );
};

Object.assign(window, { NewSession });
