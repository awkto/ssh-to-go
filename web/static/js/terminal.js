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

function initTerminal(host, session) {
    const DEFAULT_FONT_SIZE = 14;
    const MIN_FONT_SIZE = 8;
    const MAX_FONT_SIZE = 32;
    const savedFontSize = parseInt(localStorage.getItem("term-font-size"), 10) || DEFAULT_FONT_SIZE;

    // Determine initial theme — check localStorage for a session-specific override,
    // then fall back to "default". The server-saved theme is applied asynchronously.
    var currentThemeId = localStorage.getItem("term-theme:" + host + ":" + session) || "default";
    var initialTheme = (TERMINAL_THEMES[currentThemeId] || TERMINAL_THEMES["default"]).theme;

    const term = new Terminal({
        cursorBlink: true,
        fontSize: savedFontSize,
        fontFamily: "'SF Mono', 'Fira Code', 'Cascadia Code', monospace",
        rightClickSelectsWord: true,
        scrollback: 5000,
        theme: initialTheme,
    });

    const fitAddon = new FitAddon.FitAddon();
    term.loadAddon(fitAddon);

    const webLinksAddon = new WebLinksAddon.WebLinksAddon();
    term.loadAddon(webLinksAddon);

    const container = document.getElementById("terminal");
    term.open(container);
    fitAddon.fit();

    // Scroll overlay buttons (touch devices) — dispatch wheel events same as
    // desktop mouse wheel, which tmux handles natively for scrollback.
    (function () {
        const screen = container.querySelector(".xterm-screen");
        if (!screen) return;

        function sendWheel(direction, count) {
            for (var i = 0; i < count; i++) {
                screen.dispatchEvent(new WheelEvent("wheel", {
                    deltaY: direction,
                    deltaMode: 1,
                    bubbles: true,
                    cancelable: true,
                }));
            }
        }

        // Continuous scroll on hold
        var holdTimer = null;
        var holdInterval = null;

        function startHold(direction) {
            sendWheel(direction, 3);
            holdTimer = setTimeout(function () {
                holdInterval = setInterval(function () {
                    sendWheel(direction, 5);
                }, 50);
            }, 300);
        }

        function stopHold() {
            clearTimeout(holdTimer);
            clearInterval(holdInterval);
            holdTimer = null;
            holdInterval = null;
        }

        var upBtn = document.getElementById("scroll-up");
        var downBtn = document.getElementById("scroll-down");

        if (upBtn && downBtn) {
            upBtn.addEventListener("touchstart", function (e) { e.preventDefault(); startHold(-1); });
            upBtn.addEventListener("mousedown", function (e) { e.preventDefault(); startHold(-1); });
            upBtn.addEventListener("touchend", stopHold);
            upBtn.addEventListener("mouseup", stopHold);
            upBtn.addEventListener("touchcancel", stopHold);
            upBtn.addEventListener("mouseleave", stopHold);

            downBtn.addEventListener("touchstart", function (e) { e.preventDefault(); startHold(1); });
            downBtn.addEventListener("mousedown", function (e) { e.preventDefault(); startHold(1); });
            downBtn.addEventListener("touchend", stopHold);
            downBtn.addEventListener("mouseup", stopHold);
            downBtn.addEventListener("touchcancel", stopHold);
            downBtn.addEventListener("mouseleave", stopHold);
        }

        // Touch swipe scrolling — throttled via rAF for smooth delivery.
        var touchY = 0;
        var pendingLines = 0;
        var scrollRaf = null;
        var PX_PER_LINE = 10;

        function flushScroll() {
            scrollRaf = null;
            if (pendingLines !== 0) {
                sendWheel(pendingLines > 0 ? 1 : -1, Math.abs(pendingLines));
                pendingLines = 0;
            }
        }

        // Touch swipe scrolling disabled — use scroll overlay buttons instead
        /*
        container.addEventListener("touchstart", function (e) {
            if (e.touches.length === 1 && !e.target.closest("#scroll-overlay")) {
                touchY = e.touches[0].clientY;
                pendingLines = 0;
            }
        }, { passive: true });

        container.addEventListener("touchmove", function (e) {
            if (e.touches.length !== 1) return;
            if (e.target.closest("#scroll-overlay")) return;
            e.preventDefault();

            var curY = e.touches[0].clientY;
            var dy = touchY - curY;
            var lines = Math.trunc(dy / PX_PER_LINE);
            if (lines !== 0) {
                touchY += lines * PX_PER_LINE;
                pendingLines += lines;
                if (!scrollRaf) scrollRaf = requestAnimationFrame(flushScroll);
            }
        }, { passive: false });
        */

    })();

    const statusEl = document.getElementById("ws-status");

    // Shared reference to the active WebSocket so button handlers can send data.
    let activeWs = null;

    // Disposables for event listeners that must be cleaned up on reconnect
    let onDataDisposable = null;
    let onResizeDisposable = null;

    // TTY path of our relay's tmux client, used to exclude ourselves when kicking others
    let myTTY = "";

    function sendBytes(bytes) {
        if (activeWs && activeWs.readyState === WebSocket.OPEN) {
            activeWs.send(new Uint8Array(bytes));
        }
    }

    function connect() {
        const proto = location.protocol === "https:" ? "wss:" : "ws:";
        const url = `${proto}//${location.host}/ws/${encodeURIComponent(host)}/${encodeURIComponent(session)}`;
        const ws = new WebSocket(url);
        activeWs = ws;
        ws.binaryType = "arraybuffer";

        ws.onopen = function () {
            statusEl.className = "status connected";
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
                var str = new TextDecoder().decode(bytes);
                var filtered = str.replace(mouseSeqRegex, "");
                if (filtered.length > 0) {
                    term.write(filtered);
                }
            } else {
                // Check for control messages (resize acks, tty, etc)
                try {
                    const msg = JSON.parse(e.data);
                    if (msg.type === "resize") return;
                    if (msg.type === "tty") { myTTY = msg.tty; return; }
                } catch (_) {}
                term.write(e.data.replace(mouseSeqRegex, ""));
            }
        };

        ws.onclose = function (e) {
            statusEl.className = "status disconnected";
            // Code 4000 = session ended normally (killed/destroyed)
            if (e.code === 4000) {
                term.write("\r\n\x1b[93m--- session ended ---\x1b[0m\r\n");
                return;
            }
            // Code 4001 = kicked/detached by another client
            if (e.code === 4001) {
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

        // Terminal input -> WebSocket
        onDataDisposable = term.onData(function (data) {
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(new TextEncoder().encode(data));
            }
        });

        // Handle resize
        onResizeDisposable = term.onResize(function (size) {
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: "resize",
                    cols: size.cols,
                    rows: size.rows,
                }));
            }
        });

    }

    // Handoff button
    document.getElementById("handoff-btn").addEventListener("click", async function () {
        try {
            const res = await fetch(`/api/hosts/${encodeURIComponent(host)}/sessions/${encodeURIComponent(session)}/handoff`);
            const data = await res.json();
            await clipCopy(data.command);
            this.textContent = "Copied!";
            setTimeout(() => { this.textContent = "Handoff"; }, 2000);
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

    // Zoom controls
    function setFontSize(size) {
        size = Math.max(MIN_FONT_SIZE, Math.min(MAX_FONT_SIZE, size));
        term.options.fontSize = size;
        localStorage.setItem("term-font-size", size);
        var el1 = document.getElementById("zoom-level");
        var el2 = document.getElementById("zoom-level-m");
        if (el1) el1.textContent = size;
        if (el2) el2.textContent = size;
        fitAddon.fit();
    }
    document.getElementById("zoom-level").textContent = savedFontSize;
    document.getElementById("zoom-in-btn").addEventListener("click", function () {
        setFontSize(term.options.fontSize + 2); term.focus();
    });
    document.getElementById("zoom-out-btn").addEventListener("click", function () {
        setFontSize(term.options.fontSize - 2); term.focus();
    });
    document.getElementById("zoom-reset-btn").addEventListener("click", function () {
        setFontSize(DEFAULT_FONT_SIZE); term.focus();
    });

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
        // Update the theme label in toolbar
        var themeLabel = document.getElementById("theme-label");
        if (themeLabel) themeLabel.textContent = entry.name;
        // Persist locally for instant load next time
        localStorage.setItem("term-theme:" + host + ":" + session, themeId);
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

        themeBtn.addEventListener("click", function (e) {
            e.stopPropagation();
            var isOpen = themeDropdown.style.display !== "none";
            themeDropdown.style.display = isOpen ? "none" : "block";
        });

        themeDropdown.addEventListener("click", function (e) {
            var opt = e.target.closest(".theme-option");
            if (!opt) return;
            var id = opt.dataset.themeId;
            applyTheme(id);
            saveThemeToServer(id);
            // Update active state
            themeDropdown.querySelectorAll(".theme-option").forEach(function (el) {
                el.classList.toggle("active", el.dataset.themeId === id);
            });
            themeDropdown.style.display = "none";
            term.focus();
        });

        document.addEventListener("click", function () { themeDropdown.style.display = "none"; });
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
    burgerBtn.addEventListener("click", function (e) {
        e.stopPropagation();
        burgerMenu.style.display = burgerMenu.style.display === "none" ? "block" : "none";
    });
    document.addEventListener("click", function () { burgerMenu.style.display = "none"; });
    burgerMenu.addEventListener("click", function (e) {
        if (e.target.closest(".toolbar-menu-item")) {
            setTimeout(function () { burgerMenu.style.display = "none"; }, 50);
        }
    });

    // Duplicate button — create a new session on the same host in the same working dir
    document.getElementById("duplicate-btn").addEventListener("click", async function () {
        const btn = this;
        btn.disabled = true;
        btn.textContent = "Opening…";
        try {
            // Get the current session's working directory
            let cwd = "";
            try {
                const cwdRes = await fetch(`/api/hosts/${encodeURIComponent(host)}/sessions/${encodeURIComponent(session)}/cwd`);
                if (cwdRes.ok) {
                    const cwdData = await cwdRes.json();
                    cwd = cwdData.cwd || "";
                }
            } catch (_) {}

            const newName = `${session}-dup-${Date.now()}`;
            const body = { name: newName };
            if (cwd) body.cwd = cwd;
            const res = await fetch(`/api/hosts/${encodeURIComponent(host)}/sessions`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(body),
            });
            if (!res.ok) {
                throw new Error(`Server returned ${res.status}`);
            }
            const data = await res.json();
            const sessName = data.name || newName;
            window.open(`/terminal/${encodeURIComponent(host)}/${encodeURIComponent(sessName)}`, "_blank");
        } catch (e) {
            alert("Duplicate failed: " + e.message);
        } finally {
            btn.disabled = false;
            btn.textContent = "Duplicate";
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

    // Strip mouse mode escape sequences from incoming data so xterm.js never
    // enters mouse reporting mode. This ensures local text selection works
    // natively without fighting tmux's mouse handling.
    // Matches: \x1b[?1000h, \x1b[?1002h, \x1b[?1003h, \x1b[?1006h and their
    // disable variants (l), plus \x1b[?1005h/l
    const mouseSeqRegex = /\x1b\[\?10(00|02|03|05|06)[hl]/g;

    // Since we strip mouse mode, scroll wheel won't be reported to tmux.
    // Manually send SGR mouse scroll sequences so tmux scrollback works.
    // tmux still expects mouse events (it set mouse mode, we just hid it from xterm).
    // SGR encoding: \x1b[<64;col;rowM = scroll up, \x1b[<65;col;rowM = scroll down
    container.addEventListener("wheel", function (e) {
        if (!activeWs || activeWs.readyState !== WebSocket.OPEN) return;
        e.preventDefault();
        e.stopPropagation();
        e.stopImmediatePropagation();

        // Clear any text selection so it doesn't shift/drift as the terminal scrolls
        term.clearSelection();

        // Use deltaMode to normalize: LINE mode sends 1 line per notch, PIXEL mode
        // divides by a larger value (120px ≈ one scroll notch on most mice).
        var lines;
        if (e.deltaMode === WheelEvent.DOM_DELTA_LINE) {
            lines = Math.max(1, Math.round(Math.abs(e.deltaY)));
        } else {
            lines = Math.max(1, Math.round(Math.abs(e.deltaY) / 120));
        }
        var btn = e.deltaY < 0 ? 64 : 65;
        var col = Math.floor(term.cols / 2);
        var row = Math.floor(term.rows / 2);
        var seq = "\x1b[<" + btn + ";" + col + ";" + row + "M";
        for (var i = 0; i < lines; i++) {
            activeWs.send(new TextEncoder().encode(seq));
        }
    }, { capture: true, passive: false });

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
                fitAddon.fit();
                if (atBottom) term.scrollToBottom();
            });
        }
    }

    // On any viewport resize (including mobile keyboard), refit and scroll to cursor
    window.addEventListener("resize", function () {
        var atBottom = term.buffer.active.viewportY >= term.buffer.active.baseY;
        fitAddon.fit();
        if (atBottom) term.scrollToBottom();
    });

    connect();
}
