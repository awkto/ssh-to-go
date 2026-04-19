function _extends() { return _extends = Object.assign ? Object.assign.bind() : function (n) { for (var e = 1; e < arguments.length; e++) { var t = arguments[e]; for (var r in t) ({}).hasOwnProperty.call(t, r) && (n[r] = t[r]); } return n; }, _extends.apply(null, arguments); }
const Button = ({
  variant = 'secondary',
  size = 'md',
  children,
  icon: Icon,
  onClick,
  ...rest
}) => {
  const cls = `btn btn-${variant}${size === 'sm' ? ' btn-sm' : ''}${size === 'xs' ? ' btn-xs' : ''}`;
  return React.createElement("button", _extends({
    className: cls,
    onClick: onClick
  }, rest), Icon && React.createElement(Icon, {
    size: size === 'xs' ? 12 : 14
  }), children);
};
const Pill = ({
  variant = 'default',
  mono,
  children
}) => React.createElement("span", {
  className: `pill ${variant !== 'default' ? variant : ''} ${mono ? 'mono' : ''}`
}, children);
const Kbd = ({
  children
}) => React.createElement("span", {
  className: "kbd"
}, children);
const StatusDot = ({
  status,
  pulse
}) => {
  const cls = status === 'online' || status === 'active' ? 'ok' : status === 'idle' || status === 'warn' ? 'warn' : status === 'offline' || status === 'err' ? 'err' : '';
  return React.createElement("span", {
    className: `dot ${cls} ${pulse ? 'pulse' : ''}`
  });
};
const Presence = ({
  clients,
  max = 3
}) => {
  if (!clients || clients.length === 0) return React.createElement("span", {
    className: "subtle",
    style: {
      fontFamily: 'var(--font-mono)',
      fontSize: 11
    }
  }, "\u2014");
  const shown = clients.slice(0, max);
  const extra = clients.length - shown.length;
  const colorMap = {
    indigo: {
      bg: 'oklch(0.58 0.18 270 / 0.2)',
      fg: 'oklch(0.75 0.16 270)'
    },
    violet: {
      bg: 'oklch(0.58 0.2 300 / 0.2)',
      fg: 'oklch(0.75 0.18 300)'
    },
    teal: {
      bg: 'oklch(0.62 0.12 195 / 0.2)',
      fg: 'oklch(0.75 0.12 195)'
    },
    amber: {
      bg: 'oklch(0.78 0.14 75 / 0.2)',
      fg: 'oklch(0.8 0.14 75)'
    }
  };
  return React.createElement("span", {
    className: "presence"
  }, shown.map((c, i) => {
    const col = colorMap[c.color] || colorMap.indigo;
    return React.createElement("span", {
      key: i,
      className: "av",
      style: {
        background: col.bg,
        color: col.fg
      }
    }, c.name);
  }), extra > 0 && React.createElement("span", {
    className: "av more"
  }, "+", extra));
};
const ActivityCell = ({
  session
}) => {
  const isActive = session.activity === 'active';
  return React.createElement("span", {
    className: "row gap-2"
  }, React.createElement(StatusDot, {
    status: isActive ? 'active' : 'idle',
    pulse: isActive && session.idle < 30
  }), React.createElement("span", {
    className: isActive ? '' : 'muted',
    style: {
      fontFamily: 'var(--font-mono)',
      fontSize: 12
    }
  }, session.lastInput));
};
Object.assign(window, {
  Button,
  Pill,
  Kbd,
  StatusDot,
  Presence,
  ActivityCell
});