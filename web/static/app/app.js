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
    setSidebarOpen(false);
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
    const onKey = e => {
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
  const openSession = s => openTerminal(s.hostName, s.id);
  const renderView = () => {
    switch (view) {
      case 'dashboard':
        return React.createElement(Dashboard, {
          store: store,
          setView: navigate,
          openSession: openSession,
          openNewSession: () => setNewSessOpen(true)
        });
      case 'sessions':
        return React.createElement(Sessions, {
          store: store,
          openSession: openSession,
          openNewSession: () => setNewSessOpen(true),
          initialFilter: sessionFilter
        });
      case 'hosts':
        return React.createElement(Hosts, {
          store: store,
          openNewSession: () => setNewSessOpen(true)
        });
      case 'settings':
        return React.createElement(Settings, {
          store: store
        });
      default:
        return React.createElement(Dashboard, {
          store: store,
          setView: navigate,
          openSession: openSession,
          openNewSession: () => setNewSessOpen(true)
        });
    }
  };
  const favCount = store.sessions.filter(s => s.starred).length;
  return React.createElement("div", {
    className: "app"
  }, React.createElement(Sidebar, {
    view: view,
    setView: navigate,
    openPalette: () => {
      setPaletteOpen(true);
      setSidebarOpen(false);
    },
    openTweaks: () => setTweaksOpen(true),
    sessionCount: store.sessions.length,
    hostCount: store.hosts.length,
    activeCount: store.activeSessionCount,
    favCount: favCount,
    version: store.raw.version,
    open: sidebarOpen
  }), React.createElement("div", {
    className: `sidebar-overlay${sidebarOpen ? ' open' : ''}`,
    onClick: () => setSidebarOpen(false)
  }), React.createElement("div", {
    className: "main"
  }, React.createElement(Topbar, {
    openPalette: () => setPaletteOpen(true),
    theme: theme,
    toggleTheme: () => setTheme(theme === 'dark' ? 'light' : 'dark'),
    openNewSession: () => setNewSessOpen(true),
    openTweaks: () => setTweaksOpen(true),
    toggleSidebar: () => setSidebarOpen(o => !o)
  }), React.createElement("div", {
    className: "content"
  }, renderView())), paletteOpen && React.createElement(Palette, {
    store: store,
    onClose: () => setPaletteOpen(false),
    onOpenSession: s => {
      openSession(s);
      setPaletteOpen(false);
    },
    setView: setView,
    openNewSession: () => setNewSessOpen(true)
  }), newSessOpen && React.createElement(NewSession, {
    store: store,
    onClose: () => setNewSessOpen(false)
  }), tweaksOpen && React.createElement(Tweaks, {
    theme: theme,
    setTheme: setTheme,
    accent: accent,
    setAccent: setAccent,
    density: density,
    setDensity: setDensity,
    onClose: () => setTweaksOpen(false)
  }));
};
ReactDOM.createRoot(document.getElementById('root')).render(React.createElement(App, null));