// ── Tab Title Formatting ──
// Shared by initial page load (template) and rename handler.
window._tabTitleFormat = "host-session";
window.formatTabTitle = function(host, session) {
    switch (window._tabTitleFormat) {
        case "host-only":    return host;
        case "session-only": return session;
        case "session-host": return session + " / " + host;
        default:             return host + " / " + session;
    }
};

// ── Terminal Themes ──
// Well-known terminal color schemes with full ANSI 16-color palettes.
var TERMINAL_THEMES = {
    "default": {
        name: "Default",
        theme: {
            background: "#1a1a2e", foreground: "#e0e0e8", cursor: "#7c83ff",
            selectionBackground: "#3a3a5a",
            black: "#1a1a2e", red: "#ff5555", green: "#50fa7b", yellow: "#f1fa8c",
            blue: "#7c83ff", magenta: "#ff79c6", cyan: "#8be9fd", white: "#e0e0e8",
            brightBlack: "#4a4a6a", brightRed: "#ff6e6e", brightGreen: "#69ff94",
            brightYellow: "#ffffa5", brightBlue: "#9a9fff", brightMagenta: "#ff92d0",
            brightCyan: "#a4edff", brightWhite: "#ffffff",
        },
    },
    "dracula": {
        name: "Dracula",
        theme: {
            background: "#282a36", foreground: "#f8f8f2", cursor: "#f8f8f2",
            selectionBackground: "#44475a",
            black: "#21222c", red: "#ff5555", green: "#50fa7b", yellow: "#f1fa8c",
            blue: "#bd93f9", magenta: "#ff79c6", cyan: "#8be9fd", white: "#f8f8f2",
            brightBlack: "#6272a4", brightRed: "#ff6e6e", brightGreen: "#69ff94",
            brightYellow: "#ffffa5", brightBlue: "#d6acff", brightMagenta: "#ff92df",
            brightCyan: "#a4ffff", brightWhite: "#ffffff",
        },
    },
    "monokai": {
        name: "Monokai",
        theme: {
            background: "#272822", foreground: "#f8f8f2", cursor: "#f8f8f0",
            selectionBackground: "#49483e",
            black: "#272822", red: "#f92672", green: "#a6e22e", yellow: "#f4bf75",
            blue: "#66d9ef", magenta: "#ae81ff", cyan: "#a1efe4", white: "#f8f8f2",
            brightBlack: "#75715e", brightRed: "#f92672", brightGreen: "#a6e22e",
            brightYellow: "#f4bf75", brightBlue: "#66d9ef", brightMagenta: "#ae81ff",
            brightCyan: "#a1efe4", brightWhite: "#f9f8f5",
        },
    },
    "nord": {
        name: "Nord",
        theme: {
            background: "#2e3440", foreground: "#d8dee9", cursor: "#d8dee9",
            selectionBackground: "#434c5e",
            black: "#3b4252", red: "#bf616a", green: "#a3be8c", yellow: "#ebcb8b",
            blue: "#81a1c1", magenta: "#b48ead", cyan: "#88c0d0", white: "#e5e9f0",
            brightBlack: "#4c566a", brightRed: "#bf616a", brightGreen: "#a3be8c",
            brightYellow: "#ebcb8b", brightBlue: "#81a1c1", brightMagenta: "#b48ead",
            brightCyan: "#8fbcbb", brightWhite: "#eceff4",
        },
    },
    "solarized-dark": {
        name: "Solarized Dark",
        theme: {
            background: "#002b36", foreground: "#839496", cursor: "#839496",
            selectionBackground: "#073642",
            black: "#073642", red: "#dc322f", green: "#859900", yellow: "#b58900",
            blue: "#268bd2", magenta: "#d33682", cyan: "#2aa198", white: "#eee8d5",
            brightBlack: "#586e75", brightRed: "#cb4b16", brightGreen: "#586e75",
            brightYellow: "#657b83", brightBlue: "#839496", brightMagenta: "#6c71c4",
            brightCyan: "#93a1a1", brightWhite: "#fdf6e3",
        },
    },
    "solarized-light": {
        name: "Solarized Light",
        theme: {
            background: "#fdf6e3", foreground: "#657b83", cursor: "#586e75",
            selectionBackground: "#eee8d5",
            black: "#073642", red: "#dc322f", green: "#859900", yellow: "#b58900",
            blue: "#268bd2", magenta: "#d33682", cyan: "#2aa198", white: "#eee8d5",
            brightBlack: "#002b36", brightRed: "#cb4b16", brightGreen: "#586e75",
            brightYellow: "#657b83", brightBlue: "#839496", brightMagenta: "#6c71c4",
            brightCyan: "#93a1a1", brightWhite: "#fdf6e3",
        },
    },
    "one-dark": {
        name: "One Dark",
        theme: {
            background: "#282c34", foreground: "#abb2bf", cursor: "#528bff",
            selectionBackground: "#3e4451",
            black: "#282c34", red: "#e06c75", green: "#98c379", yellow: "#e5c07b",
            blue: "#61afef", magenta: "#c678dd", cyan: "#56b6c2", white: "#abb2bf",
            brightBlack: "#5c6370", brightRed: "#e06c75", brightGreen: "#98c379",
            brightYellow: "#e5c07b", brightBlue: "#61afef", brightMagenta: "#c678dd",
            brightCyan: "#56b6c2", brightWhite: "#ffffff",
        },
    },
    "gruvbox-dark": {
        name: "Gruvbox Dark",
        theme: {
            background: "#282828", foreground: "#ebdbb2", cursor: "#ebdbb2",
            selectionBackground: "#3c3836",
            black: "#282828", red: "#cc241d", green: "#98971a", yellow: "#d79921",
            blue: "#458588", magenta: "#b16286", cyan: "#689d6a", white: "#a89984",
            brightBlack: "#928374", brightRed: "#fb4934", brightGreen: "#b8bb26",
            brightYellow: "#fabd2f", brightBlue: "#83a598", brightMagenta: "#d3869b",
            brightCyan: "#8ec07c", brightWhite: "#ebdbb2",
        },
    },
    "tokyo-night": {
        name: "Tokyo Night",
        theme: {
            background: "#1a1b26", foreground: "#a9b1d6", cursor: "#c0caf5",
            selectionBackground: "#33467c",
            black: "#15161e", red: "#f7768e", green: "#9ece6a", yellow: "#e0af68",
            blue: "#7aa2f7", magenta: "#bb9af7", cyan: "#7dcfff", white: "#a9b1d6",
            brightBlack: "#414868", brightRed: "#f7768e", brightGreen: "#9ece6a",
            brightYellow: "#e0af68", brightBlue: "#7aa2f7", brightMagenta: "#bb9af7",
            brightCyan: "#7dcfff", brightWhite: "#c0caf5",
        },
    },
    "catppuccin-mocha": {
        name: "Catppuccin Mocha",
        theme: {
            background: "#1e1e2e", foreground: "#cdd6f4", cursor: "#f5e0dc",
            selectionBackground: "#45475a",
            black: "#45475a", red: "#f38ba8", green: "#a6e3a1", yellow: "#f9e2af",
            blue: "#89b4fa", magenta: "#f5c2e7", cyan: "#94e2d5", white: "#bac2de",
            brightBlack: "#585b70", brightRed: "#f38ba8", brightGreen: "#a6e3a1",
            brightYellow: "#f9e2af", brightBlue: "#89b4fa", brightMagenta: "#f5c2e7",
            brightCyan: "#94e2d5", brightWhite: "#a6adc8",
        },
    },
    "rose-pine": {
        name: "Rose Pine",
        theme: {
            background: "#191724", foreground: "#e0def4", cursor: "#524f67",
            selectionBackground: "#2a283e",
            black: "#26233a", red: "#eb6f92", green: "#31748f", yellow: "#f6c177",
            blue: "#9ccfd8", magenta: "#c4a7e7", cyan: "#ebbcba", white: "#e0def4",
            brightBlack: "#6e6a86", brightRed: "#eb6f92", brightGreen: "#31748f",
            brightYellow: "#f6c177", brightBlue: "#9ccfd8", brightMagenta: "#c4a7e7",
            brightCyan: "#ebbcba", brightWhite: "#e0def4",
        },
    },
    "github-dark": {
        name: "GitHub Dark",
        theme: {
            background: "#0d1117", foreground: "#c9d1d9", cursor: "#c9d1d9",
            selectionBackground: "#264f78",
            black: "#484f58", red: "#ff7b72", green: "#3fb950", yellow: "#d29922",
            blue: "#58a6ff", magenta: "#bc8cff", cyan: "#39c5cf", white: "#b1bac4",
            brightBlack: "#6e7681", brightRed: "#ffa198", brightGreen: "#56d364",
            brightYellow: "#e3b341", brightBlue: "#79c0ff", brightMagenta: "#d2a8ff",
            brightCyan: "#56d4dd", brightWhite: "#f0f6fc",
        },
    },
    "github-light": {
        name: "GitHub Light",
        theme: {
            background: "#ffffff", foreground: "#24292f", cursor: "#044289",
            selectionBackground: "#accef7",
            black: "#24292f", red: "#cf222e", green: "#116329", yellow: "#4d2d00",
            blue: "#0969da", magenta: "#8250df", cyan: "#1b7c83", white: "#6e7781",
            brightBlack: "#57606a", brightRed: "#a40e26", brightGreen: "#1a7f37",
            brightYellow: "#633c01", brightBlue: "#218bff", brightMagenta: "#a475f9",
            brightCyan: "#3192aa", brightWhite: "#8c959f",
        },
    },
    "one-light": {
        name: "One Light",
        theme: {
            background: "#fafafa", foreground: "#383a42", cursor: "#526fff",
            selectionBackground: "#e5e5e6",
            black: "#383a42", red: "#e45649", green: "#50a14f", yellow: "#c18401",
            blue: "#4078f2", magenta: "#a626a4", cyan: "#0184bc", white: "#a0a1a7",
            brightBlack: "#696c77", brightRed: "#e06c75", brightGreen: "#98c379",
            brightYellow: "#e5c07b", brightBlue: "#61afef", brightMagenta: "#c678dd",
            brightCyan: "#56b6c2", brightWhite: "#ffffff",
        },
    },
    "catppuccin-latte": {
        name: "Catppuccin Latte",
        theme: {
            background: "#eff1f5", foreground: "#4c4f69", cursor: "#dc8a78",
            selectionBackground: "#ccd0da",
            black: "#5c5f77", red: "#d20f39", green: "#40a02b", yellow: "#df8e1d",
            blue: "#1e66f5", magenta: "#ea76cb", cyan: "#179299", white: "#acb0be",
            brightBlack: "#6c6f85", brightRed: "#d20f39", brightGreen: "#40a02b",
            brightYellow: "#df8e1d", brightBlue: "#1e66f5", brightMagenta: "#ea76cb",
            brightCyan: "#179299", brightWhite: "#bcc0cc",
        },
    },
    "gruvbox-light": {
        name: "Gruvbox Light",
        theme: {
            background: "#fbf1c7", foreground: "#3c3836", cursor: "#3c3836",
            selectionBackground: "#ebdbb2",
            black: "#fbf1c7", red: "#cc241d", green: "#98971a", yellow: "#d79921",
            blue: "#458588", magenta: "#b16286", cyan: "#689d6a", white: "#7c6f64",
            brightBlack: "#928374", brightRed: "#9d0006", brightGreen: "#79740e",
            brightYellow: "#b57614", brightBlue: "#076678", brightMagenta: "#8f3f71",
            brightCyan: "#427b58", brightWhite: "#3c3836",
        },
    },
    "rose-pine-dawn": {
        name: "Rose Pine Dawn",
        theme: {
            background: "#faf4ed", foreground: "#575279", cursor: "#9893a5",
            selectionBackground: "#dfdad9",
            black: "#f2e9e1", red: "#b4637a", green: "#286983", yellow: "#ea9d34",
            blue: "#56949f", magenta: "#907aa9", cyan: "#d7827e", white: "#575279",
            brightBlack: "#9893a5", brightRed: "#b4637a", brightGreen: "#286983",
            brightYellow: "#ea9d34", brightBlue: "#56949f", brightMagenta: "#907aa9",
            brightCyan: "#d7827e", brightWhite: "#575279",
        },
    },
    "tokyo-night-light": {
        name: "Tokyo Night Light",
        theme: {
            background: "#d5d6db", foreground: "#343b58", cursor: "#343b58",
            selectionBackground: "#c4c8da",
            black: "#0f0f14", red: "#8c4351", green: "#33635c", yellow: "#8f5e15",
            blue: "#34548a", magenta: "#5a4a78", cyan: "#0f4b6e", white: "#343b58",
            brightBlack: "#9699a3", brightRed: "#8c4351", brightGreen: "#33635c",
            brightYellow: "#8f5e15", brightBlue: "#34548a", brightMagenta: "#5a4a78",
            brightCyan: "#0f4b6e", brightWhite: "#343b58",
        },
    },
    "sepia": {
        name: "Sepia",
        theme: {
            background: "#f4ecd8", foreground: "#5b4636", cursor: "#5b4636",
            selectionBackground: "#d6c9a8",
            black: "#5b4636", red: "#a03b2e", green: "#5a7e3e", yellow: "#9b7721",
            blue: "#4a6fa5", magenta: "#8b5e83", cyan: "#3e8a75", white: "#c4b59a",
            brightBlack: "#7a6652", brightRed: "#c05040", brightGreen: "#6f9a4e",
            brightYellow: "#b5912e", brightBlue: "#5c85bf", brightMagenta: "#a57199",
            brightCyan: "#4ea38c", brightWhite: "#f4ecd8",
        },
    },
    "everforest-light": {
        name: "Everforest Light",
        theme: {
            background: "#fdf6e3", foreground: "#5c6a72", cursor: "#5c6a72",
            selectionBackground: "#e0dcc7",
            black: "#5c6a72", red: "#f85552", green: "#8da101", yellow: "#dfa000",
            blue: "#3a94c5", magenta: "#df69ba", cyan: "#35a77c", white: "#ddd8be",
            brightBlack: "#708089", brightRed: "#f85552", brightGreen: "#8da101",
            brightYellow: "#dfa000", brightBlue: "#3a94c5", brightMagenta: "#df69ba",
            brightCyan: "#35a77c", brightWhite: "#fdf6e3",
        },
    },
    "kanagawa-lotus": {
        name: "Kanagawa Lotus",
        theme: {
            background: "#f2ecbc", foreground: "#545464", cursor: "#43436c",
            selectionBackground: "#d5cea3",
            black: "#545464", red: "#c84053", green: "#6f894e", yellow: "#a07a30",
            blue: "#4d699b", magenta: "#b35b79", cyan: "#597b75", white: "#dcd7ba",
            brightBlack: "#706e61", brightRed: "#d7474b", brightGreen: "#87a764",
            brightYellow: "#c4a83f", brightBlue: "#6693bf", brightMagenta: "#c87b9d",
            brightCyan: "#6e9a8d", brightWhite: "#f2ecbc",
        },
    },
    "ayu-light": {
        name: "Ayu Light",
        theme: {
            background: "#fafafa", foreground: "#575f66", cursor: "#ff6a00",
            selectionBackground: "#d1e4f4",
            black: "#000000", red: "#f51818", green: "#86b300", yellow: "#f2ae49",
            blue: "#36a3d9", magenta: "#a37acc", cyan: "#4dbf99", white: "#828c99",
            brightBlack: "#575f66", brightRed: "#f07171", brightGreen: "#99c425",
            brightYellow: "#f2ae49", brightBlue: "#55b4d4", brightMagenta: "#c47ade",
            brightCyan: "#6cbf8b", brightWhite: "#fafafa",
        },
    },
    "alabaster": {
        name: "Alabaster",
        theme: {
            background: "#f7f7f7", foreground: "#434343", cursor: "#007acc",
            selectionBackground: "#bfdbfe",
            black: "#000000", red: "#aa3731", green: "#448c27", yellow: "#cb9000",
            blue: "#325cc0", magenta: "#7a3e9d", cyan: "#0083b2", white: "#bbbbbb",
            brightBlack: "#777777", brightRed: "#f05050", brightGreen: "#60cb00",
            brightYellow: "#ffbc5d", brightBlue: "#007acc", brightMagenta: "#e64ce6",
            brightCyan: "#00aacb", brightWhite: "#f7f7f7",
        },
    },
};

// ── Terminal Fonts ──
// Self-hosted woff2 faces (web/static/vendor/fonts/) plus the original
// system-font stack as the default. `css` is the quoted family name used in
// document.fonts.load(); `weights` lists the weights that actually shipped —
// the picker greys nothing out (browsers nearest-match missing weights) but
// it's documentation for anyone adding faces.
var TERMINAL_FONTS = [
    { id: "default",         name: "System Default",  css: null,               family: "'SF Mono', 'Fira Code', 'Cascadia Code', monospace", weights: [400, 700] },
    { id: "jetbrains-mono",  name: "JetBrains Mono",  css: "'JetBrains Mono'", family: "'JetBrains Mono', monospace",  weights: [300, 400, 500, 700] },
    { id: "fira-code",       name: "Fira Code",       css: "'Fira Code'",      family: "'Fira Code', monospace",       weights: [300, 400, 500, 700] },
    { id: "cascadia-code",   name: "Cascadia Code",   css: "'Cascadia Code'",  family: "'Cascadia Code', monospace",   weights: [300, 400, 500, 700] },
    { id: "source-code-pro", name: "Source Code Pro", css: "'Source Code Pro'",family: "'Source Code Pro', monospace", weights: [300, 400, 500, 700] },
    { id: "ibm-plex-mono",   name: "IBM Plex Mono",   css: "'IBM Plex Mono'",  family: "'IBM Plex Mono', monospace",   weights: [300, 400, 500, 700] },
    { id: "roboto-mono",     name: "Roboto Mono",     css: "'Roboto Mono'",    family: "'Roboto Mono', monospace",     weights: [300, 400, 500, 700] },
    { id: "inconsolata",     name: "Inconsolata",     css: "'Inconsolata'",    family: "'Inconsolata', monospace",     weights: [300, 400, 500, 700] },
    { id: "hack",            name: "Hack",            css: "'Hack'",           family: "'Hack', monospace",            weights: [400, 700] },
    { id: "ubuntu-mono",     name: "Ubuntu Mono",     css: "'Ubuntu Mono'",    family: "'Ubuntu Mono', monospace",     weights: [400, 700] },
    { id: "dejavu-mono",     name: "DejaVu Mono",     css: "'DejaVu Mono'",    family: "'DejaVu Mono', monospace",     weights: [400, 700] },
    { id: "anonymous-pro",   name: "Anonymous Pro",   css: "'Anonymous Pro'",  family: "'Anonymous Pro', monospace",   weights: [400, 700] },
    { id: "space-mono",      name: "Space Mono",      css: "'Space Mono'",     family: "'Space Mono', monospace",      weights: [400, 700] },
    { id: "cousine",         name: "Cousine",         css: "'Cousine'",        family: "'Cousine', monospace",         weights: [400, 700] },
];

function initTerminal(host, session) {
    const DEFAULT_FONT_SIZE = 14;
    const MIN_FONT_SIZE = 8;
    const MAX_FONT_SIZE = 32;
    const savedFontSize = parseInt(localStorage.getItem("term-font-size"), 10) || DEFAULT_FONT_SIZE;

    // Saved font family + weight (global preference, like font size). The
    // terminal is constructed on the default stack and the saved face is
    // applied asynchronously below once its woff2 has loaded — constructing
    // straight onto an unloaded web font would measure the fallback.
    var currentFontId = localStorage.getItem("term-font-family") || "default";
    var currentFontWeight = parseInt(localStorage.getItem("term-font-weight"), 10) || 400;

    // Determine initial theme — check localStorage for a session-specific override,
    // then fall back to "default". The server-saved theme is applied asynchronously.
    var currentThemeId = localStorage.getItem("term-theme:" + host + ":" + session) || "default";
    var initialTheme = (TERMINAL_THEMES[currentThemeId] || TERMINAL_THEMES["default"]).theme;

    const term = new Terminal({
        cursorBlink: true,
        fontSize: savedFontSize,
        fontFamily: "'SF Mono', 'Fira Code', 'Cascadia Code', monospace",
        rightClickSelectsWord: true,
        // Server-configured scrollback depth (settings.scrollback_lines),
        // sized to match the tmux history-limit so the deep prefill isn't
        // truncated. Falls back to 5000 if the page didn't inject it.
        scrollback: (window._scrollbackLines || 5000),
        theme: initialTheme,
    });

    const fitAddon = new FitAddon.FitAddon();
    term.loadAddon(fitAddon);

    const webLinksAddon = new WebLinksAddon.WebLinksAddon();
    term.loadAddon(webLinksAddon);

    const container = document.getElementById("terminal");
    term.open(container);
    fitAddon.fit();

    // Resize-war handling (issue #80): when another tmux client (a native
    // handoff terminal, another browser tab) resizes the window, the relay
    // sends a "winsize" message and we force the xterm grid to the window's
    // dimensions so the raw output stream keeps rendering correctly —
    // scrolling the overflow instead of turning to soup. While overridden,
    // browser resizes only report the would-be fit size to the server (so
    // tmux still knows our real size) without touching the grid; when the
    // window comes back to our size the override lifts and fit resumes.
    let sizeOverride = false;
    let applyingWinsize = false;
    function sendFitSize() {
        const prop = fitAddon.proposeDimensions();
        if (prop && activeWs && activeWs.readyState === WebSocket.OPEN) {
            activeWs.send(JSON.stringify({ type: "resize", cols: prop.cols, rows: prop.rows }));
        }
    }
    function refit() {
        if (sizeOverride) { sendFitSize(); return; }
        fitAddon.fit();
    }
    function applyWinsize(cols, rows) {
        const prop = fitAddon.proposeDimensions();
        const matchesFit = prop && prop.cols === cols && prop.rows === rows;
        // Erase the frame BEFORE resizing the grid: it was drawn for the old
        // width, and resizing first would reflow it into wrapped junk that
        // survives above the server's repaint. The write callback sequences
        // the resize after the erase lands; the repaint bytes queue behind.
        term.write("\x1b[2J\x1b[H", function () {
            applyingWinsize = true;
            try {
                if (matchesFit) {
                    sizeOverride = false;
                    container.style.overflow = "";
                    fitAddon.fit();
                } else {
                    sizeOverride = true;
                    container.style.overflow = "auto";
                    term.resize(cols, rows);
                }
            } finally {
                applyingWinsize = false;
            }
        });
    }
    function clearWinsizeOverride() {
        sizeOverride = false;
        container.style.overflow = "";
    }

    // On touch devices, .xterm-screen has pointer-events:none so finger
    // drags pass through to the sibling .xterm-viewport and the browser
    // scrolls it natively. Two complications we handle here:
    //
    // 1) xterm.js binds its own touchmove handler to the .xterm element
    //    that manually sets viewport.scrollTop on each move. That race
    //    against the browser's native scroll on .xterm-viewport seems to
    //    kill momentum on Android Chrome/Firefox. stopPropagation in the
    //    capture phase on #terminal prevents xterm's listener (on the
    //    inner .xterm node) from ever firing, leaving the browser as the
    //    sole driver of the gesture.
    //
    // 2) Because the canvas no longer receives the tap, xterm.js never
    //    focuses its hidden helper textarea — soft-keyboard keystrokes
    //    have no input target. Detect a brief stationary tap on
    //    touchend and call term.focus() to fix that.
    (function () {
        container.addEventListener("touchstart", function (e) {
            e.stopPropagation();
        }, { capture: true, passive: true });
        container.addEventListener("touchmove", function (e) {
            e.stopPropagation();
        }, { capture: true, passive: true });

        let startX = 0, startY = 0, startT = 0;
        container.addEventListener("touchstart", function (e) {
            if (e.touches.length !== 1) return;
            startX = e.touches[0].pageX;
            startY = e.touches[0].pageY;
            startT = performance.now();
        }, { passive: true });
        container.addEventListener("touchend", function (e) {
            if (e.changedTouches.length !== 1) return;
            const t = e.changedTouches[0];
            const dx = Math.abs(t.pageX - startX);
            const dy = Math.abs(t.pageY - startY);
            const dt = performance.now() - startT;
            if (dx < 10 && dy < 10 && dt < 300) {
                // Tap (not a swipe). Focus the helper textarea so Android
                // raises the soft keyboard and routes IME keys to xterm's
                // input listener.
                const ta = container.querySelector(".xterm-helper-textarea");
                if (ta) ta.focus();
                else term.focus();
            }
        }, { passive: true });
    })();

    const statusEl = document.getElementById("ws-status");

    // Shared reference to the active WebSocket so button handlers can send data.
    let activeWs = null;

    // Disposables for event listeners that must be cleaned up on reconnect
    let onDataDisposable = null;
    let onResizeDisposable = null;

    // TTY path of our relay's tmux client, used to exclude ourselves when kicking others
    let myTTY = "";
    // Set to true when server sends a "kicked" message — prevents auto-reconnect
    let kicked = false;
    // Reason this session was deliberately ended ("killed" / "offloaded"),
    // from the server's "terminated" message, the 4002 close code, or our own
    // Kill/Offload buttons. Any of them suppresses the auto-reconnect below —
    // that reconnect runs `tmux new-session -A`, which would recreate the
    // session three seconds after you ended it.
    let terminated = "";
    let endStateDrawn = false;

    // Explicit end state in place of the terminal, so an ended session reads
    // as ended rather than as a connection that silently stopped coming back.
    function drawEndState(reason) {
        if (endStateDrawn) return;
        endStateDrawn = true;
        statusEl.className = "status disconnected";
        var offloaded = reason === "offloaded";
        var title = offloaded ? "Session Offloaded" : "Session Killed";
        var detail = offloaded
            ? "tmux stopped, but ssh-to-go kept it — resume it from the dashboard."
            : "The tmux session is gone and ssh-to-go is no longer tracking it.";
        term.write("\r\n\r\n\x1b[1;93m─── " + title + " ───\x1b[0m\r\n" +
                   "\x1b[90m" + detail + "\x1b[0m\r\n");
    }

    function sendBytes(bytes) {
        if (activeWs && activeWs.readyState === WebSocket.OPEN) {
            activeWs.send(new Uint8Array(bytes));
        }
    }

    // Relay pipeline: "control" (default) attaches via tmux control mode —
    // history comes from capture-pane over the control channel and live
    // output as %output events, so nothing ever repaints and the local
    // scrollback stays accurate. Add ?mode=legacy to the page URL to fall
    // back to the classic PTY attach pipeline for comparison.
    const relayMode = new URLSearchParams(location.search).get("mode") === "legacy" ? "" : "control";
    // App/passthrough mode (spike): ?mouse=on passes through to the relay. With
    // ?mode=legacy&mouse=on the relay does a plain PTY attach with NO alt-screen
    // / mouse-tracking stripping, so xterm.js is a real terminal — the clean host
    // for Claude Code's fullscreen (GPU/no-flicker) renderer, which owns the
    // alt-screen and uses DECSET 2026 sync (which the stripper never touched).
    // Defaults to "off" (today's scroll-first behaviour) — opt-in only.
    const mouseParam = new URLSearchParams(location.search).get("mouse") === "on" ? "on" : "off";

    function connect() {
        const proto = location.protocol === "https:" ? "wss:" : "ws:";
        // mouse=off opts into the relay's native-client pipeline: tmux history
        // is prefilled via capture-pane, and mouse-tracking + alt-screen escape
        // sequences are stripped server-side. xterm.js then drives smooth
        // scrolling locally against its own scrollback instead of tmux's
        // row-quantised copy-mode wheel handler.
        const url = `${proto}//${location.host}/ws/${encodeURIComponent(host)}/${encodeURIComponent(session)}?mouse=${mouseParam}${relayMode ? "&mode=" + relayMode : ""}`;
        const ws = new WebSocket(url);
        activeWs = ws;
        ws.binaryType = "arraybuffer";

        // ONE decoder for the whole connection, always used with
        // {stream:true}. The relay forwards raw pty bytes and frames break
        // wherever the 32KB read did, so a multi-byte rune is regularly
        // split across two frames. Decoding each frame in isolation turned
        // the truncated tail and the orphan continuation bytes into U+FFFD
        // each — a 1-cell box-drawing char became 2+ cells, shifting the
        // rest of the line sideways. A streaming decoder holds the partial
        // sequence back until the bytes that complete it arrive.
        const decoder = new TextDecoder("utf-8");

        ws.onopen = function () {
            statusEl.className = "status connected";
            if (relayMode === "control") {
                // The server resends the full tmux history on every
                // (re)connect; start from a clean buffer so it never stacks.
                term.reset();
            }
            // A fresh relay knows nothing of a previous override; start from
            // our own fit so both sides agree on the size again.
            if (sizeOverride) {
                clearWinsizeOverride();
                fitAddon.fit();
            }
            // Send initial size
            ws.send(JSON.stringify({
                type: "resize",
                cols: term.cols,
                rows: term.rows,
            }));
        };

        ws.onmessage = function (e) {
            if (e.data instanceof ArrayBuffer) {
                // Filter mouse mode sequences from binary data
                var bytes = new Uint8Array(e.data);
                var str = decoder.decode(bytes, { stream: true });
                var filtered = str.replace(mouseSeqRegex, "");
                if (filtered.length > 0) {
                    term.write(filtered);
                }
            } else {
                // Check for control messages (resize acks, tty, etc)
                try {
                    const msg = JSON.parse(e.data);
                    if (msg.type === "resize") return;
                    if (msg.type === "winsize") { applyWinsize(msg.cols, msg.rows); return; }
                    if (msg.type === "tty") { myTTY = msg.tty; return; }
                    if (msg.type === "kicked") { kicked = true; return; }
                    // Sent while the socket is still healthy, just before
                    // the server tears tmux down on purpose.
                    if (msg.type === "terminated") { terminated = msg.reason || "killed"; return; }
                } catch (_) {}
                term.write(e.data.replace(mouseSeqRegex, ""));
            }
        };

        ws.onclose = function (e) {
            statusEl.className = "status disconnected";
            // Deliberately killed or offloaded (code 4002, or the control
            // message that preceded it). No reconnect: this is the whole
            // fix for sessions coming back from the dead.
            if (terminated || e.code === 4002) {
                drawEndState(terminated || e.reason || "killed");
                return;
            }
            // Code 4000 = session ended normally (killed/destroyed)
            if (e.code === 4000) {
                term.write("\r\n\x1b[93m--- session ended ---\x1b[0m\r\n");
                return;
            }
            // Kicked by another client (via control message or close code 4001)
            if (kicked || e.code === 4001) {
                kicked = false;
                term.write("\r\n\x1b[93m--- disconnected by another client ---\x1b[0m\r\n");
                return;
            }
            term.write("\r\n\x1b[90m--- disconnected, reconnecting in 3s ---\x1b[0m\r\n");
            setTimeout(connect, 3000);
        };

        ws.onerror = function () {
            ws.close();
        };

        // Dispose old listeners before registering new ones to prevent
        // accumulating handlers across reconnects.
        if (onDataDisposable) { onDataDisposable.dispose(); }
        if (onResizeDisposable) { onResizeDisposable.dispose(); }

        // Terminal input -> WebSocket. Strip terminal-report auto-replies
        // (issue #79): xterm.js answers queries that reach it — DA1/DA2
        // "who are you", DSR status, CPR cursor position — and query bytes
        // can arrive embedded in captured history or repaint frames. tmux
        // already answered the real query; a second reply typed into the
        // pane surfaces as "?1;2c"-style garbage at the prompt. No keyboard
        // produces these byte shapes, so stripping them is safe.
        onDataDisposable = term.onData(function (data) {
            data = data.replace(reportReplyRegex, "");
            if (data.length === 0) return;
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(new TextEncoder().encode(data));
            }
        });

        // Handle resize. Server-driven winsize overrides must not echo back
        // as our declared size — the server tracks the browser's REAL fit
        // size so it can tell when the window returns to it.
        onResizeDisposable = term.onResize(function (size) {
            if (applyingWinsize) return;
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: "resize",
                    cols: size.cols,
                    rows: size.rows,
                }));
            }
        });

    }

    // Handoff button — icon-only now, so success is a transient check mark
    // in place of the glyph instead of a text swap.
    document.getElementById("handoff-btn").addEventListener("click", async function () {
        const btn = this;
        try {
            const res = await fetch(`/api/hosts/${encodeURIComponent(host)}/sessions/${encodeURIComponent(session)}/handoff`);
            const data = await res.json();
            await clipCopy(data.command);
            const original = btn.innerHTML;
            btn.innerHTML = '<svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" ' +
                'stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>';
            btn.classList.add("ok");
            btn.setAttribute("data-tip", "SSH command copied");
            setTimeout(function () {
                btn.innerHTML = original;
                btn.classList.remove("ok");
                btn.setAttribute("data-tip", "Copy the SSH command to attach from your own terminal");
            }, 2000);
        } catch (e) {
            alert("Failed to copy: " + e.message);
        }
    });

    // Detach/kick other clients
    document.getElementById("detach-btn").addEventListener("click", async function () {
        const btn = this;
        btn.disabled = true;
        try {
            const body = myTTY ? JSON.stringify({ exclude_tty: myTTY }) : "{}";
            const res = await fetch(`/api/hosts/${encodeURIComponent(host)}/sessions/${encodeURIComponent(session)}/detach-clients`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: body,
            });
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();
            btn.textContent = data.detached > 0 ? `Kicked ${data.detached}` : "No others";
            setTimeout(() => { btn.textContent = "Kick Other Clients"; btn.disabled = false; }, 2000);
        } catch (e) {
            alert("Detach failed: " + e.message);
            btn.textContent = "Kick Other Clients";
            btn.disabled = false;
        }
    });

    // Rename button
    document.getElementById("rename-btn").addEventListener("click", async function () {
        const newName = prompt(`Rename session "${session}":`, session);
        if (!newName || newName === session) return;
        try {
            const res = await fetch(`/api/hosts/${encodeURIComponent(host)}/sessions/${encodeURIComponent(session)}`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ new_name: newName }),
            });
            if (!res.ok) throw new Error(await res.text());
            // Migrate icon cache entry to new name
            if (window.renameSessionIcon) window.renameSessionIcon(host, session, newName);
            // Update the page title, label, and URL
            session = newName;
            var title = window.formatTabTitle(host, newName);
            document.getElementById("session-label").textContent = title;
            document.title = title;
            window.history.replaceState(null, "", `/terminal/${encodeURIComponent(host)}/${encodeURIComponent(newName)}`);
            term.focus();
        } catch (e) {
            alert("Rename failed: " + e.message);
        }
    });

    // Ctrl-D button (EOF, 0x04)
    document.getElementById("ctrl-d-btn").addEventListener("click", function () {
        sendBytes([0x04]);
        term.focus();
    });

    // Ctrl-C button (interrupt, 0x03)
    document.getElementById("ctrl-c-btn").addEventListener("click", function () {
        sendBytes([0x03]);
        term.focus();
    });

    // Ctrl-W button (delete word, 0x17)
    document.getElementById("ctrl-w-btn").addEventListener("click", function () {
        sendBytes([0x17]);
        term.focus();
    });

    // Paste button — read clipboard and send to terminal
    document.getElementById("paste-btn").addEventListener("click", async function () {
        try {
            const text = await clipPaste();
            if (text && activeWs && activeWs.readyState === WebSocket.OPEN) {
                activeWs.send(new TextEncoder().encode(text));
            }
            term.focus();
        } catch (e) {
            // Clipboard read requires HTTPS — prompt user to paste manually
            var input = prompt("Paste text here (clipboard read requires HTTPS):");
            if (input && activeWs && activeWs.readyState === WebSocket.OPEN) {
                activeWs.send(new TextEncoder().encode(input));
            }
            term.focus();
        }
    });

    // Zoom controls — the size row lives inside the font panel, which
    // swallows its own clicks (see the font picker below), so stepping the
    // size never closes the panel.
    function setFontSize(size) {
        size = Math.max(MIN_FONT_SIZE, Math.min(MAX_FONT_SIZE, size));
        term.options.fontSize = size;
        localStorage.setItem("term-font-size", size);
        var el1 = document.getElementById("zoom-level");
        var el2 = document.getElementById("zoom-level-m");
        if (el1) el1.textContent = size;
        if (el2) el2.textContent = size;
        refit();
    }
    document.getElementById("zoom-level").textContent = savedFontSize;
    document.getElementById("zoom-in-btn").addEventListener("click", function () {
        setFontSize(term.options.fontSize + 2);
    });
    document.getElementById("zoom-out-btn").addEventListener("click", function () {
        setFontSize(term.options.fontSize - 2);
    });
    document.getElementById("zoom-reset-btn").addEventListener("click", function () {
        setFontSize(DEFAULT_FONT_SIZE);
    });

    // ── Font family & weight ──
    // xterm measures the cell grid off the current face, so the woff2 must
    // be fetched BEFORE the options change — otherwise the canvas measures
    // the fallback font and every glyph lands in the wrong cell until the
    // next repaint. document.fonts.load() resolves once the face is usable;
    // weights the family doesn't ship nearest-match to the ones it does.
    function applyFont(fontId, weight) {
        var f = null;
        for (var i = 0; i < TERMINAL_FONTS.length; i++) {
            if (TERMINAL_FONTS[i].id === fontId) { f = TERMINAL_FONTS[i]; break; }
        }
        if (!f) f = TERMINAL_FONTS[0];
        weight = parseInt(weight, 10) || 400;
        currentFontId = f.id;
        currentFontWeight = weight;
        var boldWeight = Math.min(weight + 300, 800);
        function set() {
            term.options.fontFamily = f.family;
            term.options.fontWeight = weight;
            term.options.fontWeightBold = boldWeight;
            refit();
        }
        if (f.css && document.fonts && document.fonts.load) {
            // The 400 face must load too even when another weight is picked:
            // xterm's CharSizeService measure element only sets font-family
            // and font-size, so it measures at the inherited normal (400)
            // weight. Without the 400 face the grid is measured against the
            // fallback font and fit() packs in extra columns — the glyph rows
            // then overflow the window and typing drags the view sideways.
            Promise.all([
                document.fonts.load(weight + " 14px " + f.css),
                document.fonts.load(boldWeight + " 14px " + f.css),
                document.fonts.load("400 14px " + f.css),
            ]).then(set, set);
        } else {
            set();
        }
        localStorage.setItem("term-font-family", f.id);
        localStorage.setItem("term-font-weight", weight);
    }

    // Font picker dropdown
    (function () {
        var fontBtn = document.getElementById("font-btn");
        var fontDropdown = document.getElementById("font-dropdown");
        var fontList = document.getElementById("font-list");
        var weightRow = document.getElementById("font-weight-row");
        if (!fontBtn || !fontDropdown || !fontList) return;

        // Each option previews in its own face — the browser only fetches a
        // face when something renders in it, so the woff2s load lazily the
        // first time the panel opens.
        var html = "";
        TERMINAL_FONTS.forEach(function (f) {
            html += '<button class="font-option' + (f.id === currentFontId ? ' active' : '') + '"' +
                ' data-font-id="' + f.id + '" style="font-family:' +
                (f.family || "inherit").replace(/'/g, "&#39;") + '">' + f.name + '</button>';
        });
        fontList.innerHTML = html;

        function markActive() {
            fontList.querySelectorAll(".font-option").forEach(function (el) {
                el.classList.toggle("active", el.dataset.fontId === currentFontId);
            });
            if (weightRow) {
                weightRow.querySelectorAll("button").forEach(function (el) {
                    el.classList.toggle("active", parseInt(el.dataset.weight, 10) === currentFontWeight);
                });
            }
        }
        markActive();

        fontBtn.addEventListener("click", function (e) {
            e.stopPropagation();
            var isOpen = fontDropdown.style.display !== "none";
            // One toolbar panel at a time.
            var themeDd = document.getElementById("theme-dropdown");
            if (themeDd) themeDd.style.display = "none";
            var burger = document.getElementById("toolbar-menu");
            if (burger) burger.style.display = "none";
            fontDropdown.style.display = isOpen ? "none" : "block";
            fontBtn.setAttribute("aria-expanded", isOpen ? "false" : "true");
        });

        // The panel stays open across clicks — it's a settings surface you
        // tweak repeatedly (size, weight, compare two faces), unlike the
        // pick-one-and-done theme list. Click anywhere else to close it.
        fontDropdown.addEventListener("click", function (e) {
            e.stopPropagation();
            var opt = e.target.closest(".font-option");
            if (opt) {
                applyFont(opt.dataset.fontId, currentFontWeight);
                markActive();
                return;
            }
            var w = e.target.closest("[data-weight]");
            if (w) {
                applyFont(currentFontId, w.dataset.weight);
                markActive();
            }
        });

        document.addEventListener("click", function () {
            fontDropdown.style.display = "none";
            fontBtn.setAttribute("aria-expanded", "false");
        });
    })();

    // Apply the saved font once at startup (async: waits for the woff2).
    if (currentFontId !== "default" || currentFontWeight !== 400) {
        applyFont(currentFontId, currentFontWeight);
    }

    // ── Theme switching ──
    function applyTheme(themeId) {
        var entry = TERMINAL_THEMES[themeId] || TERMINAL_THEMES["default"];
        term.options.theme = entry.theme;
        currentThemeId = themeId;
        // Sync page background & toolbar with terminal background
        document.body.style.background = entry.theme.background;
        var toolbar = document.getElementById("toolbar");
        if (toolbar) {
            // Darken the background slightly for toolbar contrast
            toolbar.style.background = darkenColor(entry.theme.background, 0.15);
        }
        var mobileBar = document.getElementById("mobile-bar");
        if (mobileBar) {
            mobileBar.style.background = darkenColor(entry.theme.background, 0.15);
        }
        // The theme button is a 2x2 swatch grid that previews the active
        // theme. Colors are pulled toward the toolbar background so the icon
        // reads as UI chrome rather than a row of highlighter pens.
        var chrome = darkenColor(entry.theme.background, 0.15);
        var swatches = [entry.theme.blue, entry.theme.green, entry.theme.magenta || entry.theme.red, entry.theme.yellow];
        swatches.forEach(function (color, i) {
            var el = document.getElementById("sw-" + (i + 1));
            if (el) el.setAttribute("fill", mixColor(color || entry.theme.foreground, chrome, 0.35));
        });
        applyScrollbarTheme(entry.theme);
        // Persist locally for instant load next time
        localStorage.setItem("term-theme:" + host + ":" + session, themeId);
    }

    // Tint the scrollbars (xterm's scrollback viewport, the theme dropdown,
    // the mobile key bar) to the active theme. The thumb is the terminal
    // background blended toward the foreground, so it stays low-contrast on
    // dark and light themes alike instead of the bright browser default.
    function applyScrollbarTheme(theme) {
        var bg = theme.background || "#1a1a2e";
        var fg = theme.foreground || "#e0e0e8";
        var root = document.documentElement;
        root.style.setProperty("--sb-thumb", mixColor(bg, fg, 0.28));
        root.style.setProperty("--sb-thumb-hover", mixColor(bg, fg, 0.5));
        // Keeps any unstyled native UI (Firefox, form controls) on the same side.
        root.style.colorScheme = luminance(bg) < 0.5 ? "dark" : "light";
    }

    function parseHex(hex) {
        hex = String(hex).replace("#", "");
        if (hex.length === 3) hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2];
        return [
            parseInt(hex.substring(0, 2), 16) || 0,
            parseInt(hex.substring(2, 4), 16) || 0,
            parseInt(hex.substring(4, 6), 16) || 0,
        ];
    }

    // Linear blend of two hex colors; amount 0 = a, 1 = b.
    function mixColor(a, b, amount) {
        var ca = parseHex(a), cb = parseHex(b);
        var r = Math.round(ca[0] + (cb[0] - ca[0]) * amount);
        var g = Math.round(ca[1] + (cb[1] - ca[1]) * amount);
        var bl = Math.round(ca[2] + (cb[2] - ca[2]) * amount);
        return "#" + ((1 << 24) + (r << 16) + (g << 8) + bl).toString(16).slice(1);
    }

    // Rough perceptual brightness, 0..1.
    function luminance(hex) {
        var c = parseHex(hex);
        return (0.299 * c[0] + 0.587 * c[1] + 0.114 * c[2]) / 255;
    }

    function darkenColor(hex, amount) {
        hex = hex.replace("#", "");
        var r = parseInt(hex.substring(0, 2), 16);
        var g = parseInt(hex.substring(2, 4), 16);
        var b = parseInt(hex.substring(4, 6), 16);
        r = Math.max(0, Math.round(r * (1 - amount)));
        g = Math.max(0, Math.round(g * (1 - amount)));
        b = Math.max(0, Math.round(b * (1 - amount)));
        return "#" + ((1 << 24) + (r << 16) + (g << 8) + b).toString(16).slice(1);
    }

    function saveThemeToServer(themeId) {
        // Fetch existing session icon data first, then merge theme
        fetch("/api/session-icons").then(function (r) { return r.json(); }).then(function (all) {
            var key = host + ":" + session;
            var existing = all[key] || {};
            existing.theme = themeId;
            return fetch("/api/session-icons/" + encodeURIComponent(host) + "/" + encodeURIComponent(session), {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(existing),
            });
        }).catch(function () {});
    }

    // Apply initial theme (sets page background etc.)
    applyTheme(currentThemeId);

    // Theme picker dropdown
    (function () {
        var themeBtn = document.getElementById("theme-btn");
        var themeDropdown = document.getElementById("theme-dropdown");
        if (!themeBtn || !themeDropdown) return;

        // Populate theme options
        var html = "";
        Object.keys(TERMINAL_THEMES).forEach(function (id) {
            var t = TERMINAL_THEMES[id];
            var colors = [t.theme.background, t.theme.foreground, t.theme.red || "#f00",
                t.theme.green || "#0f0", t.theme.blue || "#00f", t.theme.yellow || "#ff0"];
            html += '<button class="theme-option' + (id === currentThemeId ? ' active' : '') + '" data-theme-id="' + id + '">';
            html += '<span class="theme-swatches">';
            colors.forEach(function (c) {
                html += '<span class="theme-swatch" style="background:' + c + '"></span>';
            });
            html += '</span>';
            html += '<span class="theme-name">' + t.name + '</span>';
            html += '</button>';
        });
        themeDropdown.innerHTML = html;

        // Keep the dropdown's tick in step with the active theme, however
        // the theme was changed (picked from the list, or cycled).
        function markActive(id) {
            themeDropdown.querySelectorAll(".theme-option").forEach(function (el) {
                el.classList.toggle("active", el.dataset.themeId === id);
            });
        }

        var THEME_IDS = Object.keys(TERMINAL_THEMES);
        var flashTimer = null;

        function flashThemeName(name) {
            var flash = document.getElementById("theme-flash");
            if (!flash) return;
            flash.textContent = name;
            flash.classList.add("show");
            clearTimeout(flashTimer);
            flashTimer = setTimeout(function () { flash.classList.remove("show"); }, 1100);
        }

        // step +1 = next theme, -1 = previous. Wraps both ways.
        function cycleTheme(step) {
            var i = THEME_IDS.indexOf(currentThemeId);
            if (i < 0) i = 0;
            var next = THEME_IDS[(i + step + THEME_IDS.length) % THEME_IDS.length];
            applyTheme(next);
            saveThemeToServer(next);
            markActive(next);
            flashThemeName(TERMINAL_THEMES[next].name);
        }

        // Middle-press opens the browser's autoscroll widget over the page
        // (that four-way pan cursor) unless the default is cancelled here.
        themeBtn.addEventListener("mousedown", function (e) {
            if (e.button === 1) e.preventDefault();
        });
        // Middle click cycles forward, shift+middle back. Desktop-only by
        // nature — the dropdown stays the discoverable path.
        themeBtn.addEventListener("auxclick", function (e) {
            if (e.button !== 1) return;
            e.preventDefault();
            e.stopPropagation();
            cycleTheme(e.shiftKey ? -1 : 1);
            term.focus();
        });

        themeBtn.addEventListener("click", function (e) {
            e.stopPropagation();
            var isOpen = themeDropdown.style.display !== "none";
            // Only one toolbar panel open at a time — the click that opens
            // this one is swallowed above, so the others won't self-close.
            var burger = document.getElementById("toolbar-menu");
            if (burger) burger.style.display = "none";
            var fontDd = document.getElementById("font-dropdown");
            if (fontDd) fontDd.style.display = "none";
            themeDropdown.style.display = isOpen ? "none" : "block";
            themeBtn.setAttribute("aria-expanded", isOpen ? "false" : "true");
        });

        themeDropdown.addEventListener("click", function (e) {
            var opt = e.target.closest(".theme-option");
            if (!opt) return;
            var id = opt.dataset.themeId;
            applyTheme(id);
            saveThemeToServer(id);
            markActive(id);
            themeDropdown.style.display = "none";
            themeBtn.setAttribute("aria-expanded", "false");
            term.focus();
        });

        document.addEventListener("click", function () {
            themeDropdown.style.display = "none";
            themeBtn.setAttribute("aria-expanded", "false");
        });
    })();

    // Load server-saved theme (overrides localStorage if different)
    (function () {
        fetch("/api/session-icons").then(function (r) { return r.json(); }).then(function (all) {
            var key = host + ":" + session;
            var data = all[key];
            if (data && data.theme && data.theme !== currentThemeId && TERMINAL_THEMES[data.theme]) {
                applyTheme(data.theme);
                // Update active state in dropdown
                var dd = document.getElementById("theme-dropdown");
                if (dd) {
                    dd.querySelectorAll(".theme-option").forEach(function (el) {
                        el.classList.toggle("active", el.dataset.themeId === data.theme);
                    });
                }
            }
        }).catch(function () {});
    })();

    // Burger menu toggle
    var burgerBtn = document.getElementById("toolbar-burger");
    var burgerMenu = document.getElementById("toolbar-menu");
    var detachBtn = document.getElementById("detach-btn");
    detachBtn.style.display = "none"; // hidden by default until we know there are other clients
    burgerBtn.addEventListener("click", async function (e) {
        e.stopPropagation();
        var opening = burgerMenu.style.display === "none";
        // See the theme button: the toolbar panels are mutually exclusive.
        var themeDd = document.getElementById("theme-dropdown");
        if (themeDd) themeDd.style.display = "none";
        var fontDd = document.getElementById("font-dropdown");
        if (fontDd) fontDd.style.display = "none";
        burgerMenu.style.display = opening ? "block" : "none";
        burgerBtn.setAttribute("aria-expanded", opening ? "true" : "false");
        if (opening) {
            // Check how many clients are attached to decide whether to show kick button
            try {
                const res = await fetch(`/api/hosts/${encodeURIComponent(host)}/sessions/${encodeURIComponent(session)}/clients`);
                if (res.ok) {
                    const clients = await res.json();
                    detachBtn.style.display = clients.length > 1 ? "" : "none";
                }
            } catch (_) {}
        }
    });
    document.addEventListener("click", function () {
        burgerMenu.style.display = "none";
        burgerBtn.setAttribute("aria-expanded", "false");
    });
    burgerMenu.addEventListener("click", function (e) {
        if (e.target.closest(".toolbar-menu-item")) {
            setTimeout(function () { burgerMenu.style.display = "none"; }, 50);
        }
    });

    // Duplicate button — a second session beside this one, in the directory
    // it is sitting in now and with the command it was launched with. The
    // server owns the naming (foo → foo-COPY → foo-COPY2) because picking
    // the next free number means seeing both live and offloaded sessions.
    document.getElementById("duplicate-btn").addEventListener("click", async function () {
        const btn = this;
        btn.disabled = true;
        btn.textContent = "Opening…";
        try {
            const res = await fetch(`/api/hosts/${encodeURIComponent(host)}/sessions/${encodeURIComponent(session)}/duplicate`, {
                method: "POST",
            });
            if (!res.ok) throw new Error(await res.text() || `Server returned ${res.status}`);
            const data = await res.json();
            window.open(`/terminal/${encodeURIComponent(host)}/${encodeURIComponent(data.name)}`, "_blank");
        } catch (e) {
            alert("Duplicate failed: " + e.message);
        } finally {
            btn.disabled = false;
            btn.textContent = "Duplicate";
        }
    });

    // Offload — stop tmux but keep the session resumable. Not confirmed:
    // it's reversible from the dashboard, working directory and launch
    // command included.
    document.getElementById("offload-btn").addEventListener("click", async function () {
        const btn = this;
        btn.disabled = true;
        btn.textContent = "Offloading…";
        // Set before the request: if the relay drops before the server's
        // "terminated" message arrives, this still blocks the reconnect.
        terminated = "offloaded";
        try {
            const res = await fetch(`/api/hosts/${encodeURIComponent(host)}/sessions/${encodeURIComponent(session)}/offload`, {
                method: "POST",
            });
            if (!res.ok) throw new Error(await res.text() || `Server returned ${res.status}`);
            drawEndState("offloaded");
            if (activeWs) activeWs.close();
        } catch (e) {
            terminated = "";
            alert("Offload failed: " + e.message);
            btn.disabled = false;
            btn.textContent = "Offload";
        }
    });

    // Kill — irreversible, and this menu is one click from Send Ctrl-C, so
    // it asks first.
    document.getElementById("kill-btn").addEventListener("click", async function () {
        const btn = this;
        if (!confirm(`Kill session "${session}"? The tmux session is destroyed and ssh-to-go forgets it — this can't be undone.`)) return;
        btn.disabled = true;
        btn.textContent = "Killing…";
        terminated = "killed";
        try {
            const res = await fetch(`/api/hosts/${encodeURIComponent(host)}/sessions/${encodeURIComponent(session)}`, {
                method: "DELETE",
            });
            if (!res.ok) throw new Error(await res.text() || `Server returned ${res.status}`);
            drawEndState("killed");
            if (activeWs) activeWs.close();
        } catch (e) {
            terminated = "";
            alert("Kill failed: " + e.message);
            btn.disabled = false;
            btn.textContent = "Kill Session";
        }
    });

    // Copy-on-select: automatically copy highlighted text to clipboard
    // Uses a brief delay so the selection is finalized before copying
    let copyTimer = null;
    term.onSelectionChange(function () {
        clearTimeout(copyTimer);
        copyTimer = setTimeout(function () {
            const sel = term.getSelection();
            if (sel) {
                clipCopy(sel);
            }
        }, 150);
    });

    // Defense-in-depth: server already strips mouse-reporting DECSET sequences
    // for mouse=off attaches, but if anything slips through we also drop them
    // here so xterm.js never enters mouse-tracking mode.
    const mouseSeqRegex = /\x1b\[\?(9|10(00|01|02|03|05|06|15|16))[hl]/g;

    // Terminal-report replies xterm.js generates automatically: primary and
    // secondary device attributes (CSI ? … c / CSI > … c), cursor position
    // reports (CSI row;col R — digits required, so F-key variants like ESC[R
    // never match), and DSR status (CSI 0n). See onData below (issue #79).
    const reportReplyRegex = /\x1b\[\?[0-9;]*c|\x1b\[>[0-9;]*c|\x1b\[[0-9]+;[0-9]+R|\x1b\[0n/g;

    // No manual wheel forwarding — with mouse=off the relay isn't expecting
    // mouse-button reports, and xterm.js's native viewport handles wheel and
    // touch scrolling against its local 5000-row scrollback (which the relay
    // prefilled via tmux capture-pane on attach).

    // Clipboard helper that works on HTTP (non-secure contexts)
    function clipCopy(text) {
        if (navigator.clipboard && window.isSecureContext) {
            return navigator.clipboard.writeText(text);
        }
        var ta = document.createElement("textarea");
        ta.value = text;
        ta.style.cssText = "position:fixed;left:-9999px;top:-9999px;opacity:0";
        document.body.appendChild(ta);
        ta.focus();
        ta.select();
        try { document.execCommand("copy"); } catch (_) {}
        document.body.removeChild(ta);
        return Promise.resolve();
    }

    function clipPaste() {
        if (navigator.clipboard && window.isSecureContext) {
            return navigator.clipboard.readText();
        }
        return Promise.reject(new Error("Clipboard read requires HTTPS"));
    }

    // Track the link under cursor for "Copy Link" in context menu
    let hoveredLink = null;
    container.addEventListener("mouseover", function (e) {
        const linkEl = e.target.closest("a");
        hoveredLink = linkEl ? linkEl.href : null;
    });
    container.addEventListener("mouseout", function (e) {
        if (e.target.closest("a")) hoveredLink = null;
    });

    // Right-click context menu: Copy / Copy Link / Paste
    const ctxMenu = document.createElement("div");
    ctxMenu.id = "ctx-menu";
    ctxMenu.style.cssText = "display:none;position:fixed;z-index:1000;background:#1e1e38;border:1px solid #3a3a5a;border-radius:6px;padding:4px 0;min-width:140px;font-family:sans-serif;font-size:13px;color:#e0e0e8;box-shadow:0 4px 16px rgba(0,0,0,0.4);";
    document.body.appendChild(ctxMenu);

    function hideCtxMenu() { ctxMenu.style.display = "none"; }
    document.addEventListener("click", hideCtxMenu);
    document.addEventListener("mousedown", function (e) {
        if (ctxMenu.style.display !== "none" && !ctxMenu.contains(e.target)) {
            hideCtxMenu();
        }
    });
    document.addEventListener("keydown", hideCtxMenu);

    container.addEventListener("contextmenu", function (e) {
        e.preventDefault();

        const sel = term.getSelection();
        let items = "";
        if (sel) {
            items += '<div class="ctx-item" data-action="copy">Copy</div>';
        }
        if (hoveredLink) {
            items += '<div class="ctx-item" data-action="copy-link">Copy Link</div>';
            items += '<div class="ctx-item" data-action="open-link">Open Link</div>';
        }
        items += '<div class="ctx-item" data-action="paste">Paste</div>';
        ctxMenu.innerHTML = items;
        ctxMenu.style.display = "block";
        ctxMenu.style.left = Math.min(e.clientX, window.innerWidth - 160) + "px";
        ctxMenu.style.top = Math.min(e.clientY, window.innerHeight - 120) + "px";

        // Stash the link for the click handler
        const linkForMenu = hoveredLink;

        ctxMenu.querySelectorAll(".ctx-item").forEach(function (el) {
            el.style.cssText = "padding:6px 16px;cursor:pointer;";
            el.addEventListener("mouseenter", function () { this.style.background = "#3a3a5a"; });
            el.addEventListener("mouseleave", function () { this.style.background = "none"; });
            el.addEventListener("click", async function () {
                hideCtxMenu();
                const action = this.dataset.action;
                if (action === "copy") {
                    clipCopy(term.getSelection());
                } else if (action === "copy-link") {
                    clipCopy(linkForMenu);
                } else if (action === "open-link") {
                    window.open(linkForMenu, "_blank");
                } else if (action === "paste") {
                    try {
                        const text = await clipPaste();
                        if (text && activeWs && activeWs.readyState === WebSocket.OPEN) {
                            activeWs.send(new TextEncoder().encode(text));
                        }
                    } catch (_) {
                        var input = prompt("Paste text here (clipboard read requires HTTPS):");
                        if (input && activeWs && activeWs.readyState === WebSocket.OPEN) {
                            activeWs.send(new TextEncoder().encode(input));
                        }
                    }
                }
                term.focus();
            });
        });
    });

    // Mobile keyboard toolbar
    let ctrlActive = false;
    const mobileBar = document.getElementById("mobile-bar");
    const mobileInput = document.getElementById("mobile-input");

    // Mobile text input: forwards characters straight to the WebSocket so
    // we don't depend on xterm's hidden helper textarea getting focus +
    // input events on Android (which is fragile across browsers + IMEs).
    //
    // We intercept beforeinput rather than input — on Android Gboard,
    // ordinary letters go through IME composition that only commits on
    // space/enter, so listening for input alone would miss every
    // intermediate keystroke. beforeinput fires for every keystroke
    // BEFORE the field's value changes, and gives us e.data + e.inputType.
    // We send the typed character immediately and preventDefault so the
    // field stays empty and the IME has nothing to compose against.
    if (mobileInput) {
        function send(data) {
            if (!data) return;
            if (ctrlActive && data.length === 1) {
                const code = data.toLowerCase().charCodeAt(0);
                if (code >= 97 && code <= 122) data = String.fromCharCode(code - 96);
                ctrlActive = false;
                const ctrlBtn = mobileBar && mobileBar.querySelector('[data-mod="ctrl"]');
                if (ctrlBtn) { ctrlBtn.style.background = ""; ctrlBtn.style.color = ""; }
            }
            if (activeWs && activeWs.readyState === WebSocket.OPEN) {
                activeWs.send(new TextEncoder().encode(data));
            }
        }

        mobileInput.addEventListener("beforeinput", function (e) {
            switch (e.inputType) {
                case "insertText":
                case "insertCompositionText":
                case "insertReplacementText":
                case "insertFromComposition":
                    if (e.data) send(e.data);
                    e.preventDefault();
                    break;
                case "insertLineBreak":
                case "insertParagraph":
                    send("\r");
                    e.preventDefault();
                    break;
                case "deleteContentBackward":
                case "deleteWordBackward":
                    send("\x7f");
                    e.preventDefault();
                    break;
                case "deleteContentForward":
                    send("\x1b[3~");
                    e.preventDefault();
                    break;
                default:
                    // Unknown inputType — let the default happen and rely on
                    // the input event below to catch it.
                    break;
            }
        });

        // Safety net: if the field ever ends up with text (shouldn't, given
        // preventDefault above) ship it and reset.
        mobileInput.addEventListener("input", function () {
            if (mobileInput.value) {
                send(mobileInput.value);
                mobileInput.value = "";
            }
        });

        // Some Android IMEs send Enter and Backspace as keydown rather than
        // through beforeinput. Handle them here as a fallback.
        mobileInput.addEventListener("keydown", function (e) {
            if (e.key === "Enter") {
                e.preventDefault();
                send("\r");
            } else if (e.key === "Backspace" && !mobileInput.value) {
                e.preventDefault();
                send("\x7f");
            }
        });
    }

    if (mobileBar) {
        const keyMap = {
            Tab: "\t",
            Escape: "\x1b",
            ArrowUp: "\x1b[A",
            ArrowDown: "\x1b[B",
            ArrowRight: "\x1b[C",
            ArrowLeft: "\x1b[D",
        };

        mobileBar.addEventListener("click", function (e) {
            const btn = e.target.closest("button");
            if (!btn) return;
            e.preventDefault();

            // Ctrl modifier toggle
            if (btn.dataset.mod === "ctrl") {
                ctrlActive = !ctrlActive;
                btn.style.background = ctrlActive ? "#7c83ff" : "";
                btn.style.color = ctrlActive ? "#fff" : "";
                return;
            }

            let data;
            if (btn.dataset.key && keyMap[btn.dataset.key]) {
                data = keyMap[btn.dataset.key];
            } else if (btn.dataset.key) {
                data = btn.dataset.key.trim();
            }

            if (data && ctrlActive && data.length === 1) {
                // Convert to ctrl character (a=1, b=2, ..., z=26)
                const code = data.toLowerCase().charCodeAt(0);
                if (code >= 97 && code <= 122) {
                    data = String.fromCharCode(code - 96);
                }
                ctrlActive = false;
                const ctrlBtn = mobileBar.querySelector('[data-mod="ctrl"]');
                if (ctrlBtn) { ctrlBtn.style.background = ""; ctrlBtn.style.color = ""; }
            }

            if (data && activeWs && activeWs.readyState === WebSocket.OPEN) {
                activeWs.send(new TextEncoder().encode(data));
            }
            term.focus();
        });

        // Use visualViewport to refit terminal when keyboard opens/closes
        if (window.visualViewport) {
            window.visualViewport.addEventListener("resize", function () {
                var atBottom = term.buffer.active.viewportY >= term.buffer.active.baseY;
                refit();
                if (atBottom) term.scrollToBottom();
            });
        }
    }

    // On any viewport resize (including mobile keyboard), refit and scroll to cursor
    window.addEventListener("resize", function () {
        var atBottom = term.buffer.active.viewportY >= term.buffer.active.baseY;
        refit();
        if (atBottom) term.scrollToBottom();
    });

    connect();
}
