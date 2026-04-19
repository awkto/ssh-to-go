// Shell: sidebar + topbar

const Sidebar = ({ view, setView, openPalette, sessionCount, hostCount, activeCount, favCount, version }) => {
  return (
    <aside className="sidebar">
      <div className="brand">
        <BrandMark size={30} />
        <span className="brand-name">SSH-to-go</span>
      </div>

      <div className="nav-section">
        <NavItem icon={IconDashboard} label="Dashboard" active={view==='dashboard'} onClick={() => setView('dashboard')} />
        <NavItem icon={IconTerminal} label="Sessions" active={view==='sessions'} onClick={() => setView('sessions')} badge={sessionCount} />
        <NavItem icon={IconServer} label="Hosts" active={view==='hosts'} onClick={() => setView('hosts')} badge={hostCount} />
        <NavItem icon={IconSettings} label="Settings" active={view==='settings'} onClick={() => setView('settings')} />
      </div>

      <div className="nav-section">
        <div className="nav-label">Filters</div>
        <NavItem icon={IconSearch} label="Search…" onClick={openPalette} badge="⌘K" />
        <NavItem icon={IconCheck} label="Active sessions" onClick={() => setView('sessions', 'active')} badge={activeCount} />
        <NavItem icon={IconStar} label="Favorites" onClick={() => setView('sessions', 'favorites')} badge={favCount} />
      </div>

      {/* Workspaces section hidden until host tags land (issue #20). */}

      <div className="sidebar-footer">
        <span className="dot ok" style={{width:6, height:6, boxShadow:'none'}}></span>
        <span>{version ? `v${version}` : 'ssh-to-go'}</span>
      </div>
    </aside>
  );
};

const NavItem = ({ icon: Icon, label, active, onClick, badge, tag, color }) => {
  if (tag) {
    const c = {
      violet: 'oklch(0.7 0.18 300)',
      teal: 'oklch(0.72 0.12 195)',
      amber: 'oklch(0.78 0.14 75)',
    }[color];
    return (
      <div className="nav-item" onClick={onClick}>
        <span style={{width:16, display:'grid', placeItems:'center'}}>
          <span style={{width:8, height:8, borderRadius:2, background: c}}></span>
        </span>
        <span style={{fontFamily:'var(--font-mono)', fontSize:12.5}}>{label}</span>
      </div>
    );
  }
  return (
    <div className={`nav-item ${active ? 'active' : ''}`} onClick={onClick}>
      {Icon && <Icon size={16} />}
      <span>{label}</span>
      {badge != null && <span className="nav-badge">{badge}</span>}
    </div>
  );
};

const Topbar = ({ openPalette, theme, toggleTheme, openNewSession, openTweaks }) => {
  return (
    <header className="topbar">
      <div className="search" onClick={openPalette}>
        <IconSearch size={14} />
        <span className="search-placeholder">Search sessions, hosts, commands…</span>
        <span className="search-kbd">
          <Kbd>⌘</Kbd>
          <Kbd>K</Kbd>
        </span>
      </div>

      <div className="topbar-right">
        <button className="icon-btn" onClick={openTweaks} title="Tweaks">
          <IconActivity size={15} />
        </button>
        <button className="icon-btn" onClick={toggleTheme} title="Toggle theme">
          {theme === 'dark' ? <IconSun size={15}/> : <IconMoon size={15}/>}
        </button>
        <a className="icon-btn" href="/logout" title="Log out" onClick={(e)=>{ e.preventDefault(); fetch('/api/auth/logout', { method: 'POST' }).then(() => location.href = '/login'); }}>
          <IconExternalLink size={15} />
        </a>
        <div style={{width: 1, height: 20, background: 'var(--hairline)', margin: '0 4px'}} />
        <Button variant="primary" size="sm" icon={IconPlus} onClick={openNewSession}>New session</Button>
      </div>
    </header>
  );
};

Object.assign(window, { Sidebar, Topbar, NavItem });
