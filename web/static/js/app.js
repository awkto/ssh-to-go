(function () {
    "use strict";

    // ── State ──
    let hosts = [];
    let sessions = [];
    let keypairs = [];
    let settings = {};
    let activityLog = [];
    let currentView = "dashboard";
    let currentFilter = "all";
    let sessionSort = { key: "host", dir: 1 };
    let hostSort = { key: "name", dir: 1 };
    let modalHandler = null;

    // ── DOM refs ──
    const modal = document.getElementById("modal-overlay");
    const modalTitle = document.getElementById("modal-title");
    const modalFields = document.getElementById("modal-fields");
    const modalForm = document.getElementById("modal-form");
    const modalSubmit = document.getElementById("modal-submit");

    // ── Auth-aware fetch ──
    async function authFetch(url, opts) {
        const res = await fetch(url, opts);
        if (res.status === 401) {
            window.location.href = "/login";
            throw new Error("Session expired");
        }
        return res;
    }

    // ── Data fetching ──
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
            renderCurrentView();
        } catch (e) {
            console.error("fetch:", e);
        }
    }

    // ── View switching ──
    window.switchView = function (view) {
        currentView = view;
        document.querySelectorAll(".view-panel").forEach(el => el.classList.remove("active"));
        const panel = document.getElementById("view-" + view);
        if (panel) panel.classList.add("active");

        document.querySelectorAll(".nav-item[data-view]").forEach(el => {
            el.classList.toggle("active", el.dataset.view === view);
        });

        renderCurrentView();
        closeSidebar();
    };

    // Navigate to sessions view filtered by a specific host
    window.showSessionsForHost = function (hostName) {
        currentView = "sessions";
        document.querySelectorAll(".view-panel").forEach(el => el.classList.remove("active"));
        document.getElementById("view-sessions").classList.add("active");
        document.querySelectorAll(".nav-item[data-view]").forEach(el => {
            el.classList.toggle("active", el.dataset.view === "sessions");
        });
        const filterInput = document.getElementById("session-filter");
        if (filterInput) filterInput.value = hostName;
        renderSessions();
        closeSidebar();
    };

    function renderCurrentView() {
        switch (currentView) {
            case "dashboard": renderDashboard(); break;
            case "sessions": renderSessions(); break;
            case "hosts": renderHosts(); break;
            case "logs": renderLogs(); break;
        }
    }

    // ── Filters ──
    window.setFilter = function (filter) {
        currentFilter = filter;
        document.querySelectorAll(".filter-item").forEach(el => {
            el.classList.toggle("active", el.dataset.filter === filter);
        });
        renderCurrentView();
    };

    function getFilteredSessions() {
        let filtered = sessions.slice();
        const search = (document.getElementById("global-search").value || "").toLowerCase();
        const viewFilter = (document.getElementById("session-filter")?.value || "").toLowerCase();

        if (currentFilter === "active") {
            filtered = filtered.filter(s => s.session.attached);
        }

        if (search) {
            filtered = filtered.filter(s =>
                s.session.name.toLowerCase().includes(search) ||
                s.host_name.toLowerCase().includes(search)
            );
        }

        if (viewFilter) {
            filtered = filtered.filter(s =>
                s.session.name.toLowerCase().includes(viewFilter) ||
                s.host_name.toLowerCase().includes(viewFilter)
            );
        }

        return filtered;
    }

    function getFilteredHosts() {
        let filtered = hosts.slice();
        const search = (document.getElementById("global-search").value || "").toLowerCase();
        const viewFilter = (document.getElementById("host-filter")?.value || "").toLowerCase();

        if (currentFilter === "active") {
            filtered = filtered.filter(h => {
                const hostSessions = sessions.filter(s => s.host_name === h.config.name);
                return hostSessions.length > 0;
            });
        }

        if (search) {
            filtered = filtered.filter(h =>
                h.config.name.toLowerCase().includes(search) ||
                h.config.address.toLowerCase().includes(search)
            );
        }

        if (viewFilter) {
            filtered = filtered.filter(h =>
                h.config.name.toLowerCase().includes(viewFilter) ||
                h.config.address.toLowerCase().includes(viewFilter)
            );
        }

        return filtered;
    }

    // ── Dashboard rendering ──
    function renderDashboard() {
        const totalHosts = hosts.length;
        const onlineHosts = hosts.filter(h => h.online).length;
        const totalSessions = sessions.length;
        const attachedSessions = sessions.filter(s => s.session.attached).length;

        document.getElementById("dashboard-cards").innerHTML = `
            <div class="dash-card">
                <div class="dash-card-label">Total Hosts</div>
                <div class="dash-card-value">${totalHosts}</div>
                <div class="dash-card-sub">${onlineHosts} online</div>
            </div>
            <div class="dash-card">
                <div class="dash-card-label">Active Sessions</div>
                <div class="dash-card-value">${totalSessions}</div>
                <div class="dash-card-sub">${attachedSessions} attached</div>
            </div>
            <div class="dash-card">
                <div class="dash-card-label">Keypairs</div>
                <div class="dash-card-value">${keypairs.length}</div>
                <div class="dash-card-sub">default: ${esc(settings.default_keypair || "none")}</div>
            </div>
        `;

        // Recent sessions (last 10)
        const recent = sessions.slice().sort((a, b) =>
            new Date(b.session.created) - new Date(a.session.created)
        ).slice(0, 10);

        const body = document.getElementById("dashboard-recent-body");
        if (recent.length === 0) {
            body.innerHTML = `<tr><td colspan="5" class="empty-state">No active sessions</td></tr>`;
            return;
        }

        body.innerHTML = recent.map(s => {
            const host = hosts.find(h => h.config.name === s.host_name);
            const statusClass = s.session.attached ? "attached" : "detached";
            const age = timeAgo(new Date(s.session.created));
            return `<tr>
                <td><span class="status-dot ${statusClass}"></span></td>
                <td><a class="session-link" href="/terminal/${eu(s.host_name)}/${eu(s.session.name)}" target="_blank">${esc(s.session.name)}</a></td>
                <td>${esc(s.host_name)}</td>
                <td class="hide-mobile">${age}</td>
                <td>
                    <div class="action-group">
                        <a class="btn btn-sm btn-primary" href="/terminal/${eu(s.host_name)}/${eu(s.session.name)}" target="_blank">Web</a>
                        <button class="btn btn-sm btn-terminal hide-mobile" onclick="handoff('${ea(s.host_name)}','${ea(s.session.name)}')">Terminal</button>
                        <button class="btn btn-sm btn-danger" onclick="killSession('${ea(s.host_name)}','${ea(s.session.name)}')">End</button>
                    </div>
                </td>
            </tr>`;
        }).join("");
    }

    // ── Sessions rendering ──
    window.sortSessions = function (key) {
        if (sessionSort.key === key) {
            sessionSort.dir *= -1;
        } else {
            sessionSort.key = key;
            sessionSort.dir = 1;
        }
        renderSessions();
    };

    window.renderSessions = function () {
        let filtered = getFilteredSessions();

        // Sort
        filtered.sort((a, b) => {
            let va, vb;
            switch (sessionSort.key) {
                case "name": va = a.session.name; vb = b.session.name; break;
                case "host": va = a.host_name; vb = b.host_name; break;
                case "uptime": va = new Date(a.session.created); vb = new Date(b.session.created); return (va - vb) * sessionSort.dir;
                case "status": va = a.session.attached ? 1 : 0; vb = b.session.attached ? 1 : 0; return (vb - va) * sessionSort.dir;
                default: va = a.host_name; vb = b.host_name;
            }
            return String(va).localeCompare(String(vb)) * sessionSort.dir;
        });

        const body = document.getElementById("sessions-body");
        if (filtered.length === 0) {
            body.innerHTML = `<tr><td colspan="5" class="empty-state">No sessions found</td></tr>`;
            return;
        }

        body.innerHTML = filtered.map(s => {
            const statusClass = s.session.attached ? "attached" : "detached";
            const age = timeAgo(new Date(s.session.created));
            const host = hosts.find(h => h.config.name === s.host_name);
            const hostAddr = host ? host.config.address : "";

            return `<tr>
                <td><span class="status-dot ${statusClass}"></span></td>
                <td><a class="session-link" href="/terminal/${eu(s.host_name)}/${eu(s.session.name)}" target="_blank">${esc(s.session.name)}</a></td>
                <td>
                    <div class="host-cell">
                        <span class="host-fqdn">${esc(s.host_name)}</span>
                        <span class="host-ip">${esc(hostAddr)}</span>
                    </div>
                </td>
                <td class="hide-mobile">${age}</td>
                <td>
                    <div class="action-group">
                        <a class="btn btn-sm btn-primary" href="/terminal/${eu(s.host_name)}/${eu(s.session.name)}" target="_blank">Web</a>
                        <button class="btn btn-sm btn-terminal hide-mobile" onclick="handoff('${ea(s.host_name)}','${ea(s.session.name)}')">Terminal</button>
                        <button class="btn btn-sm btn-danger" onclick="killSession('${ea(s.host_name)}','${ea(s.session.name)}')">End</button>
                    </div>
                </td>
            </tr>`;
        }).join("");
    };

    // ── Hosts rendering ──
    window.sortHosts = function (key) {
        if (hostSort.key === key) {
            hostSort.dir *= -1;
        } else {
            hostSort.key = key;
            hostSort.dir = 1;
        }
        renderHosts();
    };

    window.renderHosts = function () {
        let filtered = getFilteredHosts();

        // Sort
        filtered.sort((a, b) => {
            let va, vb;
            switch (hostSort.key) {
                case "name": va = a.config.name; vb = b.config.name; break;
                case "sessions":
                    va = sessions.filter(s => s.host_name === a.config.name).length;
                    vb = sessions.filter(s => s.host_name === b.config.name).length;
                    return (va - vb) * hostSort.dir;
                case "sync":
                    va = a.last_poll ? new Date(a.last_poll) : new Date(0);
                    vb = b.last_poll ? new Date(b.last_poll) : new Date(0);
                    return (va - vb) * hostSort.dir;
                default: va = a.config.name; vb = b.config.name;
            }
            return String(va).localeCompare(String(vb)) * hostSort.dir;
        });

        const body = document.getElementById("hosts-body");
        if (filtered.length === 0) {
            body.innerHTML = `<tr><td colspan="5" class="empty-state">No hosts found. Add a host to get started.</td></tr>`;
            return;
        }

        body.innerHTML = filtered.map(h => {
            const name = h.config.name;
            const addr = h.config.address;
            const port = h.config.port && h.config.port !== 22 ? ":" + h.config.port : "";
            const user = h.config.user || settings.default_username || "";
            const hostSessions = sessions.filter(s => s.host_name === name);
            const sessionCount = hostSessions.length;
            const lastPoll = h.last_poll ? timeAgo(new Date(h.last_poll)) : "never";

            let statusClass = "offline";
            if (h.online && h.tmux_detected) statusClass = "online";
            else if (h.online && !h.tmux_detected) statusClass = "no-tmux";

            // OS: prefer manual override, then detected, then guess from name
            const osName = h.config.os || h.detected_os || guessOS(name, addr);

            let sessionsBadge;
            if (sessionCount > 0) {
                const label = sessionCount === 1 ? "1 Session" : sessionCount + " Sessions";
                sessionsBadge = `<span class="badge-sessions badge-clickable" onclick="showSessionsForHost('${ea(name)}')">${label}</span>`;
            } else {
                sessionsBadge = `<span class="badge-no-sessions">No Sessions</span>`;
            }

            return `<tr>
                <td>
                    <div class="host-cell">
                        <span class="host-fqdn">
                            <span class="status-dot ${statusClass}" style="margin-right:8px"></span>
                            ${esc(name)}
                        </span>
                        <span class="host-ip">${esc(user ? user + "@" : "")}${esc(addr)}${esc(port)}</span>
                    </div>
                </td>
                <td>${sessionsBadge}</td>
                <td class="hide-mobile">
                    <div class="os-badge">${osIcon(osName)} ${esc(osName)}</div>
                </td>
                <td class="hide-mobile">${lastPoll}</td>
                <td>
                    <div class="action-group">
                        <button class="btn btn-sm btn-primary" onclick="newSessionFor('${ea(name)}')">New Session</button>
                        <button class="btn btn-sm hide-mobile" onclick="editHost('${ea(name)}')">Edit</button>
                    </div>
                </td>
            </tr>`;
        }).join("");
    };

    // ── Logs rendering ──
    function renderLogs() {
        const el = document.getElementById("log-entries");
        if (activityLog.length === 0) {
            el.innerHTML = `<div class="empty-state">Activity logging will appear here as you interact with sessions and hosts.</div>`;
            return;
        }
        el.innerHTML = activityLog.map(log => `
            <div class="log-entry">
                <span class="log-time">${log.time}</span>
                <span class="log-level ${log.level}">${log.level}</span>
                <span class="log-msg">${esc(log.msg)}</span>
            </div>
        `).join("");
    }

    function addLog(level, msg) {
        const now = new Date();
        const time = now.toLocaleTimeString();
        activityLog.unshift({ time, level, msg });
        if (activityLog.length > 100) activityLog.pop();
        if (currentView === "logs") renderLogs();
    }

    // ── OS guessing (heuristic from hostname) ──
    function guessOS(name, addr) {
        const lower = (name + " " + addr).toLowerCase();
        if (lower.includes("ubuntu")) return "Ubuntu";
        if (lower.includes("debian")) return "Debian";
        if (lower.includes("centos")) return "CentOS";
        if (lower.includes("rhel") || lower.includes("redhat")) return "RHEL";
        if (lower.includes("oracle")) return "Oracle Linux";
        if (lower.includes("alpine")) return "Alpine";
        if (lower.includes("arch")) return "Arch Linux";
        if (lower.includes("fedora")) return "Fedora";
        if (lower.includes("suse") || lower.includes("sles")) return "SUSE";
        if (lower.includes("freebsd")) return "FreeBSD";
        if (lower.includes("mac") || lower.includes("darwin")) return "macOS";
        if (lower.includes("win")) return "Windows";
        return "Linux";
    }

    function osIcon(osName) {
        // Simple colored circle based on OS
        const colors = {
            "Ubuntu": "#E95420",
            "Debian": "#A80030",
            "CentOS": "#932178",
            "RHEL": "#EE0000",
            "Oracle Linux": "#F80000",
            "Alpine": "#0D597F",
            "Arch Linux": "#1793D1",
            "Fedora": "#294172",
            "SUSE": "#73BA25",
            "FreeBSD": "#AB2B28",
            "macOS": "#999",
            "Windows": "#0078D6",
            "Linux": "#FCC624",
        };
        const color = colors[osName] || "#888";
        return `<span style="display:inline-block;width:14px;height:14px;border-radius:50%;background:${color};flex-shrink:0"></span>`;
    }

    // ── Actions ──
    function copyToClipboard(text) {
        if (navigator.clipboard && window.isSecureContext) {
            return navigator.clipboard.writeText(text);
        }
        // Fallback for non-secure contexts
        const ta = document.createElement("textarea");
        ta.value = text;
        ta.style.position = "fixed";
        ta.style.left = "-9999px";
        document.body.appendChild(ta);
        ta.select();
        document.execCommand("copy");
        document.body.removeChild(ta);
        return Promise.resolve();
    }

    window.handoff = async function (host, session) {
        try {
            const res = await authFetch(`/api/hosts/${eu(host)}/sessions/${eu(session)}/handoff`);
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();
            await copyToClipboard(data.command);
            toast("Copied: " + data.command, "success");
            addLog("info", `Copied handoff command for ${session}@${host}`);
        } catch (e) {
            toast("Failed to copy handoff command: " + e.message, "error");
        }
    };

    window.killSession = async function (host, session) {
        if (!confirm(`End session "${session}" on ${host}?`)) return;
        try {
            await authFetch(`/api/hosts/${eu(host)}/sessions/${eu(session)}`, { method: "DELETE" });
            toast("Session killed", "success");
            addLog("warn", `Killed session ${session}@${host}`);
            fetchAll();
        } catch (e) {
            toast("Failed to kill session", "error");
        }
    };

    window.newSessionFor = async function (host) {
        const name = "s-" + Date.now().toString(36);
        try {
            const res = await authFetch(`/api/hosts/${eu(host)}/sessions`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name }),
            });
            if (!res.ok) throw new Error(await res.text());
            window.open(`/terminal/${eu(host)}/${eu(name)}`, "_blank");
            addLog("info", `Created session ${name}@${host}`);
            fetchAll();
        } catch (e) {
            toast("Failed: " + e.message, "error");
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

        const currentOS = h.config.os || "";
        const detectedOS = h.detected_os || "";
        const osOptions = ["", "Ubuntu", "Debian", "CentOS", "RHEL", "Oracle Linux", "Fedora", "Alpine", "Arch Linux", "SUSE", "FreeBSD", "macOS", "Windows", "Linux"];
        const osSelectHTML = osOptions.map(o => {
            const label = o === "" ? (detectedOS ? `Auto-detect (${detectedOS})` : "Auto-detect") : o;
            return `<option value="${ea(o)}" ${o === currentOS ? "selected" : ""}>${esc(label)}</option>`;
        }).join("");

        modalFields.innerHTML = `
            <label for="m-addr">Host Address</label>
            <input type="text" id="m-addr" value="${ea(h.config.address)}" required>
            <label for="m-port">SSH Port</label>
            <input type="number" id="m-port" value="${h.config.port || 22}" min="1" max="65535">
            <label for="m-user">User</label>
            <input type="text" id="m-user" value="${ea(h.config.user)}">
            ${keypairHTML}
            <label for="m-os">Operating System</label>
            <select id="m-os">${osSelectHTML}</select>
            <hr style="border:0;border-top:1px solid var(--border);margin:16px 0 8px">
            <button type="button" class="btn btn-danger" id="m-delete-host" style="width:100%">Delete Host</button>
        `;

        setTimeout(() => {
            document.getElementById("m-delete-host").addEventListener("click", async () => {
                if (!confirm('Delete host "' + hostName + '"? This cannot be undone.')) return;
                try {
                    const res = await authFetch('/api/hosts/' + eu(hostName), { method: 'DELETE' });
                    if (!res.ok) throw new Error(await res.text());
                    toast('Host "' + hostName + '" deleted', 'success');
                    addLog("warn", `Deleted host ${hostName}`);
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
                os: document.getElementById("m-os").value,
            };
            const kpEl = document.getElementById("m-keyname");
            if (kpEl) body.key_name = kpEl.value;

            const res = await authFetch(`/api/hosts/${eu(hostName)}`, {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(body),
            });
            if (!res.ok) throw new Error(await res.text());
            toast("Host updated", "success");
            addLog("info", `Updated host ${hostName}`);
            fetchAll();
        };

        modal.classList.remove("hidden");
        setTimeout(() => document.getElementById("m-addr").focus(), 50);
    };

    // ── New Session Modal (from top bar) ──
    window.showNewSessionModal = function () {
        if (hosts.length === 0) {
            // Show add host modal instead
            showAddHostModal();
            return;
        }

        modalTitle.textContent = "New Session";
        modalSubmit.textContent = "Create & Connect";

        const hostOptions = hosts.filter(h => h.online && h.tmux_detected).map(h =>
            `<option value="${ea(h.config.name)}">${esc(h.config.name)}</option>`
        ).join("");

        if (!hostOptions) {
            toast("No online hosts with tmux available", "error");
            return;
        }

        modalFields.innerHTML = `
            <label for="m-host">Host</label>
            <select id="m-host">${hostOptions}</select>
        `;

        modalHandler = async () => {
            const host = document.getElementById("m-host").value;
            if (!host) return;
            await window.newSessionFor(host);
        };

        modal.classList.remove("hidden");
    };

    // ── Add Host Modal ──
    function showAddHostModal() {
        modalTitle.textContent = "Add Host";
        modalSubmit.textContent = "Add";

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

            const res = await authFetch("/api/hosts", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(body),
            });
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();
            toast(`Host "${data.name}" added`, "success");
            addLog("info", `Added host ${data.name}`);
            fetchAll();
        };

        modal.classList.remove("hidden");
        setTimeout(() => document.getElementById("m-addr").focus(), 50);
    }

    // Also expose add host from hosts view if needed
    window.showAddHostModal = showAddHostModal;

    // ── Search ──
    window.onSearch = function () {
        renderCurrentView();
    };

    // ── Mobile sidebar ──
    window.toggleSidebar = function () {
        document.getElementById("sidebar").classList.toggle("open");
        document.getElementById("sidebar-overlay").classList.toggle("open");
    };

    window.closeSidebar = function () {
        document.getElementById("sidebar").classList.remove("open");
        document.getElementById("sidebar-overlay").classList.remove("open");
    };

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
        if (hours < 24) return hours + "h ago";
        const days = Math.floor(hours / 24);
        return days + "d ago";
    }

    function toast(msg, type) {
        const el = document.createElement("div");
        el.className = `toast toast-${type}`;
        el.textContent = msg;
        document.body.appendChild(el);
        setTimeout(() => el.remove(), 3000);
    }

    // ── Version ──
    async function fetchVersion() {
        try {
            const res = await authFetch("/api/version");
            const data = await res.json();
            const el = document.getElementById("version-footer");
            if (el && data.version) {
                var v = data.version;
                el.textContent = v.startsWith("v") ? v : "v" + v;
            }
        } catch (e) { /* ignore */ }
    }

    // ── Init ──
    fetchAll();
    fetchVersion();
    setInterval(fetchAll, 5000);
})();
