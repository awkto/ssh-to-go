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

// Tooltips for icon-only controls, via a `data-tip` attribute.
//
// The native title= tooltip waits about 1.5s before appearing and is easy to
// miss entirely — not good enough for a row of buttons whose only label is a
// glyph. This shows the same text promptly, in the app's own styling.
//
// The popup is a single element on <body> positioned against the viewport,
// not an ::after on the button: the table panel sets `overflow-x: auto`,
// which makes overflow-y compute to auto as well, so an absolutely
// positioned tooltip would be clipped on the first and last rows.
//
// Listeners are delegated from document, so React re-renders need no wiring.
(function () {
  const DELAY = 250;
  let pop = null, timer = null;

  const place = (target) => {
    const tip = target.getAttribute('data-tip');
    if (!tip) return;
    if (!pop) {
      pop = document.createElement('div');
      pop.className = 'tip-pop';
      document.body.appendChild(pop);
    }
    pop.textContent = tip;
    // Measure before showing: visibility:hidden still takes layout.
    pop.classList.remove('show');
    pop.style.visibility = 'hidden';
    pop.style.left = '0px';
    pop.style.top = '0px';
    const btn = target.getBoundingClientRect();
    const box = pop.getBoundingClientRect();
    const left = Math.max(8, Math.min(
      btn.left + btn.width / 2 - box.width / 2,
      window.innerWidth - box.width - 8));
    // Above by default, flipping below when the control is near the top.
    const above = btn.top - box.height - 8;
    pop.style.left = left + 'px';
    pop.style.top = (above < 8 ? btn.bottom + 8 : above) + 'px';
    pop.style.visibility = '';
    pop.classList.add('show');
  };

  const hide = () => { clearTimeout(timer); if (pop) pop.classList.remove('show'); };
  const targetOf = (e) => e.target && e.target.closest && e.target.closest('[data-tip]');

  document.addEventListener('mouseover', (e) => {
    const t = targetOf(e);
    if (!t) return;
    clearTimeout(timer);
    timer = setTimeout(() => place(t), DELAY);
  });
  document.addEventListener('mouseout', (e) => { if (targetOf(e)) hide(); });
  // Keyboard users get it without the delay.
  document.addEventListener('focusin', (e) => { const t = targetOf(e); if (t) place(t); });
  document.addEventListener('focusout', hide);
  // Clicking a button acts on it; leaving the tooltip up would just be litter.
  document.addEventListener('click', hide);
  window.addEventListener('scroll', hide, true);
})();

Object.assign(window, { Button, Pill, Kbd, StatusDot, Presence, ActivityCell });
