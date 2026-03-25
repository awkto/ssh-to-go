function initTerminal(host, session) {
    const term = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: "'SF Mono', 'Fira Code', 'Cascadia Code', monospace",
        theme: {
            background: "#1a1a2e",
            foreground: "#e0e0e8",
            cursor: "#7c83ff",
            selectionBackground: "#3a3a5a",
        },
    });

    const fitAddon = new FitAddon.FitAddon();
    term.loadAddon(fitAddon);

    const container = document.getElementById("terminal");
    term.open(container);
    fitAddon.fit();

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

        // Window resize -> fit -> terminal resize event
        window.addEventListener("resize", function () {
            fitAddon.fit();
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
    term.onSelectionChange(function () {
        const sel = term.getSelection();
        if (sel) {
            navigator.clipboard.writeText(sel).catch(() => {});
        }
    });

    connect();
}
