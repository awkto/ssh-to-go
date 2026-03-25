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
        if (!el || !pubKey) return;
        el.textContent = pubKey;
        document.getElementById("pubkey-section").classList.remove("hidden");
    }

    async function fetchAll() {
        try {
            const [hRes, sRes, kRes] = await Promise.all([
                authFetch("/api/hosts"),
                authFetch("/api/sessions"),
                authFetch("/api/keypairs"),
            ]);
            hosts = await hRes.json();
            sessions = await sRes.json();
            keypairs = await kRes.json();
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
                                <button class="btn btn-sm" onclick="renameSession('${ea(name)}','${ea(s.session.name)}')">Rename</button>
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

    // ── New Session ──

    document.getElementById("new-session-btn").addEventListener("click", async () => {
        if (hosts.length === 0) {
            toast("Add a host first", "error");
            return;
        }
        if (hosts.length === 1) {
            // One host — just create directly
            window.newSessionFor(hosts[0].config.name);
            return;
        }
        // Multiple hosts — show a quick picker
        modalTitle.textContent = "New Session";
        modalSubmit.textContent = "Create";
        const options = hosts.map(h =>
            `<option value="${ea(h.config.name)}">${esc(h.config.name)}</option>`
        ).join("");
        modalFields.innerHTML = `
            <label for="m-host">Host</label>
            <select id="m-host" required>${options}</select>
        `;
        modalHandler = async () => {
            const host = document.getElementById("m-host").value;
            if (!host) return;
            modal.classList.add("hidden");
            window.newSessionFor(host);
        };
        modal.classList.remove("hidden");
    });

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

    // ── Init ──

    fetchPubKey();
    fetchAll();
    setInterval(fetchAll, 5000);
})();
