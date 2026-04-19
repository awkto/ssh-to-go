const ic = (d, o = {}) => {
  const {
    size = 16,
    stroke = 1.75,
    fill = 'none',
    vb = '0 0 24 24'
  } = o;
  return React.createElement("svg", {
    width: size,
    height: size,
    viewBox: vb,
    fill: fill,
    stroke: "currentColor",
    strokeWidth: stroke,
    strokeLinecap: "round",
    strokeLinejoin: "round"
  }, d);
};
const IconDashboard = p => ic(React.createElement(React.Fragment, null, React.createElement("rect", {
  x: "3",
  y: "3",
  width: "7",
  height: "9",
  rx: "1.5"
}), React.createElement("rect", {
  x: "14",
  y: "3",
  width: "7",
  height: "5",
  rx: "1.5"
}), React.createElement("rect", {
  x: "14",
  y: "12",
  width: "7",
  height: "9",
  rx: "1.5"
}), React.createElement("rect", {
  x: "3",
  y: "16",
  width: "7",
  height: "5",
  rx: "1.5"
})), p);
const IconTerminal = p => ic(React.createElement(React.Fragment, null, React.createElement("polyline", {
  points: "4 8 8 12 4 16"
}), React.createElement("line", {
  x1: "12",
  y1: "16",
  x2: "20",
  y2: "16"
})), p);
const IconServer = p => ic(React.createElement(React.Fragment, null, React.createElement("rect", {
  x: "3",
  y: "4",
  width: "18",
  height: "7",
  rx: "1.5"
}), React.createElement("rect", {
  x: "3",
  y: "13",
  width: "18",
  height: "7",
  rx: "1.5"
}), React.createElement("line", {
  x1: "7",
  y1: "7.5",
  x2: "7.01",
  y2: "7.5"
}), React.createElement("line", {
  x1: "7",
  y1: "16.5",
  x2: "7.01",
  y2: "16.5"
})), p);
const IconSettings = p => ic(React.createElement(React.Fragment, null, React.createElement("circle", {
  cx: "12",
  cy: "12",
  r: "3"
}), React.createElement("path", {
  d: "M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"
})), p);
const IconGlobe = p => ic(React.createElement(React.Fragment, null, React.createElement("circle", {
  cx: "12",
  cy: "12",
  r: "9"
}), React.createElement("line", {
  x1: "3",
  y1: "12",
  x2: "21",
  y2: "12"
}), React.createElement("path", {
  d: "M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18"
})), p);
const IconCheck = p => ic(React.createElement(React.Fragment, null, React.createElement("polyline", {
  points: "20 6 9 17 4 12"
})), p);
const IconStar = p => ic(React.createElement(React.Fragment, null, React.createElement("polygon", {
  points: "12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"
})), p);
const IconSearch = p => ic(React.createElement(React.Fragment, null, React.createElement("circle", {
  cx: "11",
  cy: "11",
  r: "7"
}), React.createElement("line", {
  x1: "21",
  y1: "21",
  x2: "16.65",
  y2: "16.65"
})), p);
const IconMoon = p => ic(React.createElement(React.Fragment, null, React.createElement("path", {
  d: "M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"
})), p);
const IconSun = p => ic(React.createElement(React.Fragment, null, React.createElement("circle", {
  cx: "12",
  cy: "12",
  r: "4"
}), React.createElement("line", {
  x1: "12",
  y1: "2",
  x2: "12",
  y2: "4"
}), React.createElement("line", {
  x1: "12",
  y1: "20",
  x2: "12",
  y2: "22"
}), React.createElement("line", {
  x1: "4.93",
  y1: "4.93",
  x2: "6.34",
  y2: "6.34"
}), React.createElement("line", {
  x1: "17.66",
  y1: "17.66",
  x2: "19.07",
  y2: "19.07"
}), React.createElement("line", {
  x1: "2",
  y1: "12",
  x2: "4",
  y2: "12"
}), React.createElement("line", {
  x1: "20",
  y1: "12",
  x2: "22",
  y2: "12"
}), React.createElement("line", {
  x1: "4.93",
  y1: "19.07",
  x2: "6.34",
  y2: "17.66"
}), React.createElement("line", {
  x1: "17.66",
  y1: "6.34",
  x2: "19.07",
  y2: "4.93"
})), p);
const IconPlus = p => ic(React.createElement(React.Fragment, null, React.createElement("line", {
  x1: "12",
  y1: "5",
  x2: "12",
  y2: "19"
}), React.createElement("line", {
  x1: "5",
  y1: "12",
  x2: "19",
  y2: "12"
})), p);
const IconFilter = p => ic(React.createElement(React.Fragment, null, React.createElement("polygon", {
  points: "22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"
})), p);
const IconShield = p => ic(React.createElement(React.Fragment, null, React.createElement("path", {
  d: "M12 2L4 6v6c0 5 3.5 9 8 10 4.5-1 8-5 8-10V6l-8-4z"
})), p);
const IconKey = p => ic(React.createElement(React.Fragment, null, React.createElement("circle", {
  cx: "8",
  cy: "15",
  r: "4"
}), React.createElement("line", {
  x1: "10.85",
  y1: "12.15",
  x2: "19",
  y2: "4"
}), React.createElement("line", {
  x1: "18",
  y1: "5",
  x2: "20",
  y2: "7"
}), React.createElement("line", {
  x1: "15",
  y1: "8",
  x2: "17",
  y2: "10"
})), p);
const IconEdit = p => ic(React.createElement(React.Fragment, null, React.createElement("path", {
  d: "M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"
}), React.createElement("path", {
  d: "M18.5 2.5a2.12 2.12 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"
})), p);
const IconClose = p => ic(React.createElement(React.Fragment, null, React.createElement("line", {
  x1: "18",
  y1: "6",
  x2: "6",
  y2: "18"
}), React.createElement("line", {
  x1: "6",
  y1: "6",
  x2: "18",
  y2: "18"
})), p);
const IconMore = p => ic(React.createElement(React.Fragment, null, React.createElement("circle", {
  cx: "12",
  cy: "12",
  r: "1"
}), React.createElement("circle", {
  cx: "19",
  cy: "12",
  r: "1"
}), React.createElement("circle", {
  cx: "5",
  cy: "12",
  r: "1"
})), p);
const IconCopy = p => ic(React.createElement(React.Fragment, null, React.createElement("rect", {
  x: "9",
  y: "9",
  width: "13",
  height: "13",
  rx: "2"
}), React.createElement("path", {
  d: "M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"
})), p);
const IconArrowRight = p => ic(React.createElement(React.Fragment, null, React.createElement("line", {
  x1: "5",
  y1: "12",
  x2: "19",
  y2: "12"
}), React.createElement("polyline", {
  points: "12 5 19 12 12 19"
})), p);
const IconExternalLink = p => ic(React.createElement(React.Fragment, null, React.createElement("path", {
  d: "M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"
}), React.createElement("polyline", {
  points: "15 3 21 3 21 9"
}), React.createElement("line", {
  x1: "10",
  y1: "14",
  x2: "21",
  y2: "3"
})), p);
const IconCpu = p => ic(React.createElement(React.Fragment, null, React.createElement("rect", {
  x: "4",
  y: "4",
  width: "16",
  height: "16",
  rx: "2"
}), React.createElement("rect", {
  x: "9",
  y: "9",
  width: "6",
  height: "6"
}), React.createElement("line", {
  x1: "9",
  y1: "1",
  x2: "9",
  y2: "4"
}), React.createElement("line", {
  x1: "15",
  y1: "1",
  x2: "15",
  y2: "4"
}), React.createElement("line", {
  x1: "9",
  y1: "20",
  x2: "9",
  y2: "23"
}), React.createElement("line", {
  x1: "15",
  y1: "20",
  x2: "15",
  y2: "23"
}), React.createElement("line", {
  x1: "20",
  y1: "9",
  x2: "23",
  y2: "9"
}), React.createElement("line", {
  x1: "20",
  y1: "14",
  x2: "23",
  y2: "14"
}), React.createElement("line", {
  x1: "1",
  y1: "9",
  x2: "4",
  y2: "9"
}), React.createElement("line", {
  x1: "1",
  y1: "14",
  x2: "4",
  y2: "14"
})), p);
const IconActivity = p => ic(React.createElement(React.Fragment, null, React.createElement("polyline", {
  points: "22 12 18 12 15 21 9 3 6 12 2 12"
})), p);
const IconUsers = p => ic(React.createElement(React.Fragment, null, React.createElement("path", {
  d: "M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"
}), React.createElement("circle", {
  cx: "9",
  cy: "7",
  r: "4"
}), React.createElement("path", {
  d: "M23 21v-2a4 4 0 0 0-3-3.87"
}), React.createElement("path", {
  d: "M16 3.13a4 4 0 0 1 0 7.75"
})), p);
const IconQr = p => ic(React.createElement(React.Fragment, null, React.createElement("rect", {
  x: "3",
  y: "3",
  width: "7",
  height: "7",
  rx: "1"
}), React.createElement("rect", {
  x: "14",
  y: "3",
  width: "7",
  height: "7",
  rx: "1"
}), React.createElement("rect", {
  x: "3",
  y: "14",
  width: "7",
  height: "7",
  rx: "1"
}), React.createElement("line", {
  x1: "14",
  y1: "14",
  x2: "14",
  y2: "17"
}), React.createElement("line", {
  x1: "14",
  y1: "20",
  x2: "14",
  y2: "21"
}), React.createElement("line", {
  x1: "17",
  y1: "14",
  x2: "17",
  y2: "14"
}), React.createElement("line", {
  x1: "20",
  y1: "14",
  x2: "21",
  y2: "14"
}), React.createElement("line", {
  x1: "17",
  y1: "17",
  x2: "17",
  y2: "21"
}), React.createElement("line", {
  x1: "20",
  y1: "17",
  x2: "21",
  y2: "17"
}), React.createElement("line", {
  x1: "20",
  y1: "20",
  x2: "21",
  y2: "21"
})), p);
const IconTag = p => ic(React.createElement(React.Fragment, null, React.createElement("path", {
  d: "M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"
}), React.createElement("line", {
  x1: "7",
  y1: "7",
  x2: "7.01",
  y2: "7"
})), p);
const IconPlay = p => ic(React.createElement(React.Fragment, null, React.createElement("polygon", {
  points: "5 3 19 12 5 21 5 3"
})), p);
const IconStop = p => ic(React.createElement(React.Fragment, null, React.createElement("rect", {
  x: "5",
  y: "5",
  width: "14",
  height: "14",
  rx: "2"
})), p);
const IconChevronDown = p => ic(React.createElement(React.Fragment, null, React.createElement("polyline", {
  points: "6 9 12 15 18 9"
})), p);
const IconChevronUp = p => ic(React.createElement(React.Fragment, null, React.createElement("polyline", {
  points: "18 15 12 9 6 15"
})), p);
const IconCommand = p => ic(React.createElement(React.Fragment, null, React.createElement("path", {
  d: "M18 3a3 3 0 0 0-3 3v12a3 3 0 0 0 3 3 3 3 0 0 0 3-3 3 3 0 0 0-3-3H6a3 3 0 0 0-3 3 3 3 0 0 0 3 3 3 3 0 0 0 3-3V6a3 3 0 0 0-3-3 3 3 0 0 0-3 3 3 3 0 0 0 3 3h12a3 3 0 0 0 3-3 3 3 0 0 0-3-3z"
})), p);
const IconRefresh = p => ic(React.createElement(React.Fragment, null, React.createElement("polyline", {
  points: "23 4 23 10 17 10"
}), React.createElement("polyline", {
  points: "1 20 1 14 7 14"
}), React.createElement("path", {
  d: "M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"
})), p);
const IconClock = p => ic(React.createElement(React.Fragment, null, React.createElement("circle", {
  cx: "12",
  cy: "12",
  r: "9"
}), React.createElement("polyline", {
  points: "12 7 12 12 15 14"
})), p);
const IconLink = p => ic(React.createElement(React.Fragment, null, React.createElement("path", {
  d: "M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"
}), React.createElement("path", {
  d: "M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"
})), p);
const IconSend = p => ic(React.createElement(React.Fragment, null, React.createElement("line", {
  x1: "22",
  y1: "2",
  x2: "11",
  y2: "13"
}), React.createElement("polygon", {
  points: "22 2 15 22 11 13 2 9 22 2"
})), p);
const SessIcon = ({
  kind,
  color
}) => {
  const colors = {
    violet: {
      bg: 'oklch(0.58 0.2 300 / 0.14)',
      fg: 'oklch(0.7 0.18 300)'
    },
    teal: {
      bg: 'oklch(0.62 0.12 195 / 0.14)',
      fg: 'oklch(0.72 0.12 195)'
    },
    indigo: {
      bg: 'oklch(0.58 0.18 270 / 0.14)',
      fg: 'oklch(0.7 0.18 270)'
    },
    amber: {
      bg: 'oklch(0.78 0.14 75 / 0.14)',
      fg: 'oklch(0.78 0.14 75)'
    },
    rose: {
      bg: 'oklch(0.68 0.2 25 / 0.14)',
      fg: 'oklch(0.72 0.18 25)'
    }
  };
  const c = colors[color] || colors.indigo;
  const glyphs = {
    chevron: '>_',
    stack: '§',
    cloud: '☁',
    git: 'ᚵ',
    lock: '◉',
    heart: '♥',
    fox: 'ƒ',
    key: '⌘',
    folder: '▣',
    node: '◈',
    terminal: '>_'
  };
  return React.createElement("span", {
    className: "sess-icon",
    style: {
      background: c.bg,
      color: c.fg
    }
  }, glyphs[kind] || '>_');
};
const Sparkline = ({
  data,
  width = 80,
  height = 24,
  color
}) => {
  const max = Math.max(...data, 0.01);
  const min = Math.min(...data);
  const range = max - min || 1;
  const step = width / (data.length - 1);
  const points = data.map((v, i) => [i * step, height - (v - min) / range * (height - 3) - 1.5]);
  const d = points.map((p, i) => `${i === 0 ? 'M' : 'L'}${p[0].toFixed(1)},${p[1].toFixed(1)}`).join(' ');
  const fillD = d + ` L${width},${height} L0,${height} Z`;
  return React.createElement("svg", {
    className: "spark",
    width: width,
    height: height,
    viewBox: `0 0 ${width} ${height}`,
    style: color ? {
      color
    } : {}
  }, React.createElement("path", {
    className: "fill",
    d: fillD
  }), React.createElement("path", {
    className: "stroke",
    d: d
  }));
};
const BrandMark = ({
  size = 30
}) => React.createElement("span", {
  className: "brand-mark",
  style: {
    width: size,
    height: size
  }
}, React.createElement("svg", {
  width: size * 0.6,
  height: size * 0.6,
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: "2.6",
  strokeLinecap: "round",
  strokeLinejoin: "round"
}, React.createElement("polyline", {
  points: "5 7 10 12 5 17"
}), React.createElement("line", {
  x1: "13",
  y1: "17",
  x2: "19",
  y2: "17"
})));
const OsChip = ({
  os,
  kind
}) => React.createElement("span", {
  className: "os-chip"
}, React.createElement("span", {
  className: `os-dot ${kind}`
}), os);
Object.assign(window, {
  IconDashboard,
  IconTerminal,
  IconServer,
  IconSettings,
  IconGlobe,
  IconCheck,
  IconStar,
  IconSearch,
  IconMoon,
  IconSun,
  IconPlus,
  IconFilter,
  IconShield,
  IconKey,
  IconEdit,
  IconClose,
  IconMore,
  IconCopy,
  IconArrowRight,
  IconExternalLink,
  IconCpu,
  IconActivity,
  IconUsers,
  IconQr,
  IconTag,
  IconPlay,
  IconStop,
  IconChevronDown,
  IconChevronUp,
  IconCommand,
  IconRefresh,
  IconClock,
  IconLink,
  IconSend,
  SessIcon,
  Sparkline,
  BrandMark,
  OsChip
});