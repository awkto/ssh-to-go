const NewSession = ({
  store,
  onClose
}) => {
  const HOSTS = store.hosts;
  const [step, setStep] = React.useState(1);
  const [host, setHost] = React.useState(HOSTS[0] ? HOSTS[0].id : '');
  const [name, setName] = React.useState('');
  const [cwd, setCwd] = React.useState('');
  const [attach, setAttach] = React.useState(true);
  const [busy, setBusy] = React.useState(false);
  const [err, setErr] = React.useState('');
  const [kind, setKind] = React.useState('shell');
  React.useEffect(() => {
    if (!host && HOSTS[0]) setHost(HOSTS[0].id);
  }, [HOSTS.length]);
  React.useEffect(() => {
    setCwd(prev => prev || (name ? `~/projects/${name}` : ''));
  }, [name]);
  const submit = async () => {
    setErr('');
    setBusy(true);
    try {
      const finalName = name || `session-${Math.random().toString(36).slice(2, 7)}`;
      await createSession(host, finalName, cwd || '');
      onClose();
      if (attach) openTerminal(host, finalName);
    } catch (e) {
      setErr(e.message || 'failed');
    } finally {
      setBusy(false);
    }
  };
  const kinds = [{
    id: 'shell',
    label: 'Blank shell',
    sub: '$SHELL'
  }, {
    id: 'claude',
    label: 'Claude Code',
    sub: 'claude'
  }, {
    id: 'custom',
    label: 'Custom command',
    sub: 'run anything'
  }];
  return React.createElement("div", {
    className: "modal-backdrop",
    onClick: onClose
  }, React.createElement("div", {
    className: "modal",
    onClick: e => e.stopPropagation()
  }, React.createElement("div", {
    className: "modal-head"
  }, React.createElement("div", {
    style: {
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      gap: 12
    }
  }, React.createElement("div", null, React.createElement("h3", null, "New session"), React.createElement("p", null, "Spin up a detached tmux session on one of your hosts.")), React.createElement("button", {
    className: "icon-btn",
    onClick: onClose
  }, React.createElement(IconClose, {
    size: 15
  }))), React.createElement("div", {
    className: "stepper mt-3"
  }, React.createElement("div", {
    className: `step ${step === 1 ? 'active' : step > 1 ? 'done' : ''}`
  }, React.createElement("span", {
    className: "step-num"
  }, step > 1 ? React.createElement(IconCheck, {
    size: 12
  }) : 1), " Host"), React.createElement("span", {
    className: "step-divider"
  }), React.createElement("div", {
    className: `step ${step === 2 ? 'active' : step > 2 ? 'done' : ''}`
  }, React.createElement("span", {
    className: "step-num"
  }, step > 2 ? React.createElement(IconCheck, {
    size: 12
  }) : 2), " Command"), React.createElement("span", {
    className: "step-divider"
  }), React.createElement("div", {
    className: `step ${step === 3 ? 'active' : ''}`
  }, React.createElement("span", {
    className: "step-num"
  }, "3"), " Review"))), React.createElement("div", {
    className: "modal-body"
  }, step === 1 && React.createElement(React.Fragment, null, React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Target host"), HOSTS.length === 0 && React.createElement("div", {
    className: "muted",
    style: {
      fontSize: 12.5
    }
  }, "No hosts registered yet \u2014 add one from the Hosts page first."), React.createElement("div", {
    style: {
      display: 'flex',
      flexDirection: 'column',
      gap: 6
    }
  }, HOSTS.map(h => React.createElement("div", {
    key: h.id,
    className: `radio-card ${host === h.id ? 'selected' : ''}`,
    onClick: () => setHost(h.id)
  }, React.createElement(StatusDot, {
    status: h.status === 'online' ? 'active' : 'offline'
  }), React.createElement("div", {
    style: {
      flex: 1
    }
  }, React.createElement("div", {
    className: "radio-title mono"
  }, h.fqdn), React.createElement("div", {
    className: "radio-sub"
  }, h.user, "@", h.fqdn.split(':')[0], " \xB7 ", h.os)), React.createElement(Pill, {
    variant: h.sessions > 0 ? 'accent' : 'default',
    mono: true
  }, h.sessions, " sess"))))), React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Session name"), React.createElement("input", {
    className: "input mono",
    placeholder: kinds.find(k => k.id === kind)?.id || 'my-session',
    value: name,
    onChange: e => setName(e.target.value)
  }), React.createElement("div", {
    className: "hint"
  }, "Auto-generated if left empty. Lowercase, no spaces."))), step === 2 && React.createElement(React.Fragment, null, React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "What to run"), React.createElement("div", {
    className: "radio-grid"
  }, kinds.map(k => React.createElement("div", {
    key: k.id,
    className: `radio-card ${kind === k.id ? 'selected' : ''}`,
    onClick: () => setKind(k.id)
  }, React.createElement("div", {
    style: {
      flex: 1
    }
  }, React.createElement("div", {
    className: "radio-title"
  }, k.label), React.createElement("div", {
    className: "radio-sub"
  }, k.sub)))))), kind === 'custom' && React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Command"), React.createElement("input", {
    className: "input mono",
    placeholder: "e.g. npm run dev"
  })), React.createElement("div", {
    className: "field"
  }, React.createElement("label", null, "Working directory"), React.createElement("input", {
    className: "input mono",
    value: cwd,
    onChange: e => setCwd(e.target.value),
    placeholder: `~/projects/${name || 'new'}`
  })), React.createElement("div", {
    className: "field"
  }, React.createElement("label", {
    className: "checkbox"
  }, React.createElement("input", {
    type: "checkbox",
    checked: attach,
    onChange: e => setAttach(e.target.checked)
  }), " Attach immediately"))), step === 3 && React.createElement("div", null, React.createElement("div", {
    style: {
      background: 'var(--bg-elev-2)',
      border: '1px solid var(--hairline)',
      borderRadius: 8,
      padding: 14,
      marginBottom: 12
    }
  }, React.createElement("div", {
    className: "row gap-3",
    style: {
      marginBottom: 10
    }
  }, React.createElement(SessIcon, {
    kind: "terminal",
    color: "indigo"
  }), React.createElement("span", {
    className: "mono",
    style: {
      fontWeight: 500
    }
  }, name || 'new-session'), React.createElement(Pill, {
    variant: "default",
    mono: true
  }, "will be created")), React.createElement("div", {
    style: {
      fontFamily: 'var(--font-mono)',
      fontSize: 12,
      color: 'var(--fg-muted)',
      lineHeight: 1.8
    }
  }, React.createElement("div", null, React.createElement("span", {
    style: {
      color: 'var(--fg-subtle)'
    }
  }, "host  "), " ", HOSTS.find(h => h.id === host)?.fqdn), React.createElement("div", null, React.createElement("span", {
    style: {
      color: 'var(--fg-subtle)'
    }
  }, "user  "), " ", HOSTS.find(h => h.id === host)?.user), React.createElement("div", null, React.createElement("span", {
    style: {
      color: 'var(--fg-subtle)'
    }
  }, "cwd   "), " ", cwd || `~/projects/${name || 'new'}`))), React.createElement("div", {
    className: "hint",
    style: {
      fontSize: 12,
      color: 'var(--fg-subtle)'
    }
  }, "A detached tmux session will be created on the host. ", attach ? 'Your browser will open the terminal tab on success.' : 'You can attach later from any browser.'), err && React.createElement("div", {
    style: {
      color: 'var(--err)',
      fontSize: 12.5,
      marginTop: 10
    }
  }, err))), React.createElement("div", {
    className: "modal-foot"
  }, step > 1 && React.createElement(Button, {
    variant: "ghost",
    onClick: () => setStep(step - 1)
  }, "Back"), React.createElement("div", {
    style: {
      flex: 1
    }
  }), React.createElement(Button, {
    variant: "ghost",
    onClick: onClose
  }, "Cancel"), step < 3 ? React.createElement(Button, {
    variant: "primary",
    onClick: () => setStep(step + 1),
    disabled: !host
  }, "Continue") : React.createElement(Button, {
    variant: "primary",
    icon: IconPlay,
    onClick: submit,
    disabled: busy || !host
  }, busy ? 'Creating…' : 'Create' + (attach ? ' & attach' : '')))));
};
Object.assign(window, {
  NewSession
});