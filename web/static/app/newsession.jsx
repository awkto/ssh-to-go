// New Session modal — design option 5a.
//
// One screen, no stepper and no "Advanced" drawer: the host lives in the
// header, name and directory share a row, the two consequential choices
// ("start in what?", "then what?") are segmented controls with a live
// preview beside them, and the four per-row hints of the old form collapse
// into one sentence in the footer that describes the whole session.
//
// The modal's height is deliberately constant across every combination of
// toggles — nothing appears or disappears, only text and colour change. The
// old form grew and shrank as you touched it, which made it feel unstable.

// Choices the user is likely to repeat are remembered locally rather than in
// server settings: they're per-browser habits, not deployment config. Two
// exceptions: the working directory seeds from a server setting so it's the
// same on every device, and the recent-command chips come from the server so
// they reflect what actually ran — including sessions started from another
// device, the HTTP API, or an agent over MCP.
const NS_LS = {
  createDir: 's2g:newsession:create-dir',
  launch: 's2g:newsession:launch',
  command: 's2g:newsession:command',
  after: 's2g:newsession:after',
};
const nsGet = (k, fallback) => {
  try { const v = localStorage.getItem(k); return v === null ? fallback : v; } catch (_) { return fallback; }
};
const nsSet = (k, v) => { try { localStorage.setItem(k, v); } catch (_) {} };

// The name the session gets when the field is left empty. Generated once per
// modal (and again when Throwaway flips, because the prefix differs) so the
// placeholder, the ssh preview and the session that actually gets created all
// agree — rolling it at submit time would show one name and create another.
const nsAutoName = (throwaway) =>
  (throwaway ? 'tmp-' : 'session-') + Math.random().toString(36).slice(2, 6);

// --- $name / $date preview -------------------------------------------------
//
// A deliberately small mirror of internal/sessionvars (Go) and of
// sanitizeSessionName. PREVIEW ONLY: the server does the substitution that
// actually creates the session. This exists because the feature is invisible
// otherwise — you discover it by watching $name turn into the name you just
// typed. Keep the two in step: same variables, same word boundary, and above
// all the same rule that every OTHER $-form is left for the remote shell.
const NS_VARS = ['name', 'date'];
const NS_VARS_HINT =
  '$name = this session’s name, $date = today as YYYY-MM-DD. ' +
  '${name} works too. Any other $VAR is left for the shell.';
const nsIdent = (c) => /[A-Za-z0-9_]/.test(c);
// Mirrors sanitizeSessionName: $name is the name tmux will really use, so a
// typed "my session" previews as ~/sessions/my-session, not with a space.
const nsSanitize = (s) => s.trim().replace(/[ \t]+/g, '-');
const nsValue = (v, name) => (name === 'name' ? v : new Date().toLocaleDateString('en-CA'));
// Returns the variable's name and how many characters the reference spans,
// or null when it is not one of ours.
const nsMatch = (s) => {
  if (s.length < 2 || s[0] !== '$') return null;
  if (s[1] === '{') {
    const end = s.indexOf('}');
    if (end < 0) return null;
    const name = s.slice(2, end);
    return NS_VARS.includes(name) ? { name, len: end + 1 } : null;
  }
  for (const name of NS_VARS) {
    if (!s.startsWith(name, 1)) continue;
    const rest = s.slice(1 + name.length);
    if (rest && nsIdent(rest[0])) return null; // $nameless is not $name
    return { name, len: 1 + name.length };
  }
  return null;
};
const nsExpand = (s, sessionName) => {
  if (!s || s.indexOf('$') < 0) return s;
  let out = '';
  for (let i = 0; i < s.length; ) {
    // \$name keeps the dollar literal for the shell; \$HOME is the shell's
    // own escape and stays exactly as typed.
    if (s[i] === '\\' && s[i + 1] === '$') {
      const esc = nsMatch(s.slice(i + 1));
      if (esc) { out += s.substr(i + 1, esc.len); i += 1 + esc.len; continue; }
    }
    if (s[i] === '$') {
      const m = nsMatch(s.slice(i));
      if (m) { out += nsValue(sessionName, m.name); i += m.len; continue; }
    }
    out += s[i];
    i++;
  }
  return out;
};

const NewSession = ({ store, onClose }) => {
  const HOSTS = store.hosts;
  const defaultDir = (store.settings && store.settings.new_session_dir) || '~/sessions/';
  const [host, setHost] = React.useState(HOSTS[0] ? HOSTS[0].id : '');
  const [name, setName] = React.useState('');
  const [cwd, setCwd] = React.useState(defaultDir);
  const [createDir, setCreateDir] = React.useState(nsGet(NS_LS.createDir, '1') === '1');
  const [launch, setLaunch] = React.useState(nsGet(NS_LS.launch, 'shell') === 'command' ? 'command' : 'shell');
  const [command, setCommand] = React.useState(nsGet(NS_LS.command, ''));
  const [after, setAfter] = React.useState(nsGet(NS_LS.after, 'attach') === 'handoff' ? 'handoff' : 'attach');
  // Deliberately NOT persisted, unlike the fields above: both are
  // consequential enough that they should be a per-create decision rather
  // than something yesterday's session left switched on.
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
  // What $name expands to: the sanitized form, because that is the name the
  // server will give tmux.
  const varName = nsSanitize(effName);
  const isCommand = launch === 'command';
  const isHandoff = after === 'handoff';

  React.useEffect(() => { if (!host && HOSTS[0]) setHost(HOSTS[0].id); }, [HOSTS.length]);

  // Settings can land after the modal mounts (first paint of a cold load).
  // Adopt the configured default then — but never overwrite typing.
  React.useEffect(() => { if (!cwdEdited.current) setCwd(defaultDir); }, [defaultDir]);

  React.useEffect(() => { nsSet(NS_LS.createDir, createDir ? '1' : '0'); }, [createDir]);
  React.useEffect(() => { nsSet(NS_LS.launch, launch); }, [launch]);
  React.useEffect(() => { nsSet(NS_LS.command, command); }, [command]);
  React.useEffect(() => { nsSet(NS_LS.after, after); }, [after]);
  React.useEffect(() => () => clearTimeout(copyTimer.current), []);

  // Throwaway sessions are named tmp-* so they read as disposable at a
  // glance in tmux. Only the generated name changes — a name you typed is
  // still yours.
  React.useEffect(() => { setAutoName(nsAutoName(throwaway)); }, [throwaway]);

  // The real ssh command comes from the server, which knows the port and any
  // proxy details the browser doesn't. Until it lands (and if the request
  // fails) fall back to a locally-built string so the line is never empty —
  // an empty line would collapse and change the modal's height.
  const fallbackSsh = React.useMemo(() => {
    if (!hostObj) return '';
    const [addr, port] = String(hostObj.fqdn || '').split(':');
    const p = port && port !== '22' ? `-p ${port} ` : '';
    return `ssh -t ${p}${hostObj.user}@${addr} tmux attach-session -t "${effName}"`;
  }, [hostObj && hostObj.fqdn, hostObj && hostObj.user, effName]);

  React.useEffect(() => {
    if (!host) return;
    let cancelled = false;
    setSshCmd('');
    const t = setTimeout(() => {
      getHandoff(host, effName)
        .then(cmd => { if (!cancelled) setSshCmd(cmd); })
        .catch(() => {});
    }, 250);
    return () => { cancelled = true; clearTimeout(t); };
  }, [host, effName]);

  const shownSsh = sshCmd || fallbackSsh;

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

  const copySsh = async () => {
    try { await navigator.clipboard.writeText(shownSsh); } catch (_) {}
    setCopied(true);
    clearTimeout(copyTimer.current);
    copyTimer.current = setTimeout(() => setCopied(false), 1600);
  };

  const recents = (store.recentCommands || []).slice(0, 4);
  const pickRecent = (cmd) => { setCommand(cmd); setLaunch('command'); };
  const forgetRecent = async (cmd) => {
    try { await forgetRecentCommand(cmd); } catch (ex) { setErr(ex.message || 'could not forget that command'); }
  };

  // The directory and command with $name/$date filled in — what the session
  // will actually get. Shown wherever the raw field would otherwise be
  // echoed back, so typing a name visibly resolves the variables.
  const shownCwd = nsExpand(cwd.trim(), varName);
  const shownCommand = nsExpand(command.trim(), varName);

  // One sentence describing the session that is about to exist, recomposed
  // from state. It replaces the four hint paragraphs the old form carried.
  const summary = (() => {
    const head = isCommand
      ? `Runs ${shownCommand || '…'}`
      : 'Opens a shell';
    const where = `in ${shownCwd || '~'} on ${hostObj ? hostObj.fqdn : 'this host'}`;
    const tail = [throwaway ? 'ends when you detach' : 'keeps running until killed'];
    if (incognito) tail.push('untracked');
    tail.push(isHandoff ? 'hands off to your terminal' : 'attaches in a web tab');
    return `${head} ${where} · ${tail.join(', ')}.`;
  })();

  const submit = async (e) => {
    if (e && e.preventDefault) e.preventDefault();
    if (!host) { setErr('Pick a host first.'); return; }
    const runCmd = isCommand ? command.trim() : '';
    if (isCommand && !runCmd) { setErr('Type a command, or switch back to shell.'); return; }
    setErr(''); setBusy(true);
    try {
      await createSession(host, effName, cwd.trim() || '', { createDir, command: runCmd, throwaway, incognito });
      if (isHandoff) {
        // Copy before closing: the modal owns the string, and the session
        // is useless to hand off if you can't paste it.
        try { await navigator.clipboard.writeText(shownSsh); } catch (_) {}
      }
      onClose();
      if (!isHandoff) openTerminal(host, effName);
    } catch (ex) {
      setErr(ex.message || 'failed');
    } finally {
      setBusy(false);
    }
  };

  // Shorten a hostname to its first label the way a shell prompt would —
  // but leave an IP alone, since "127" is not a shorter way to say anything.
  const shortHost = (() => {
    if (!hostObj) return 'host';
    const addr = String(hostObj.fqdn).split(':')[0];
    return /^[\d.]+$/.test(addr) ? addr : addr.split('.')[0];
  })();
  const seg = (active) => `ns-seg${active ? ' active' : ''}`;

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e)=>{ e.stopPropagation(); setHostMenu(false); }}>
        <form onSubmit={submit}>
          <div className="ns-head">
            <h3>New session</h3>
            <div className="ns-hostpick">
              <button
                type="button"
                className="ns-hostchip"
                disabled={HOSTS.length <= 1}
                onClick={(e)=>{ e.stopPropagation(); setHostMenu(v => !v); }}
                title={HOSTS.length > 1 ? 'Choose a host' : undefined}
              >
                <StatusDot status={hostObj && hostObj.status === 'online' ? 'active' : 'offline'} />
                <span className="mono">{hostObj ? hostObj.fqdn : 'no hosts'}</span>
                {HOSTS.length > 1 && <IconChevronDown size={12} />}
              </button>
              {hostMenu && (
                <div className="ns-hostmenu" onClick={(e)=>e.stopPropagation()}>
                  {HOSTS.map(h => (
                    <button
                      type="button"
                      key={h.id}
                      className={`ns-hostopt${h.id === host ? ' active' : ''}`}
                      onClick={()=>{ setHost(h.id); setHostMenu(false); }}
                    >
                      <StatusDot status={h.status === 'online' ? 'active' : 'offline'} />
                      <span className="mono" style={{flex:1}}>{h.fqdn}</span>
                      <span className="ns-hostopt-sub">{h.sessions} sess</span>
                    </button>
                  ))}
                </div>
              )}
            </div>
            <div style={{flex:1}} />
            <button type="button" className="ns-close" onClick={onClose} aria-label="Close"><IconClose size={15}/></button>
          </div>

          <div className="ns-body">
            <div className="ns-grid">
              <div style={{minWidth:0}}>
                <label className="ns-label" htmlFor="ns-name">Name</label>
                <input
                  id="ns-name"
                  className="ns-input mono"
                  placeholder={autoName}
                  value={name}
                  onChange={e=>setName(e.target.value)}
                  spellCheck={false}
                  autoFocus
                />
              </div>
              <div style={{minWidth:0}}>
                <label className="ns-label" htmlFor="ns-dir">Directory</label>
                <div className="ns-fieldbox">
                  <input
                    id="ns-dir"
                    className="ns-bare mono"
                    value={cwd}
                    onChange={e=>{ cwdEdited.current = true; setCwd(e.target.value); }}
                    onFocus={caretToEnd}
                    placeholder="~/"
                    title={NS_VARS_HINT}
                    spellCheck={false}
                  />
                  <button
                    type="button"
                    className={`ns-mkdir${createDir ? ' on' : ''}`}
                    onClick={()=>setCreateDir(v => !v)}
                    aria-pressed={createDir}
                    title="Create the directory if it doesn't exist yet"
                  >mkdir</button>
                </div>
              </div>
            </div>

            <div>
              <div className="ns-labelline">
                <label className="ns-label" style={{marginRight:'auto'}}>Start in</label>
                {recents.map(rc => (
                  <span className="ns-chip" key={rc.command} title={`${rc.command} — used ${rc.count}×`}>
                    <button type="button" className="ns-chip-pick mono" onClick={()=>pickRecent(rc.command)}>{rc.command}</button>
                    <button
                      type="button"
                      className="ns-chip-forget"
                      onClick={()=>forgetRecent(rc.command)}
                      aria-label={`Forget ${rc.command}`}
                      title={`Forget ${rc.command}`}
                    >×</button>
                  </span>
                ))}
              </div>
              <div className="ns-control">
                <div className="ns-segs">
                  <button type="button" className={seg(!isCommand)} onClick={()=>setLaunch('shell')}>shell</button>
                  <button type="button" className={seg(isCommand)} onClick={()=>setLaunch('command')}>command</button>
                </div>
                <div className="ns-slot">
                  {isCommand ? (
                    <>
                      <span className="ns-dollar mono">$</span>
                      <input
                        className="ns-bare mono"
                        value={command}
                        onChange={e=>setCommand(e.target.value)}
                        onFocus={caretToEnd}
                        placeholder="claude"
                        title={NS_VARS_HINT}
                        spellCheck={false}
                      />
                    </>
                  ) : (
                    <span className="ns-ghost mono">{hostObj ? hostObj.user : 'you'}@{shortHost}:{shownCwd || '~'}$<i className="ns-caret" /></span>
                  )}
                </div>
              </div>
            </div>

            <div>
              <div className="ns-labelline">
                <label className="ns-label" style={{marginRight:'auto'}}>Then</label>
                <button
                  type="button"
                  className={`ns-flag${throwaway ? ' on' : ''}`}
                  aria-pressed={throwaway}
                  onClick={()=>setThrowaway(v => !v)}
                  title="Removed as soon as you leave it"
                ><IconClock size={13}/>throwaway</button>
                <button
                  type="button"
                  className={`ns-flag${incognito ? ' on' : ''}`}
                  aria-pressed={incognito}
                  onClick={()=>setIncognito(v => !v)}
                  title="Not tracked in the app"
                ><IconMoon size={13}/>incognito</button>
              </div>
              <div className="ns-control">
                <div className="ns-segs">
                  <button type="button" className={seg(!isHandoff)} onClick={()=>setAfter('attach')}>attach here</button>
                  <button type="button" className={seg(isHandoff)} onClick={()=>setAfter('handoff')}>hand off</button>
                </div>
                <div className="ns-slot">
                  <span className="ns-dest">
                    {isHandoff ? <IconTerminal size={13}/> : <IconExternalLink size={13}/>}
                    {isHandoff ? 'your own terminal' : 'new web-terminal tab'}
                  </span>
                </div>
              </div>
              {/* Always rendered, dimmed in attach mode — hiding it would
                  change the modal's height every time you flip the segment. */}
              <div className={`ns-ssh${isHandoff ? ' lit' : ''}`}>
                <span className="ns-ssh-cmd mono">{shownSsh}</span>
                <button type="button" className="ns-copy mono" onClick={copySsh}>
                  <IconCopy size={12}/>{copied ? 'copied' : 'copy'}
                </button>
              </div>
            </div>
          </div>

          <div className="ns-foot">
            {/* The error takes the summary's slot rather than adding a row,
                so a failed create doesn't resize the modal. Two lines are
                reserved and the text is clamped to them; the title carries
                the full string when a long command overruns. */}
            <div className={`ns-summary${err ? ' err' : ''}`} title={err || summary}>
              <span>{err || summary}</span>
            </div>
            <button type="button" className="ns-cancel" onClick={onClose}>Cancel</button>
            <button type="submit" className="ns-primary" disabled={busy || !host}>
              {isHandoff ? <IconCopy size={14}/> : <IconPlay size={14}/>}
              {busy ? 'Creating…' : (isHandoff ? 'Create & copy ssh' : 'Create & attach')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

Object.assign(window, { NewSession });
