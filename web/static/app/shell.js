const Sidebar = ({
  view,
  setView,
  openPalette,
  sessionCount,
  hostCount,
  activeCount,
  favCount,
  version,
  open
}) => {
  return React.createElement("aside", {
    className: `sidebar${open ? ' open' : ''}`
  }, React.createElement("div", {
    className: "brand"
  }, React.createElement(BrandMark, {
    size: 30
  }), React.createElement("span", {
    className: "brand-name"
  }, "SSH-to-go")), React.createElement("div", {
    className: "nav-section"
  }, React.createElement(NavItem, {
    icon: IconDashboard,
    label: "Dashboard",
    active: view === 'dashboard',
    onClick: () => setView('dashboard')
  }), React.createElement(NavItem, {
    icon: IconTerminal,
    label: "Sessions",
    active: view === 'sessions',
    onClick: () => setView('sessions'),
    badge: sessionCount
  }), React.createElement(NavItem, {
    icon: IconServer,
    label: "Hosts",
    active: view === 'hosts',
    onClick: () => setView('hosts'),
    badge: hostCount
  }), React.createElement(NavItem, {
    icon: IconSettings,
    label: "Settings",
    active: view === 'settings',
    onClick: () => setView('settings')
  })), React.createElement("div", {
    className: "nav-section"
  }, React.createElement("div", {
    className: "nav-label"
  }, "Filters"), React.createElement(NavItem, {
    icon: IconSearch,
    label: "Search\u2026",
    onClick: openPalette,
    badge: "\u2318K"
  }), React.createElement(NavItem, {
    icon: IconCheck,
    label: "Active sessions",
    onClick: () => setView('sessions', 'active'),
    badge: activeCount
  }), React.createElement(NavItem, {
    icon: IconStar,
    label: "Favorites",
    onClick: () => setView('sessions', 'favorites'),
    badge: favCount
  })), React.createElement("div", {
    className: "sidebar-footer"
  }, React.createElement("span", {
    className: "dot ok",
    style: {
      width: 6,
      height: 6,
      boxShadow: 'none'
    }
  }), React.createElement("span", null, version ? version.startsWith('v') ? version : `v${version}` : 'ssh-to-go')));
};
const NavItem = ({
  icon: Icon,
  label,
  active,
  onClick,
  badge,
  tag,
  color
}) => {
  if (tag) {
    const c = {
      violet: 'oklch(0.7 0.18 300)',
      teal: 'oklch(0.72 0.12 195)',
      amber: 'oklch(0.78 0.14 75)'
    }[color];
    return React.createElement("div", {
      className: "nav-item",
      onClick: onClick
    }, React.createElement("span", {
      style: {
        width: 16,
        display: 'grid',
        placeItems: 'center'
      }
    }, React.createElement("span", {
      style: {
        width: 8,
        height: 8,
        borderRadius: 2,
        background: c
      }
    })), React.createElement("span", {
      style: {
        fontFamily: 'var(--font-mono)',
        fontSize: 12.5
      }
    }, label));
  }
  return React.createElement("div", {
    className: `nav-item ${active ? 'active' : ''}`,
    onClick: onClick
  }, Icon && React.createElement(Icon, {
    size: 16
  }), React.createElement("span", null, label), badge != null && React.createElement("span", {
    className: "nav-badge"
  }, badge));
};
const Topbar = ({
  openPalette,
  theme,
  toggleTheme,
  openNewSession,
  openTweaks,
  toggleSidebar
}) => {
  return React.createElement("header", {
    className: "topbar"
  }, React.createElement("button", {
    className: "mobile-menu-btn",
    onClick: toggleSidebar,
    title: "Menu",
    "aria-label": "Toggle sidebar"
  }, React.createElement("svg", {
    width: "20",
    height: "20",
    viewBox: "0 0 24 24",
    fill: "none",
    stroke: "currentColor",
    strokeWidth: "2",
    strokeLinecap: "round",
    strokeLinejoin: "round"
  }, React.createElement("line", {
    x1: "3",
    y1: "6",
    x2: "21",
    y2: "6"
  }), React.createElement("line", {
    x1: "3",
    y1: "12",
    x2: "21",
    y2: "12"
  }), React.createElement("line", {
    x1: "3",
    y1: "18",
    x2: "21",
    y2: "18"
  }))), React.createElement("div", {
    className: "search",
    onClick: openPalette
  }, React.createElement(IconSearch, {
    size: 14
  }), React.createElement("span", {
    className: "search-placeholder"
  }, "Search sessions, hosts, commands\u2026"), React.createElement("span", {
    className: "search-kbd"
  }, React.createElement(Kbd, null, "\u2318"), React.createElement(Kbd, null, "K"))), React.createElement("div", {
    className: "topbar-right"
  }, React.createElement("button", {
    className: "icon-btn",
    onClick: openTweaks,
    title: "Tweaks"
  }, React.createElement(IconActivity, {
    size: 15
  })), React.createElement("button", {
    className: "icon-btn",
    onClick: toggleTheme,
    title: "Toggle theme"
  }, theme === 'dark' ? React.createElement(IconSun, {
    size: 15
  }) : React.createElement(IconMoon, {
    size: 15
  })), React.createElement("a", {
    className: "icon-btn",
    href: "/logout",
    title: "Log out",
    onClick: e => {
      e.preventDefault();
      fetch('/api/auth/logout', {
        method: 'POST'
      }).then(() => location.href = '/login');
    }
  }, React.createElement(IconExternalLink, {
    size: 15
  })), React.createElement("div", {
    style: {
      width: 1,
      height: 20,
      background: 'var(--hairline)',
      margin: '0 4px'
    }
  }), React.createElement(Button, {
    variant: "primary",
    size: "sm",
    icon: IconPlus,
    onClick: openNewSession
  }, "New session")));
};
Object.assign(window, {
  Sidebar,
  Topbar,
  NavItem
});