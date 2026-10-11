const Button = ({
  variant = 'secondary',
  size = 'md',
  children,
  icon: Icon,
  onClick,
  ...rest
}) => {
  const cls = `btn btn-${variant}${size === 'sm' ? ' btn-sm' : ''}${size === 'xs' ? ' btn-xs' : ''}`;
  return React.createElement("button", {
    className: cls,
    onClick: onClick,
    ...rest
  }, Icon && React.createElement(Icon, {
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
const ClientCount = ({
  count
}) => count ? React.createElement("span", {
  className: "mono num",
  style: {
    fontSize: 12.5
  }
}, count) : React.createElement("span", {
  className: "subtle",
  style: {
    fontFamily: 'var(--font-mono)',
    fontSize: 11
  }
}, "\u2014");
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
(function () {
  const DELAY = 250;
  let pop = null,
    timer = null;
  const place = target => {
    const tip = target.getAttribute('data-tip');
    if (!tip) return;
    if (!pop) {
      pop = document.createElement('div');
      pop.className = 'tip-pop';
      document.body.appendChild(pop);
    }
    pop.textContent = tip;
    pop.classList.remove('show');
    pop.style.visibility = 'hidden';
    pop.style.left = '0px';
    pop.style.top = '0px';
    const btn = target.getBoundingClientRect();
    const box = pop.getBoundingClientRect();
    const left = Math.max(8, Math.min(btn.left + btn.width / 2 - box.width / 2, window.innerWidth - box.width - 8));
    const above = btn.top - box.height - 8;
    pop.style.left = left + 'px';
    pop.style.top = (above < 8 ? btn.bottom + 8 : above) + 'px';
    pop.style.visibility = '';
    pop.classList.add('show');
  };
  const hide = () => {
    clearTimeout(timer);
    if (pop) pop.classList.remove('show');
  };
  const targetOf = e => e.target && e.target.closest && e.target.closest('[data-tip]');
  document.addEventListener('mouseover', e => {
    const t = targetOf(e);
    if (!t) return;
    clearTimeout(timer);
    timer = setTimeout(() => place(t), DELAY);
  });
  document.addEventListener('mouseout', e => {
    if (targetOf(e)) hide();
  });
  document.addEventListener('focusin', e => {
    const t = targetOf(e);
    if (t) place(t);
  });
  document.addEventListener('focusout', hide);
  document.addEventListener('click', hide);
  window.addEventListener('scroll', hide, true);
})();
Object.assign(window, {
  Button,
  Pill,
  Kbd,
  StatusDot,
  ClientCount,
  ActivityCell
});
