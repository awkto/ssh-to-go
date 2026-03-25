function initTerminal(host, session) {
    const term = new Terminal({
        cursorBlink: true,
        fontSize: 14,
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
                term.write(new Uint8Array(e.data));
            } else {
                // Check for control messages (resize acks, etc)
                try {
                    const msg = JSON.parse(e.data);
                    if (msg.type === "resize") return;
                } catch (_) {}
                term.write(e.data);
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
            await navigator.clipboard.writeText(data.command);
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
            const text = await navigator.clipboard.readText();
            if (text && activeWs && activeWs.readyState === WebSocket.OPEN) {
                activeWs.send(new TextEncoder().encode(text));
            }
            term.focus();
        } catch (e) {
            alert("Paste failed: " + e.message);
        }
    });

    // Duplicate button — create a new session on the same host, open in new tab
    document.getElementById("duplicate-btn").addEventListener("click", async function () {
        const btn = this;
        btn.disabled = true;
        btn.textContent = "Opening…";
        try {
            const newName = `${session}-dup-${Date.now()}`;
            const res = await fetch(`/api/hosts/${encodeURIComponent(host)}/sessions`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name: newName }),
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
                navigator.clipboard.writeText(sel).catch(() => {});
            }
        }, 100);
    });

    // Right-click context menu: Copy (if selected) / Paste
    const ctxMenu = document.createElement("div");
    ctxMenu.id = "ctx-menu";
    ctxMenu.style.cssText = "display:none;position:fixed;z-index:1000;background:#1e1e38;border:1px solid #3a3a5a;border-radius:6px;padding:4px 0;min-width:120px;font-family:sans-serif;font-size:13px;color:#e0e0e8;box-shadow:0 4px 16px rgba(0,0,0,0.4);";
    document.body.appendChild(ctxMenu);

    function hideCtxMenu() { ctxMenu.style.display = "none"; }
    document.addEventListener("click", hideCtxMenu);
    document.addEventListener("keydown", hideCtxMenu);

    container.addEventListener("contextmenu", function (e) {
        e.preventDefault();
        const sel = term.getSelection();
        let items = "";
        if (sel) {
            items += '<div class="ctx-item" data-action="copy">Copy</div>';
        }
        items += '<div class="ctx-item" data-action="paste">Paste</div>';
        ctxMenu.innerHTML = items;
        ctxMenu.style.display = "block";
        ctxMenu.style.left = Math.min(e.clientX, window.innerWidth - 140) + "px";
        ctxMenu.style.top = Math.min(e.clientY, window.innerHeight - 80) + "px";

        ctxMenu.querySelectorAll(".ctx-item").forEach(function (el) {
            el.style.cssText = "padding:6px 16px;cursor:pointer;";
            el.addEventListener("mouseenter", function () { this.style.background = "#3a3a5a"; });
            el.addEventListener("mouseleave", function () { this.style.background = "none"; });
            el.addEventListener("click", async function () {
                hideCtxMenu();
                if (this.dataset.action === "copy") {
                    await navigator.clipboard.writeText(term.getSelection()).catch(() => {});
                } else if (this.dataset.action === "paste") {
                    try {
                        const text = await navigator.clipboard.readText();
                        if (text && activeWs && activeWs.readyState === WebSocket.OPEN) {
                            activeWs.send(new TextEncoder().encode(text));
                        }
                    } catch (_) {}
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
