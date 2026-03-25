(function () {
    "use strict";

    const hostList = document.getElementById("host-list");
    const modal = document.getElementById("modal-overlay");
    const modalTitle = document.getElementById("modal-title");
    const modalFields = document.getElementById("modal-fields");
    const modalForm = document.getElementById("modal-form");
    const modalSubmit = document.getElementById("modal-submit");

    let hosts = [];
    let sessions = [];
    let modalHandler = null;

    // ── Fetch ──

    async function fetchAll() {
        try {
            const [hRes, sRes] = await Promise.all([
                fetch("/api/hosts"),
                fetch("/api/sessions"),
            ]);
            hosts = await hRes.json();
            sessions = await sRes.json();
            render();
        } catch (e) {
            console.error("fetch:", e);
        }
    }

    // ── Render ──

    function render() {
        // Group sessions by host
        const byHost = {};
        for (const s of sessions) {
            (byHost[s.host_name] = byHost[s.host_name] || []).push(s);
        }

        hostList.innerHTML = hosts.map(h => {
            const name = h.config.name;
            const hostSessions = byHost[name] || [];

            let statusClass = "offline";
            let statusText = "offline";
            if (h.online && h.tmux_detected) {
                statusClass = "online";
                statusText = h.tmux_version;
            } else if (h.online && !h.tmux_detected) {
                statusClass = "no-tmux";
                statusText = "tmux not found";
            } else if (h.error) {
                statusText = h.error.length > 50 ? h.error.slice(0, 50) + "\u2026" : h.error;
            }

            const lastPoll = h.last_poll ? timeAgo(new Date(h.last_poll)) : "never";

            let sessionsHTML = "";
            if (hostSessions.length === 0) {
                sessionsHTML = `<div class="no-sessions">No active sessions</div>`;
            } else {
                sessionsHTML = `<table class="session-table">
                    <thead><tr>
                        <th>Session</th><th>Windows</th><th>Created</th><th>Status</th><th>Actions</th>
                    </tr></thead>
                    <tbody>${hostSessions.map(s => {
                        const created = new Date(s.session.created).toLocaleString();
                        const badge = s.session.attached
                            ? `<span class="badge badge-attached">attached</span>`
                            : `<span class="badge badge-detached">detached</span>`;
                        return `<tr>
                            <td><strong>${esc(s.session.name)}</strong></td>
                            <td>${s.session.windows}</td>
                            <td>${created}</td>
                            <td>${badge}</td>
                            <td><div class="action-group">
                                <a class="btn btn-sm btn-primary" href="/terminal/${eu(name)}/${eu(s.session.name)}" target="_blank">Attach</a>
                                <button class="btn btn-sm" onclick="handoff('${ea(name)}','${ea(s.session.name)}')">SSH</button>
                                <button class="btn btn-sm btn-danger" onclick="killSession('${ea(name)}','${ea(s.session.name)}')">Kill</button>
                            </div></td>
                        </tr>`;
                    }).join("")}</tbody>
                </table>`;
            }

            return `<div class="host-card">
                <div class="host-header">
                    <div class="host-title">
                        <span class="host-status ${statusClass}"></span>
                        <span class="host-name">${esc(name)}</span>
                        <span class="host-addr">${esc(h.config.user)}@${esc(h.config.address)}</span>
                    </div>
                    <div class="host-actions">
                        <span class="host-meta-text">${esc(statusText)} &middot; polled ${lastPoll}</span>
                        <button class="btn btn-sm" onclick="scanHost('${ea(name)}')">Scan</button>
                        <button class="btn btn-sm btn-primary" onclick="newSessionFor('${ea(name)}')">+ Session</button>
                    </div>
                </div>
                <div class="host-body">${sessionsHTML}</div>
            </div>`;
        }).join("");
    }

    // ── Actions ──

    window.handoff = async function (host, session) {
        try {
            const res = await fetch(`/api/hosts/${eu(host)}/sessions/${eu(session)}/handoff`);
            const data = await res.json();
            await navigator.clipboard.writeText(data.command);
            toast("Copied: " + data.command, "success");
        } catch (e) {
            toast("Failed to copy handoff command", "error");
        }
    };

    window.killSession = async function (host, session) {
        if (!confirm(`Kill session "${session}" on ${host}?`)) return;
        try {
            await fetch(`/api/hosts/${eu(host)}/sessions/${eu(session)}`, { method: "DELETE" });
            toast("Session killed", "success");
            fetchAll();
        } catch (e) {
            toast("Failed to kill session", "error");
        }
    };

    window.scanHost = async function (host) {
        toast("Scanning " + host + "...", "success");
        try {
            await fetch(`/api/hosts/${eu(host)}/scan`, { method: "POST" });
            await fetchAll();
            toast(host + " scanned", "success");
        } catch (e) {
            toast("Scan failed: " + e.message, "error");
        }
    };

    window.newSessionFor = function (host) {
        openNewSessionModal(host);
    };

    // ── Refresh All ──

    document.getElementById("refresh-all-btn").addEventListener("click", async () => {
        const btn = document.getElementById("refresh-all-btn");
        btn.disabled = true;
        btn.textContent = "Scanning...";
        try {
            await fetch("/api/scan", { method: "POST" });
            await fetchAll();
            toast("All hosts scanned", "success");
        } catch (e) {
            toast("Refresh failed", "error");
        } finally {
            btn.disabled = false;
            btn.textContent = "Refresh All";
        }
    });

    // ── New Session Modal ──

    function openNewSessionModal(preselectedHost) {
        modalTitle.textContent = "New Session";
        modalSubmit.textContent = "Create";

        const onlineHosts = hosts.filter(h => h.online && h.tmux_detected);
        const options = onlineHosts.map(h => {
            const sel = h.config.name === preselectedHost ? "selected" : "";
            return `<option value="${ea(h.config.name)}" ${sel}>${esc(h.config.name)}</option>`;
        }).join("");

        modalFields.innerHTML = `
            <label for="m-host">Host</label>
            <select id="m-host" required>${options}</select>
            <label for="m-name">Session Name</label>
            <input type="text" id="m-name" placeholder="my-session" required>
        `;

        modalHandler = async () => {
            const host = document.getElementById("m-host").value;
            const name = document.getElementById("m-name").value.trim();
            if (!host || !name) return;

            const res = await fetch(`/api/hosts/${eu(host)}/sessions`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name }),
            });
            if (!res.ok) throw new Error(await res.text());
            toast(`Session "${name}" created on ${host}`, "success");
            fetchAll();
        };

        modal.classList.remove("hidden");
        setTimeout(() => document.getElementById("m-name").focus(), 50);
    }

    document.getElementById("new-session-btn").addEventListener("click", () => {
        openNewSessionModal(null);
    });

    // ── Add Host Modal ──

    document.getElementById("add-host-btn").addEventListener("click", () => {
        modalTitle.textContent = "Add Host";
        modalSubmit.textContent = "Add";

        modalFields.innerHTML = `
            <label for="m-hname">Name</label>
            <input type="text" id="m-hname" placeholder="my-server" required>
            <label for="m-addr">Address</label>
            <input type="text" id="m-addr" placeholder="192.168.1.100:22" required>
            <label for="m-user">User</label>
            <input type="text" id="m-user" placeholder="deploy" required>
            <label for="m-key">SSH Key Path (on server)</label>
            <input type="text" id="m-key" placeholder="~/.ssh/id_ed25519">
        `;

        modalHandler = async () => {
            const name = document.getElementById("m-hname").value.trim();
            const address = document.getElementById("m-addr").value.trim();
            const user = document.getElementById("m-user").value.trim();
            const key_path = document.getElementById("m-key").value.trim();
            if (!name || !address || !user) return;

            const res = await fetch("/api/hosts", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name, address, user, key_path }),
            });
            if (!res.ok) throw new Error(await res.text());
            toast(`Host "${name}" added`, "success");
            fetchAll();
        };

        modal.classList.remove("hidden");
        setTimeout(() => document.getElementById("m-hname").focus(), 50);
    });

    // ── Modal plumbing ──

    document.getElementById("modal-cancel").addEventListener("click", () => {
        modal.classList.add("hidden");
    });

    modal.addEventListener("click", (e) => {
        if (e.target === modal) modal.classList.add("hidden");
    });

    modalForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        if (!modalHandler) return;
        try {
            await modalHandler();
            modal.classList.add("hidden");
        } catch (err) {
            toast("Failed: " + err.message, "error");
        }
    });

    // ── Helpers ──

    function esc(s) {
        const d = document.createElement("div");
        d.textContent = s || "";
        return d.innerHTML;
    }

    function ea(s) {
        return (s || "").replace(/'/g, "\\'").replace(/"/g, "&quot;");
    }

    function eu(s) {
        return encodeURIComponent(s);
    }

    function timeAgo(date) {
        const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
        if (seconds < 5) return "just now";
        if (seconds < 60) return seconds + "s ago";
        const minutes = Math.floor(seconds / 60);
        if (minutes < 60) return minutes + "m ago";
        const hours = Math.floor(minutes / 60);
        return hours + "h ago";
    }

    function toast(msg, type) {
        const el = document.createElement("div");
        el.className = `toast toast-${type}`;
        el.textContent = msg;
        document.body.appendChild(el);
        setTimeout(() => el.remove(), 3000);
    }

    // ── Init ──

    fetchAll();
    setInterval(fetchAll, 5000);
})();
