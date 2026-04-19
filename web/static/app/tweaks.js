const Tweaks = ({
  theme,
  setTheme,
  accent,
  setAccent,
  density,
  setDensity,
  onClose
}) => {
  return React.createElement("div", {
    className: "tweaks-panel"
  }, React.createElement("div", {
    className: "tweaks-head"
  }, React.createElement("h4", null, "Tweaks"), React.createElement("button", {
    className: "icon-btn",
    onClick: onClose,
    style: {
      width: 24,
      height: 24
    }
  }, React.createElement(IconClose, {
    size: 13
  }))), React.createElement("div", {
    className: "tweaks-body"
  }, React.createElement("div", null, React.createElement("div", {
    className: "tweaks-section-label"
  }, "Theme"), React.createElement("div", {
    className: "theme-row"
  }, React.createElement("div", {
    className: `theme-opt ${theme === 'dark' ? 'selected' : ''}`,
    onClick: () => setTheme('dark')
  }, "Dark"), React.createElement("div", {
    className: `theme-opt ${theme === 'light' ? 'selected' : ''}`,
    onClick: () => setTheme('light')
  }, "Light"))), React.createElement("div", null, React.createElement("div", {
    className: "tweaks-section-label"
  }, "Accent"), React.createElement("div", {
    className: "swatch-row"
  }, [{
    id: 'indigo',
    color: 'oklch(0.58 0.18 270)'
  }, {
    id: 'violet',
    color: 'oklch(0.58 0.2 300)'
  }, {
    id: 'teal',
    color: 'oklch(0.62 0.12 195)'
  }].map(s => React.createElement("div", {
    key: s.id,
    className: `swatch ${accent === s.id ? 'selected' : ''}`,
    style: {
      background: s.color
    },
    onClick: () => setAccent(s.id),
    title: s.id
  })))), React.createElement("div", null, React.createElement("div", {
    className: "tweaks-section-label"
  }, "Density"), React.createElement("div", {
    className: "theme-row"
  }, React.createElement("div", {
    className: `theme-opt ${density === 'compact' ? 'selected' : ''}`,
    onClick: () => setDensity('compact')
  }, "Compact"), React.createElement("div", {
    className: `theme-opt ${density === 'balanced' ? 'selected' : ''}`,
    onClick: () => setDensity('balanced')
  }, "Balanced"), React.createElement("div", {
    className: `theme-opt ${density === 'spacious' ? 'selected' : ''}`,
    onClick: () => setDensity('spacious')
  }, "Spacious")))));
};
Object.assign(window, {
  Tweaks
});