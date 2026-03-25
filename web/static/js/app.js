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
    let keypairs = [];
    let settings = {};
    let pubKey = "";
    let modalHandler = null;

    // ── Auth-aware fetch ──

    async function authFetch(url, opts) {
        const res = await fetch(url, opts);
        if (res.status === 401) {
            window.location.href = "/login";
            throw new Error("Session expired");
        }
        return res;
    }

    // ── Fetch ──

    async function fetchPubKey() {
        try {
            const res = await authFetch("/api/pubkey");
            const data = await res.json();
            pubKey = data.public_key || "";
            renderPubKey();
        } catch (e) {
            console.error("fetch pubkey:", e);
        }
    }

    function renderPubKey() {
        const el = document.getElementById("pubkey-display");
        const section = document.getElementById("pubkey-section");
        if (!el || !pubKey) return;
        el.textContent = pubKey;
        // Hidden by default; only show if explicitly enabled in settings
        if (settings.show_pub_key === true) {
            section.classList.remove("hidden");
        } else {
            section.classList.add("hidden");
        }
    }

    async function fetchAll() {
        try {
            const [hRes, sRes, kRes, stRes] = await Promise.all([
                authFetch("/api/hosts"),
                authFetch("/api/sessions"),
                authFetch("/api/keypairs"),
                authFetch("/api/settings"),
            ]);
            hosts = await hRes.json();
            sessions = await sRes.json();
            keypairs = await kRes.json();
            settings = await stRes.json();
            render();
            renderPubKey();
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
                        <th>Session</th><th class="hide-mobile">Clients</th><th class="hide-mobile">Age</th><th class="hide-mobile">Status</th><th>Actions</th>
                    </tr></thead>
                    <tbody>${hostSessions.map(s => {
                        const age = timeAgo(new Date(s.session.created));
                        const badge = s.session.attached
                            ? `<span class="badge badge-attached">attached</span>`
                            : `<span class="badge badge-detached">detached</span>`;
                        return `<tr>
                            <td><a class="session-link" href="/terminal/${eu(name)}/${eu(s.session.name)}" target="_blank"><strong>${esc(s.session.name)}</strong></a></td>
                            <td class="hide-mobile">${s.session.windows}</td>
                            <td class="hide-mobile">${age}</td>
                            <td class="hide-mobile">${badge}</td>
                            <td><div class="action-group">
                                <a class="btn btn-sm btn-primary" href="/terminal/${eu(name)}/${eu(s.session.name)}" target="_blank">Webview</a>
                                <button class="btn btn-sm hide-mobile" onclick="handoff('${ea(name)}','${ea(s.session.name)}')">Terminal</button>
                                <button class="btn btn-sm hide-mobile" onclick="renameSession('${ea(name)}','${ea(s.session.name)}')">Rename</button>
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
                        <span class="host-addr">${esc(h.config.user)}@${esc(h.config.address)}${h.config.port && h.config.port !== 22 ? ':' + h.config.port : ''}</span>
                    </div>
                    <div class="host-actions">
                        <button class="btn btn-sm" onclick="editHost('${ea(name)}')">Edit</button>
                        <button class="btn btn-sm btn-primary" onclick="newSessionFor('${ea(name)}')">+ New</button>
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

    window.renameSession = function (host, session) {
        const newName = prompt(`Rename session "${session}":`, session);
        if (!newName || newName === session) return;
        fetch(`/api/hosts/${eu(host)}/sessions/${eu(session)}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ new_name: newName }),
        }).then(res => {
            if (!res.ok) return res.text().then(t => { throw new Error(t); });
            toast(`Renamed to "${newName}"`, "success");
            fetchAll();
        }).catch(e => toast("Rename failed: " + e.message, "error"));
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


    window.editHost = function (hostName) {
        const h = hosts.find(x => x.config.name === hostName);
        if (!h) return;

        modalTitle.textContent = "Edit Host";
        modalSubmit.textContent = "Save";

        let keypairHTML = "";
        if (keypairs.length > 1) {
            const kpOptions = keypairs.map(kp =>
                `<option value="${ea(kp.name)}" ${kp.name === h.config.key_name ? "selected" : ""}>${esc(kp.name)}</option>`
            ).join("");
            keypairHTML = `
                <label for="m-keyname">Keypair</label>
                <select id="m-keyname">${kpOptions}</select>
            `;
        }

        modalFields.innerHTML = `
            <label for="m-addr">Host Address</label>
            <input type="text" id="m-addr" value="${ea(h.config.address)}" required>
            <label for="m-port">SSH Port</label>
            <input type="number" id="m-port" value="${h.config.port || 22}" min="1" max="65535">
            <label for="m-user">User</label>
            <input type="text" id="m-user" value="${ea(h.config.user)}">
            ${keypairHTML}
            <hr style="border:0;border-top:1px solid #333;margin:16px 0 8px">
            <button type="button" class="btn btn-danger" id="m-delete-host" style="width:100%">Delete Host</button>
        `;

        setTimeout(() => {
            document.getElementById("m-delete-host").addEventListener("click", async () => {
                if (!confirm('Delete host "' + hostName + '"? This cannot be undone.')) return;
                try {
                    const res = await fetch('/api/hosts/' + eu(hostName), { method: 'DELETE' });
                    if (!res.ok) throw new Error(await res.text());
                    toast('Host "' + hostName + '" deleted', 'success');
                    modal.classList.add('hidden');
                    fetchAll();
                } catch (e) {
                    toast('Delete failed: ' + e.message, 'error');
                }
            });
        }, 0);

        modalHandler = async () => {
            const body = {
                address: document.getElementById("m-addr").value.trim(),
                port: parseInt(document.getElementById("m-port").value, 10) || 22,
                user: document.getElementById("m-user").value.trim(),
            };
            const kpEl = document.getElementById("m-keyname");
            if (kpEl) body.key_name = kpEl.value;

            const res = await fetch(`/api/hosts/${eu(hostName)}`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(body),
            });
            if (!res.ok) throw new Error(await res.text());
            toast("Host updated", "success");
            fetchAll();
        };

        modal.classList.remove("hidden");
        setTimeout(() => document.getElementById("m-addr").focus(), 50);
    };

    window.newSessionFor = async function (host) {
        // One-click: auto-generate session name and create immediately
        const name = "s-" + Date.now().toString(36);
        try {
            const res = await fetch(`/api/hosts/${eu(host)}/sessions`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name }),
            });
            if (!res.ok) throw new Error(await res.text());
            // Open terminal immediately
            window.open(`/terminal/${eu(host)}/${eu(name)}`, "_blank");
            fetchAll();
        } catch (e) {
            toast("Failed: " + e.message, "error");
        }
    };

    // ── Add Host Modal ──

    document.getElementById("add-host-btn").addEventListener("click", () => {
        modalTitle.textContent = "Add Host";
        modalSubmit.textContent = "Add";

        // Only show keypair selector if there's more than one
        let keypairHTML = "";
        if (keypairs.length > 1) {
            const kpOptions = keypairs.map(kp =>
                `<option value="${ea(kp.name)}">${esc(kp.name)}</option>`
            ).join("");
            keypairHTML = `
                <label for="m-keyname">Keypair</label>
                <select id="m-keyname">${kpOptions}</select>
            `;
        }

        modalFields.innerHTML = `
            <label for="m-addr">Host Address</label>
            <input type="text" id="m-addr" placeholder="myserver.example.com" required>
            <label for="m-port">SSH Port</label>
            <input type="number" id="m-port" placeholder="22" min="1" max="65535">
            <label for="m-user">User (leave blank for default)</label>
            <input type="text" id="m-user" placeholder="">
            <label for="m-hname">Name (optional, defaults to hostname)</label>
            <input type="text" id="m-hname" placeholder="">
            ${keypairHTML}
        `;

        modalHandler = async () => {
            const address = document.getElementById("m-addr").value.trim();
            if (!address) return;

            const body = { address };
            const port = parseInt(document.getElementById("m-port").value, 10);
            if (port > 0) body.port = port;
            const name = document.getElementById("m-hname").value.trim();
            const user = document.getElementById("m-user").value.trim();
            if (name) body.name = name;
            if (user) body.user = user;
            const kpEl = document.getElementById("m-keyname");
            if (kpEl && kpEl.value) body.key_name = kpEl.value;

            const res = await fetch("/api/hosts", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(body),
            });
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();
            toast(`Host "${data.name}" added`, "success");
            fetchAll();
        };

        modal.classList.remove("hidden");
        setTimeout(() => document.getElementById("m-addr").focus(), 50);
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

    // ── Copy pubkey ──

    document.getElementById("copy-pubkey-btn").addEventListener("click", async () => {
        try {
            await navigator.clipboard.writeText(pubKey);
            toast("Public key copied", "success");
        } catch (e) {
            toast("Copy failed", "error");
        }
    });

    // ── Version ──

    async function fetchVersion() {
        try {
            const res = await authFetch("/api/version");
            const data = await res.json();
            const el = document.getElementById("version-footer");
            if (el && data.version) el.textContent = data.version;
        } catch (e) { /* ignore */ }
    }

    // ── Init ──

    fetchPubKey();
    fetchAll();
    fetchVersion();
    setInterval(fetchAll, 5000);
})();
