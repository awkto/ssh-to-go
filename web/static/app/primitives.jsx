// Primitives: small reusable UI atoms

const Button = ({ variant = 'secondary', size = 'md', children, icon: Icon, onClick, ...rest }) => {
  const cls = `btn btn-${variant}${size === 'sm' ? ' btn-sm' : ''}${size === 'xs' ? ' btn-xs' : ''}`;
  return (
    <button className={cls} onClick={onClick} {...rest}>
      {Icon && <Icon size={size === 'xs' ? 12 : 14} />}
      {children}
    </button>
  );
};

const Pill = ({ variant = 'default', mono, children }) => (
  <span className={`pill ${variant !== 'default' ? variant : ''} ${mono ? 'mono' : ''}`}>{children}</span>
);

const Kbd = ({ children }) => <span className="kbd">{children}</span>;

const StatusDot = ({ status, pulse }) => {
  const cls = status === 'online' || status === 'active' ? 'ok'
    : status === 'idle' || status === 'warn' ? 'warn'
    : status === 'offline' || status === 'err' ? 'err' : '';
  return <span className={`dot ${cls} ${pulse ? 'pulse' : ''}`}></span>;
};

const Presence = ({ clients, max = 3 }) => {
  if (!clients || clients.length === 0) return <span className="subtle" style={{fontFamily:'var(--font-mono)', fontSize:11}}>—</span>;
  const shown = clients.slice(0, max);
  const extra = clients.length - shown.length;
  const colorMap = {
    indigo: { bg: 'oklch(0.58 0.18 270 / 0.2)', fg: 'oklch(0.75 0.16 270)' },
    violet: { bg: 'oklch(0.58 0.2 300 / 0.2)', fg: 'oklch(0.75 0.18 300)' },
    teal: { bg: 'oklch(0.62 0.12 195 / 0.2)', fg: 'oklch(0.75 0.12 195)' },
    amber: { bg: 'oklch(0.78 0.14 75 / 0.2)', fg: 'oklch(0.8 0.14 75)' },
  };
  return (
    <span className="presence">
      {shown.map((c, i) => {
        const col = colorMap[c.color] || colorMap.indigo;
        return <span key={i} className="av" style={{ background: col.bg, color: col.fg }}>{c.name}</span>;
      })}
      {extra > 0 && <span className="av more">+{extra}</span>}
    </span>
  );
};

// "now" | "12s ago" | idle pill
const ActivityCell = ({ session }) => {
  const isActive = session.activity === 'active';
  return (
    <span className="row gap-2">
      <StatusDot status={isActive ? 'active' : 'idle'} pulse={isActive && session.idle < 30} />
      <span className={isActive ? '' : 'muted'} style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>
        {session.lastInput}
      </span>
    </span>
  );
};

Object.assign(window, { Button, Pill, Kbd, StatusDot, Presence, ActivityCell });
