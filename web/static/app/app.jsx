// App root

const App = () => {
  const store = useStore();
  const [view, setView] = React.useState(() => {
    const h = (location.hash || '').replace(/^#/, '').split('?')[0];
    if (['dashboard', 'sessions', 'hosts', 'settings'].includes(h)) return h;
    return localStorage.getItem('sshtogo.view') || 'dashboard';
  });
  const [paletteOpen, setPaletteOpen] = React.useState(false);
  const [newSessOpen, setNewSessOpen] = React.useState(false);
  const [tweaksOpen, setTweaksOpen] = React.useState(false);
  const [sessionFilter, setSessionFilter] = React.useState('all');
  const [sidebarOpen, setSidebarOpen] = React.useState(false);

  const navigate = (nextView, filter) => {
    setView(nextView);
    if (nextView === 'sessions') setSessionFilter(filter || 'all');
    setSidebarOpen(false); // auto-dismiss on mobile after navigation
  };

  const [theme, setTheme] = React.useState(() => localStorage.getItem('sshtogo.theme') || 'dark');
  const [accent, setAccent] = React.useState(() => localStorage.getItem('sshtogo.accent') || 'indigo');
  const [density, setDensity] = React.useState(() => localStorage.getItem('sshtogo.density') || 'balanced');

  React.useEffect(() => {
    document.documentElement.dataset.theme = theme;
    document.documentElement.dataset.accent = accent;
    document.documentElement.dataset.density = density;
    localStorage.setItem('sshtogo.theme', theme);
    localStorage.setItem('sshtogo.accent', accent);
    localStorage.setItem('sshtogo.density', density);
  }, [theme, accent, density]);

  React.useEffect(() => {
    localStorage.setItem('sshtogo.view', view);
    history.replaceState(null, '', '#' + view);
  }, [view]);

  React.useEffect(() => {
    const onKey = (e) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setPaletteOpen(p => !p);
      } else if ((e.metaKey || e.ctrlKey) && e.key === 'n') {
        e.preventDefault();
        setNewSessOpen(true);
      } else if (e.key === ',' && (e.metaKey || e.ctrlKey) && e.shiftKey) {
        e.preventDefault();
        setTweaksOpen(t => !t);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const openSession = (s) => openTerminal(s.hostName, s.id);

  const renderView = () => {
    switch (view) {
      case 'dashboard': return <Dashboard store={store} setView={navigate} openSession={openSession} openNewSession={() => setNewSessOpen(true)} />;
      case 'sessions': return <Sessions store={store} openSession={openSession} openNewSession={() => setNewSessOpen(true)} initialFilter={sessionFilter} />;
      case 'hosts': return <Hosts store={store} openNewSession={() => setNewSessOpen(true)} />;
      case 'settings': return <Settings store={store} />;
      default: return <Dashboard store={store} setView={navigate} openSession={openSession} openNewSession={() => setNewSessOpen(true)} />;
    }
  };

  const favCount = store.sessions.filter(s => s.starred).length;

  return (
    <div className="app">
      <Sidebar
        view={view}
        setView={navigate}
        openPalette={() => { setPaletteOpen(true); setSidebarOpen(false); }}
        openTweaks={() => setTweaksOpen(true)}
        sessionCount={store.sessions.length}
        hostCount={store.hosts.length}
        activeCount={store.activeSessionCount}
        favCount={favCount}
        version={store.raw.version}
        open={sidebarOpen}
      />
      <div className={`sidebar-overlay${sidebarOpen ? ' open' : ''}`} onClick={() => setSidebarOpen(false)} />
      <div className="main">
        <Topbar
          openPalette={() => setPaletteOpen(true)}
          theme={theme}
          toggleTheme={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
          openNewSession={() => setNewSessOpen(true)}
          openTweaks={() => setTweaksOpen(true)}
          toggleSidebar={() => setSidebarOpen(o => !o)}
        />
        <div className="content">
          {renderView()}
        </div>
      </div>

      {paletteOpen && (
        <Palette
          store={store}
          onClose={() => setPaletteOpen(false)}
          onOpenSession={(s) => { openSession(s); setPaletteOpen(false); }}
          setView={setView}
          openNewSession={() => setNewSessOpen(true)}
        />
      )}
      {newSessOpen && <NewSession store={store} onClose={() => setNewSessOpen(false)} />}

      {tweaksOpen && (
        <Tweaks
          theme={theme}
          setTheme={setTheme}
          accent={accent}
          setAccent={setAccent}
          density={density}
          setDensity={setDensity}
          onClose={() => setTweaksOpen(false)}
        />
      )}
    </div>
  );
};

ReactDOM.createRoot(document.getElementById('root')).render(<App/>);
