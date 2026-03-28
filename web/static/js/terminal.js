function initTerminal(host, session) {
    const DEFAULT_FONT_SIZE = 14;
    const MIN_FONT_SIZE = 8;
    const MAX_FONT_SIZE = 32;
    const savedFontSize = parseInt(localStorage.getItem("term-font-size"), 10) || DEFAULT_FONT_SIZE;

    const term = new Terminal({
        cursorBlink: true,
        fontSize: savedFontSize,
        fontFamily: "'SF Mono', 'Fira Code', 'Cascadia Code', monospace",
        rightClickSelectsWord: true,
        scrollback: 5000,
        theme: {
            background: "#1a1a2e",
            foreground: "#e0e0e8",
            cursor: "#7c83ff",
            selectionBackground: "#3a3a5a",
        },
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
                // Check for control messages (resize acks, etc)
                try {
                    const msg = JSON.parse(e.data);
                    if (msg.type === "resize") return;
                } catch (_) {}
                term.write(e.data.replace(mouseSeqRegex, ""));
            }
        };

        ws.onclose = function (e) {
            statusEl.className = "status disconnected";
            // Code 4000 = session ended normally (server-side signal)
            if (e.code === 4000) {
                term.write("\r\n\x1b[93m--- session ended ---\x1b[0m\r\n");
                return;
            }
            term.write("\r\n\x1b[90m--- disconnected, reconnecting in 3s ---\x1b[0m\r\n");
            setTimeout(connect, 3000);
        };

        ws.onerror = function () {
            ws.close();
        };

        // Terminal input -> WebSocket
        term.onData(function (data) {
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(new TextEncoder().encode(data));
            }
        });

        // Handle resize
        term.onResize(function (size) {
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
            // Update the page title and label
            session = newName;
            document.getElementById("session-label").textContent = host + " / " + newName;
            document.title = host + " / " + newName + " — ssh-to-go";
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

        var lines = Math.max(1, Math.round(Math.abs(e.deltaY) / 40));
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
                fitAddon.fit();
                term.scrollToBottom();
            });
        }
    }

    // On any viewport resize (including mobile keyboard), refit and scroll to cursor
    window.addEventListener("resize", function () {
        fitAddon.fit();
        term.scrollToBottom();
    });

    connect();
}
