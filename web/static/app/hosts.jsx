// Hosts screen

const Hosts = ({ store, openNewSession }) => {
  const HOSTS = store.hosts;
  const [search, setSearch] = React.useState('');
  const [addOpen, setAddOpen] = React.useState(false);
  const filtered = HOSTS.filter(h => !search || h.fqdn.includes(search));
  return (
    <div>
      <div className="page-head">
        <div className="page-title-block">
          <h1>Hosts</h1>
          <p>{HOSTS.length} hosts registered · {HOSTS.filter(h=>h.status==='online').length} online</p>
        </div>
        <div className="page-actions">
          <Button variant="secondary" size="sm" icon={IconRefresh} onClick={() => scanAll().catch(err => alert(err.message))}>Sync all</Button>
          <Button variant="primary" size="sm" icon={IconPlus} onClick={() => setAddOpen(true)}>Add host</Button>
        </div>
      </div>

      <div className="filter-bar">
        <div className="seg">
          <button className="seg-btn active">All <span className="count">{HOSTS.length}</span></button>
          <button className="seg-btn">Online <span className="count">{HOSTS.filter(h=>h.status==='online').length}</span></button>
          <button className="seg-btn">With sessions <span className="count">{HOSTS.filter(h=>h.sessions>0).length}</span></button>
        </div>
        <div className="search-sm">
          <IconSearch size={13}/>
          <input placeholder="Filter hosts…" value={search} onChange={e=>setSearch(e.target.value)} />
        </div>
      </div>

      <div className="panel">
        <table className="tbl">
          <thead>
            <tr>
              <th style={{width:'28%'}}>Host (FQDN)</th>
              <th>Sessions</th>
              <th>Operating system</th>
              <th>Load (60s)</th>
              <th>CPU / MEM</th>
              <th>Last sync</th>
              <th style={{textAlign:'right'}}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map(h => (
              <tr key={h.id} className="host-row">
                <td>
                  <div className="row gap-3">
                    <StatusDot status={h.status==='online'?'active':'offline'} pulse={h.status==='online'} />
                    <div>
                      <div className="mono host-name">{h.fqdn}</div>
                      <div className="host-user">{h.user}@{h.fqdn.split(':')[0]}</div>
                    </div>
                  </div>
                </td>
                <td>
                  {h.sessions > 0
                    ? <Pill variant="accent">{h.sessions} sessions</Pill>
                    : <span className="muted" style={{fontSize:12.5}}>No sessions</span>}
                </td>
                <td><OsChip os={h.os} kind={h.osKind} /></td>
                <td>{h.cpu != null ? <Sparkline data={h.load} width={100} height={22} /> : <span className="subtle mono" style={{fontSize:11}}>—</span>}</td>
                <td className="mono num" style={{fontSize:12}}>
                  {h.cpu != null ? (
                    <>
                      <span style={{color: h.cpu > 60 ? 'var(--warn)' : 'var(--fg-muted)'}}>{h.cpu}%</span>
                      <span className="muted"> / </span>
                      <span className="muted">{h.mem}%</span>
                    </>
                  ) : <span className="subtle" style={{fontSize:11}}>—</span>}
                </td>
                <td className="muted mono" style={{fontSize:12}}>{h.lastSync}</td>
                <td>
                  <div className="actions-cell">
                    <button className="action-btn primary" onClick={openNewSession}>
                      <IconPlus size={12}/>New session
                    </button>
                    <button className="action-btn" onClick={() => scanAll().catch(err => alert(err.message))} title="Re-scan this host"><IconRefresh size={12}/>Sync</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {addOpen && <AddHostModal onClose={() => setAddOpen(false)} />}
    </div>
  );
};

const AddHostModal = ({ onClose }) => {
  const [name, setName] = React.useState('');
  const [address, setAddress] = React.useState('');
  const [port, setPort] = React.useState(22);
  const [user, setUser] = React.useState('');
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');
  const submit = async (e) => {
    e.preventDefault();
    setErr(''); setBusy(true);
    try {
      await addHost({ name: name || undefined, address, port: Number(port) || 22, user: user || undefined });
      onClose();
    } catch (ex) {
      setErr(ex.message || 'failed');
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e)=>e.stopPropagation()}>
        <div className="modal-head">
          <div style={{display:'flex', justifyContent:'space-between', alignItems:'center', gap:12}}>
            <div>
              <h3>Add host</h3>
              <p>Register a target host for tmux sessions.</p>
            </div>
            <button className="icon-btn" onClick={onClose}><IconClose size={15}/></button>
          </div>
        </div>
        <form onSubmit={submit}>
          <div className="modal-body">
            <div className="field"><label>Address (host or IP)</label><input className="input mono" value={address} onChange={e=>setAddress(e.target.value)} placeholder="host.example.com" required autoFocus /></div>
            <div className="field"><label>Port</label><input className="input mono" type="number" value={port} onChange={e=>setPort(e.target.value)} /></div>
            <div className="field"><label>User</label><input className="input mono" value={user} onChange={e=>setUser(e.target.value)} placeholder="leave empty to use default" /></div>
            <div className="field"><label>Name</label><input className="input mono" value={name} onChange={e=>setName(e.target.value)} placeholder="defaults to address" /></div>
            {err && <div style={{color:'var(--err)', fontSize:12.5}}>{err}</div>}
          </div>
          <div className="modal-foot">
            <Button variant="ghost" onClick={onClose}>Cancel</Button>
            <Button variant="primary" type="submit" disabled={busy}>{busy ? 'Adding…' : 'Add host'}</Button>
          </div>
        </form>
      </div>
    </div>
  );
};

Object.assign(window, { Hosts, AddHostModal });
