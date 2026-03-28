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

    // ── URL routing ──
    function updateURL() {
        const params = new URLSearchParams();
        if (currentFilter !== "all") params.set("filter", currentFilter);

        const search = document.getElementById("global-search").value.trim();
        if (search) params.set("search", search);

        if (currentView === "sessions") {
            const host = document.getElementById("session-host-select")?.value;
            const status = document.getElementById("session-status-select")?.value;
            const q = document.getElementById("session-filter")?.value?.trim();
            if (host) params.set("host", host);
            if (status) params.set("status", status);
            if (q) params.set("q", q);
        } else if (currentView === "hosts") {
            const os = document.getElementById("host-os-select")?.value;
            const status = document.getElementById("host-status-select")?.value;
            const sess = document.getElementById("host-sessions-select")?.value;
            const q = document.getElementById("host-filter")?.value?.trim();
            if (os) params.set("os", os);
            if (status) params.set("status", status);
            if (sess) params.set("sessions", sess);
            if (q) params.set("q", q);
        }

        const hash = "#" + currentView + (params.toString() ? "?" + params.toString() : "");
        history.replaceState(null, "", hash);
    }

    function loadFromURL() {
        const hash = location.hash.replace(/^#/, "");
        if (!hash) return;

        const [path, qs] = hash.split("?");
        const params = new URLSearchParams(qs || "");

        const view = path || "dashboard";
        if (["dashboard", "sessions", "hosts", "logs"].includes(view)) {
            currentView = view;
        }

        if (params.has("filter")) {
            const f = params.get("filter");
            if (["all", "active", "favorites"].includes(f)) {
                currentFilter = f;
            }
        }

        // Apply view filter after DOM is ready
        setTimeout(() => {
            if (params.has("search")) {
                document.getElementById("global-search").value = params.get("search");
            }
            if (currentView === "sessions") {
                if (params.has("host")) {
                    const el = document.getElementById("session-host-select");
                    if (el) el.value = params.get("host");
                }
                if (params.has("status")) {
                    const el = document.getElementById("session-status-select");
                    if (el) el.value = params.get("status");
                }
                if (params.has("q")) {
                    const el = document.getElementById("session-filter");
                    if (el) el.value = params.get("q");
                }
            } else if (currentView === "hosts") {
                if (params.has("os")) {
                    const el = document.getElementById("host-os-select");
                    if (el) el.value = params.get("os");
                }
                if (params.has("status")) {
                    const el = document.getElementById("host-status-select");
                    if (el) el.value = params.get("status");
                }
                if (params.has("sessions")) {
                    const el = document.getElementById("host-sessions-select");
                    if (el) el.value = params.get("sessions");
                }
                if (params.has("q")) {
                    const el = document.getElementById("host-filter");
                    if (el) el.value = params.get("q");
                }
            }
            activateView();
            updateFilterChips();
        }, 0);
    }

    function activateView() {
        document.querySelectorAll(".view-panel").forEach(el => el.classList.remove("active"));
        const panel = document.getElementById("view-" + currentView);
        if (panel) panel.classList.add("active");

        document.querySelectorAll(".nav-item[data-view]").forEach(el => {
            el.classList.toggle("active", el.dataset.view === currentView);
        });

        document.querySelectorAll(".filter-item").forEach(el => {
            el.classList.toggle("active", el.dataset.filter === currentFilter);
        });
    }

    // ── Filter state ──
    function updateFilterChips() {
        updateFilterToggleState();
    }

    window.clearFilter = function (id) {
        const el = document.getElementById(id);
        if (!el) return;
        el.value = "";
        renderCurrentView();
    };

    window.toggleFilterBar = function (view) {
        const bar = document.getElementById(view === "session" ? "session-filter-bar" : "host-filter-bar");
        if (bar) bar.classList.toggle("open");
    };

    function updateFilterToggleState() {
        // Session toggle
        const sessionToggle = document.getElementById("session-filter-toggle");
        if (sessionToggle) {
            const hasFilters = !!(
                document.getElementById("session-host-select")?.value ||
                document.getElementById("session-status-select")?.value
            );
            sessionToggle.classList.toggle("has-filters", hasFilters);
            // Auto-open if filters are active
            if (hasFilters) {
                document.getElementById("session-filter-bar")?.classList.add("open");
            }
        }

        // Host toggle
        const hostToggle = document.getElementById("host-filter-toggle");
        if (hostToggle) {
            const hasFilters = !!(
                document.getElementById("host-os-select")?.value ||
                document.getElementById("host-status-select")?.value ||
                document.getElementById("host-sessions-select")?.value
            );
            hostToggle.classList.toggle("has-filters", hasFilters);
            if (hasFilters) {
                document.getElementById("host-filter-bar")?.classList.add("open");
            }
        }
    }

    // ── View switching ──
    window.switchView = function (view) {
        currentView = view;
        activateView();
        renderCurrentView();
        updateURL();
        closeSidebar();
    };

    // Navigate to sessions view filtered by a specific host
    window.showSessionsForHost = function (hostName) {
        currentView = "sessions";
        activateView();
        populateFilterDropdowns();
        const hostSelect = document.getElementById("session-host-select");
        if (hostSelect) hostSelect.value = hostName;
        renderSessions();
        updateFilterChips();
        updateURL();
        closeSidebar();
    };

    function renderCurrentView() {
        populateFilterDropdowns();
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
        updateURL();
    };

    // ── Populate filter dropdowns ──
    function populateFilterDropdowns() {
        // Session host dropdown
        const hostSelect = document.getElementById("session-host-select");
        if (hostSelect) {
            const currentVal = hostSelect.value;
            const hostNames = [...new Set(sessions.map(s => s.host_name))].sort();
            hostSelect.innerHTML = '<option value="">All Hosts</option>' +
                hostNames.map(h => `<option value="${ea(h)}" ${h === currentVal ? "selected" : ""}>${esc(h)}</option>`).join("");
        }

        // Host OS dropdown (custom with icons)
        if (document.getElementById("host-os-select-wrap")) {
            buildOsFilterDropdown();
        }
    }

    function getFilteredSessions() {
        let filtered = sessions.slice();
        const search = (document.getElementById("global-search").value || "").toLowerCase();
        const viewFilter = (document.getElementById("session-filter")?.value || "").toLowerCase();
        const hostFilter = document.getElementById("session-host-select")?.value || "";
        const statusFilter = document.getElementById("session-status-select")?.value || "";

        if (currentFilter === "active") {
            filtered = filtered.filter(s => s.session.attached);
        } else if (currentFilter === "favorites") {
            filtered = filtered.filter(s => isSessionStarred(s.host_name, s.session.name));
        }

        if (hostFilter) {
            filtered = filtered.filter(s => s.host_name === hostFilter);
        }

        if (statusFilter === "attached") {
            filtered = filtered.filter(s => s.session.attached);
        } else if (statusFilter === "detached") {
            filtered = filtered.filter(s => !s.session.attached);
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
        const osFilter = document.getElementById("host-os-select")?.value || "";
        const statusFilter = document.getElementById("host-status-select")?.value || "";
        const sessionsFilter = document.getElementById("host-sessions-select")?.value || "";

        if (currentFilter === "active") {
            filtered = filtered.filter(h => {
                return sessions.some(s => s.host_name === h.config.name);
            });
        }

        if (osFilter) {
            filtered = filtered.filter(h => {
                const os = h.config.os || h.detected_os || "";
                return os === osFilter;
            });
        }

        if (statusFilter === "online") {
            filtered = filtered.filter(h => h.online);
        } else if (statusFilter === "offline") {
            filtered = filtered.filter(h => !h.online);
        }

        if (sessionsFilter === "with") {
            filtered = filtered.filter(h => sessions.some(s => s.host_name === h.config.name));
        } else if (sessionsFilter === "without") {
            filtered = filtered.filter(h => !sessions.some(s => s.host_name === h.config.name));
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

        // Recent sessions (last 10, starred first)
        const recent = sessions.slice().sort((a, b) => {
            const sa = isSessionStarred(a.host_name, a.session.name) ? 1 : 0;
            const sb = isSessionStarred(b.host_name, b.session.name) ? 1 : 0;
            if (sa !== sb) return sb - sa;
            return new Date(b.session.created) - new Date(a.session.created);
        }).slice(0, 10);

        const body = document.getElementById("dashboard-recent-body");
        if (recent.length === 0) {
            body.innerHTML = `<tr><td colspan="5" class="empty-state">No active sessions</td></tr>`;
            return;
        }

        body.innerHTML = recent.map(s => {
            const host = hosts.find(h => h.config.name === s.host_name);
            const statusClass = s.session.attached ? "attached" : "detached";
            const age = timeAgo(new Date(s.session.created));
            const hostIcon = host ? (host.config.icon || "terminal") : "terminal";
            const hostColor = host ? (host.config.icon_color || "default") : "default";
            const sessionIcon = getSessionIcon(s.host_name, s.session.name) || hostIcon;
            const sessionColor = getSessionIconColor(s.host_name, s.session.name) || hostColor;
            const hasCustom = sessionIcon !== "terminal" || sessionColor !== "default";
            const sTip = sessionStatusTip(s.session.attached);
            const starred = isSessionStarred(s.host_name, s.session.name);
            const starClass = starred ? "star-btn starred" : "star-btn";
            return `<tr>
                <td><span class="status-dot ${statusClass}" title="${ea(sTip)}"></span></td>
                <td><button class="session-icon-btn${hasCustom ? ' has-icon' : ''}" onclick="pickSessionIcon(this,'${ea(s.host_name)}','${ea(s.session.name)}')" title="Change icon">${renderIcon(sessionIcon, 16, sessionColor)}</button><a class="session-link" href="/terminal/${eu(s.host_name)}/${eu(s.session.name)}" target="_blank">${esc(s.session.name)}</a></td>
                <td class="hide-mobile">${esc(s.host_name)}</td>
                <td class="hide-mobile">${age}</td>
                <td>
                    <div class="action-group">
                        <button class="${starClass}" onclick="toggleStar('${ea(s.host_name)}','${ea(s.session.name)}')" title="${starred ? 'Unstar' : 'Star'}"><svg width="14" height="14" viewBox="0 0 24 24" fill="${starred ? 'currentColor' : 'none'}" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon></svg></button>
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
        updateFilterChips();
        updateURL();
        let filtered = getFilteredSessions();

        // Sort — starred sessions float to top unless explicitly sorting by another column
        filtered.sort((a, b) => {
            // If not sorting by starred, still pin starred to top as secondary
            if (sessionSort.key !== "starred") {
                const sa = isSessionStarred(a.host_name, a.session.name) ? 1 : 0;
                const sb = isSessionStarred(b.host_name, b.session.name) ? 1 : 0;
                if (sa !== sb) return sb - sa;
            }

            let va, vb;
            switch (sessionSort.key) {
                case "name": va = a.session.name; vb = b.session.name; break;
                case "host": va = a.host_name; vb = b.host_name; break;
                case "uptime": va = new Date(a.session.created); vb = new Date(b.session.created); return (va - vb) * sessionSort.dir;
                case "status": va = a.session.attached ? 1 : 0; vb = b.session.attached ? 1 : 0; return (vb - va) * sessionSort.dir;
                case "starred":
                    va = isSessionStarred(a.host_name, a.session.name) ? 1 : 0;
                    vb = isSessionStarred(b.host_name, b.session.name) ? 1 : 0;
                    return (vb - va) * sessionSort.dir;
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

            const hostName = host ? host.config.name : s.host_name;
            const showName = hostName !== hostAddr && hostName ? hostName : "";
            const hostIcon = host ? (host.config.icon || "terminal") : "terminal";
            const hostColor = host ? (host.config.icon_color || "default") : "default";
            const sessionIcon = getSessionIcon(s.host_name, s.session.name) || hostIcon;
            const sessionColor = getSessionIconColor(s.host_name, s.session.name) || hostColor;
            const hasCustom = sessionIcon !== "terminal" || sessionColor !== "default";
            const sTip2 = sessionStatusTip(s.session.attached);

            const starred = isSessionStarred(s.host_name, s.session.name);
            const starClass = starred ? "star-btn starred" : "star-btn";

            return `<tr>
                <td><span class="status-dot ${statusClass}" title="${ea(sTip2)}"></span></td>
                <td><button class="session-icon-btn${hasCustom ? ' has-icon' : ''}" onclick="pickSessionIcon(this,'${ea(s.host_name)}','${ea(s.session.name)}')" title="Change icon">${renderIcon(sessionIcon, 16, sessionColor)}</button><a class="session-link" href="/terminal/${eu(s.host_name)}/${eu(s.session.name)}" target="_blank">${esc(s.session.name)}</a></td>
                <td class="hide-mobile">
                    <div class="host-cell">
                        <span class="host-fqdn">${esc(hostAddr || s.host_name)}</span>
                        <span class="host-ip">${showName ? esc(showName) : ''}</span>
                    </div>
                </td>
                <td class="hide-mobile">${age}</td>
                <td>
                    <div class="action-group">
                        <button class="${starClass}" onclick="toggleStar('${ea(s.host_name)}','${ea(s.session.name)}')" title="${starred ? 'Unstar' : 'Star'}"><svg width="14" height="14" viewBox="0 0 24 24" fill="${starred ? 'currentColor' : 'none'}" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon></svg></button>
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
        updateFilterChips();
        updateURL();
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

            const displayName = name !== addr ? name : "";

            const hostIconName = h.config.icon || "terminal";
            const hostIconColor = h.config.icon_color || "default";

            const hTip = hostStatusTip(h);

            return `<tr>
                <td>
                    <div class="host-cell">
                        <span class="host-fqdn">
                            <span class="status-dot ${statusClass}" style="margin-right:8px" title="${ea(hTip)}"></span>
                            <span style="margin-right:6px;display:inline-flex;vertical-align:middle">${renderIcon(hostIconName, 16, hostIconColor)}</span>
                            ${esc(addr)}${esc(port)}
                        </span>
                        <span class="host-ip">${displayName ? esc(displayName) + ' &middot; ' : ''}${esc(user ? user + "@" : "")}${esc(addr)}</span>
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

    // OS icons from Simple Icons (https://simpleicons.org) — real logo paths
    // Each rendered as white fill on colored circle, scaled to fit inside a 24x24 viewBox circle
    const OS_ICONS = {
        "Ubuntu": { color: "#E95420", svg: '<circle cx="12" cy="12" r="10" fill="#E95420"/><g transform="translate(3.6,3.6) scale(0.7)" fill="#fff"><path d="M17.61.455a3.41 3.41 0 0 0-3.41 3.41 3.41 3.41 0 0 0 3.41 3.41 3.41 3.41 0 0 0 3.41-3.41 3.41 3.41 0 0 0-3.41-3.41zM12.92.8C8.923.777 5.137 2.941 3.148 6.451a4.5 4.5 0 0 1 .26-.007 4.92 4.92 0 0 1 2.585.737A8.316 8.316 0 0 1 12.688 3.6 4.944 4.944 0 0 1 13.723.834 11.008 11.008 0 0 0 12.92.8zm9.226 4.994a4.915 4.915 0 0 1-1.918 2.246 8.36 8.36 0 0 1-.273 8.303 4.89 4.89 0 0 1 1.632 2.54 11.156 11.156 0 0 0 .559-13.089zM3.41 7.932A3.41 3.41 0 0 0 0 11.342a3.41 3.41 0 0 0 3.41 3.409 3.41 3.41 0 0 0 3.41-3.41 3.41 3.41 0 0 0-3.41-3.41zm2.027 7.866a4.908 4.908 0 0 1-2.915.358 11.1 11.1 0 0 0 7.991 6.698 11.234 11.234 0 0 0 2.422.249 4.879 4.879 0 0 1-.999-2.85 8.484 8.484 0 0 1-.836-.136 8.304 8.304 0 0 1-5.663-4.32zm11.405.928a3.41 3.41 0 0 0-3.41 3.41 3.41 3.41 0 0 0 3.41 3.41 3.41 3.41 0 0 0 3.41-3.41 3.41 3.41 0 0 0-3.41-3.41z"/></g>' },
        "Debian": { color: "#A81D33", svg: '<circle cx="12" cy="12" r="10" fill="#A81D33"/><g transform="translate(3.6,3.6) scale(0.7)" fill="#fff"><path d="M13.88 12.685c-.4 0 .08.2.601.28.14-.1.27-.22.39-.33a3.001 3.001 0 01-.99.05m2.14-.53c.23-.33.4-.69.47-1.06-.06.27-.2.5-.33.73-.75.47-.07-.27 0-.56-.8 1.01-.11.6-.14.89m.781-2.05c.05-.721-.14-.501-.2-.221.07.04.13.5.2.22M12.38.31c.2.04.45.07.42.12.23-.05.28-.1-.43-.12m.43.12l-.15.03.14-.01V.43m6.633 9.944c.02.64-.2.95-.38 1.5l-.35.181c-.28.54.03.35-.17.78-.44.39-1.34 1.22-1.62 1.301-.201 0 .14-.25.19-.34-.591.4-.481.6-1.371.85l-.03-.06c-2.221 1.04-5.303-1.02-5.253-3.842-.03.17-.07.13-.12.2a3.551 3.552 0 012.001-3.501 3.361 3.362 0 013.732.48 3.341 3.342 0 00-2.721-1.3c-1.18.01-2.281.76-2.651 1.57-.6.38-.67 1.47-.93 1.661-.361 2.601.66 3.722 2.38 5.042.27.19.08.21.12.35a4.702 4.702 0 01-1.53-1.16c.23.33.47.66.8.91-.55-.18-1.27-1.3-1.48-1.35.93 1.66 3.78 2.921 5.261 2.3a6.203 6.203 0 01-2.33-.28c-.33-.16-.77-.51-.7-.57a5.802 5.803 0 005.902-.84c.44-.35.93-.94 1.07-.95-.2.32.04.16-.12.44.44-.72-.2-.3.46-1.24l.24.33c-.09-.6.74-1.321.66-2.262.19-.3.2.3 0 .97.29-.74.08-.85.15-1.46.08.2.18.42.23.63-.18-.7.2-1.2.28-1.6-.09-.05-.28.3-.32-.53 0-.37.1-.2.14-.28-.08-.05-.26-.32-.38-.861.08-.13.22.33.34.34-.08-.42-.2-.75-.2-1.08-.34-.68-.12.1-.4-.3-.34-1.091.3-.25.34-.74.54.77.84 1.96.981 2.46-.1-.6-.28-1.2-.49-1.76.16.07-.26-1.241.21-.37A7.823 7.824 0 0017.702 1.6c.18.17.42.39.33.42-.75-.45-.62-.48-.73-.67-.61-.25-.65.02-1.06 0C15.082.73 14.862.8 13.8.4l.05.23c-.77-.25-.9.1-1.73 0-.05-.04.27-.14.53-.18-.741.1-.701-.14-1.431.03.17-.13.36-.21.55-.32-.6.04-1.44.35-1.18.07C9.6.68 7.847 1.3 6.867 2.22L6.838 2c-.45.54-1.96 1.611-2.08 2.311l-.131.03c-.23.4-.38.85-.57 1.261-.3.52-.45.2-.4.28-.6 1.22-.9 2.251-1.16 3.102.18.27 0 1.65.07 2.76-.3 5.463 3.84 10.776 8.363 12.006.67.23 1.65.23 2.49.25-.99-.28-1.12-.15-2.08-.49-.7-.32-.85-.7-1.34-1.13l.2.35c-.971-.34-.57-.42-1.361-.67l.21-.27c-.31-.03-.83-.53-.97-.81l-.34.01c-.41-.501-.63-.871-.61-1.161l-.111.2c-.13-.21-1.52-1.901-.8-1.511-.13-.12-.31-.2-.5-.55l.14-.17c-.35-.44-.64-1.02-.62-1.2.2.24.32.3.45.33-.88-2.172-.93-.12-1.601-2.202l.15-.02c-.1-.16-.18-.34-.26-.51l.06-.6c-.63-.74-.18-3.102-.09-4.402.07-.54.53-1.1.88-1.981l-.21-.04c.4-.71 2.341-2.872 3.241-2.761.43-.55-.09 0-.18-.14.96-.991 1.26-.7 1.901-.88.7-.401-.6.16-.27-.151 1.2-.3.85-.7 2.421-.85.16.1-.39.14-.52.26 1-.49 3.151-.37 4.562.27 1.63.77 3.461 3.011 3.531 5.132l.08.02c-.04.85.13 1.821-.17 2.711l.2-.42M9.54 13.236l-.05.28c.26.35.47.73.8 1.01-.24-.47-.42-.66-.75-1.3m.62-.02c-.14-.15-.22-.34-.31-.52.08.32.26.6.43.88l-.12-.36m10.945-2.382l-.07.15c-.1.76-.34 1.511-.69 2.212.4-.73.65-1.541.75-2.362M12.45.12c.27-.1.66-.05.95-.12-.37.03-.74.05-1.1.1l.15.02M3.006 5.142c.07.57-.43.8.11.42.3-.66-.11-.18-.1-.42m-.64 2.661c.12-.39.15-.62.2-.84-.35.44-.17.53-.2.83"/></g>' },
        "CentOS": { color: "#262577", svg: '<circle cx="12" cy="12" r="10" fill="#262577"/><g transform="translate(3.6,3.6) scale(0.7)" fill="#fff"><path d="M12.076.066L8.883 3.28H3.348v5.434L0 12.01l3.349 3.298v5.39h5.374l3.285 3.236 3.285-3.236h5.43v-5.374L24 12.026l-3.232-3.252V3.321H15.31zm0 .749l2.49 2.506h-1.69v6.441l-.8.805-.81-.815V3.28H9.627zm-8.2 2.991h4.483L6.485 5.692l4.253 4.279v.654H9.94L5.674 6.423l-1.798 1.77zm5.227 0h1.635v5.415l-3.509-3.53zm4.302.043h1.687l1.83 1.842-3.517 3.539zm2.431 0h4.404v4.394l-1.83-1.842-4.241 4.267h-.764v-.69l4.261-4.287zm2.574 3.3l1.83 1.843v1.676h-5.327zm-12.735.013l3.515 3.462H3.876v-1.69zM3.348 9.454v1.697h6.377l.871.858-.782.77H3.35v1.786L.753 12.01zm17.42.068l2.488 2.503-2.533 2.55v-1.796h-6.41l-.75-.754.825-.83h6.38zm-9.502.978l.81.815.186-.188.614-.618v.686h.768l-.825.83.75.754h-.719v.808l-.842-.83-.741.73v-.707h-.7l.781-.77-.188-.186-.682-.672h.788zm-7.39 2.807h5.402l-3.603 3.55-1.798-1.772zm6.154 0h.708v.7l-4.404 4.338 1.852 1.824h-4.31v-4.342l1.798 1.77zm3.348 0h.715l4.317 4.343.186-.187 1.599-1.61v4.316h-4.366l1.853-1.825-.188-.185-4.116-4.054zm1.46 0h5.357v1.798l-1.785 1.796zm-2.83.191l.842.829v6.37h1.691l-2.532 2.495-2.533-2.495h1.79V14.23zm-1.27 1.251v5.42H8.939l-1.852-1.823zm2.64.097l3.552 3.499-1.853 1.825h-1.7z"/></g>' },
        "RHEL": { color: "#EE0000", svg: '<circle cx="12" cy="12" r="10" fill="#EE0000"/><g transform="translate(3.6,3.6) scale(0.7)" fill="#fff"><path d="M16.009 13.386c1.577 0 3.86-.326 3.86-2.202a1.765 1.765 0 0 0-.04-.431l-.94-4.08c-.216-.898-.406-1.305-1.982-2.093-1.223-.625-3.888-1.658-4.676-1.658-.733 0-.947.946-1.822.946-.842 0-1.467-.706-2.255-.706-.757 0-1.25.515-1.63 1.576 0 0-1.06 2.99-1.197 3.424a.81.81 0 0 0-.028.245c0 1.162 4.577 4.974 10.71 4.974m4.101-1.435c.218 1.032.218 1.14.218 1.277 0 1.765-1.984 2.745-4.593 2.745-5.895.004-11.06-3.451-11.06-5.734a2.326 2.326 0 0 1 .19-.925C2.746 9.415 0 9.794 0 12.217c0 3.969 9.405 8.861 16.851 8.861 5.71 0 7.149-2.582 7.149-4.62 0-1.605-1.387-3.425-3.887-4.512"/></g>' },
        "Oracle Linux": { color: "#F80000", svg: '<circle cx="12" cy="12" r="10" fill="#F80000"/><g transform="translate(3.6,3.6) scale(0.7)" fill="#fff"><path d="M16.412 4.412h-8.82a7.588 7.588 0 0 0-.008 15.176h8.828a7.588 7.588 0 0 0 0-15.176zm-.193 12.502H7.786a4.915 4.915 0 0 1 0-9.828h8.433a4.914 4.914 0 1 1 0 9.828z"/></g>' },
        "Alpine": { color: "#0D597F", svg: '<circle cx="12" cy="12" r="10" fill="#0D597F"/><g transform="translate(3.6,3.6) scale(0.7)" fill="#fff"><path d="M5.998 1.607L0 12l5.998 10.393h12.004L24 12 18.002 1.607H5.998zM9.965 7.12L12.66 9.9l1.598 1.595.002-.002 2.41 2.363c-.2.14-.386.252-.563.344a3.756 3.756 0 01-.496.217 2.702 2.702 0 01-.425.111c-.131.023-.25.034-.358.034-.13 0-.242-.014-.338-.034a1.317 1.317 0 01-.24-.072.95.95 0 01-.2-.113l-1.062-1.092-3.039-3.041-1.1 1.053-3.07 3.072a.974.974 0 01-.2.111 1.274 1.274 0 01-.237.073c-.096.02-.209.033-.338.033-.108 0-.227-.009-.358-.031a2.7 2.7 0 01-.425-.114 3.748 3.748 0 01-.496-.217 5.228 5.228 0 01-.563-.343l6.803-6.727zm4.72.785l4.579 4.598 1.382 1.353a5.24 5.24 0 01-.564.344 3.73 3.73 0 01-.494.217 2.697 2.697 0 01-.426.111c-.13.023-.251.034-.36.034-.129 0-.241-.014-.337-.034a1.285 1.285 0 01-.385-.146c-.033-.02-.05-.036-.053-.04l-1.232-1.218-2.111-2.111-.334.334L12.79 9.8l1.896-1.897zm-5.966 4.12v2.529a2.128 2.128 0 01-.356-.035 2.765 2.765 0 01-.422-.116 3.708 3.708 0 01-.488-.214 5.217 5.217 0 01-.555-.34l1.82-1.825Z"/></g>' },
        "Arch Linux": { color: "#1793D1", svg: '<circle cx="12" cy="12" r="10" fill="#1793D1"/><g transform="translate(3.6,3.6) scale(0.7)" fill="#fff"><path d="M11.39.605C10.376 3.092 9.764 4.72 8.635 7.132c.693.734 1.543 1.589 2.923 2.554-1.484-.61-2.496-1.224-3.252-1.86C6.86 10.842 4.596 15.138 0 23.395c3.612-2.085 6.412-3.37 9.021-3.862a6.61 6.61 0 01-.171-1.547l.003-.115c.058-2.315 1.261-4.095 2.687-3.973 1.426.12 2.534 2.096 2.478 4.409a6.52 6.52 0 01-.146 1.243c2.58.505 5.352 1.787 8.914 3.844-.702-1.293-1.33-2.459-1.929-3.57-.943-.73-1.926-1.682-3.933-2.713 1.38.359 2.367.772 3.137 1.234-6.09-11.334-6.582-12.84-8.67-17.74z"/></g>' },
        "Fedora": { color: "#51A2DA", svg: '<circle cx="12" cy="12" r="10" fill="#51A2DA"/><g transform="translate(3.6,3.6) scale(0.7)" fill="#fff"><path d="M12.001 0C5.376 0 .008 5.369.004 11.992H.002v9.287h.002A2.726 2.726 0 0 0 2.73 24h9.275c6.626-.004 11.993-5.372 11.993-11.997C23.998 5.375 18.628 0 12 0zm2.431 4.94c2.015 0 3.917 1.543 3.917 3.671 0 .197.001.395-.03.619a1.002 1.002 0 0 1-1.137.893 1.002 1.002 0 0 1-.842-1.175 2.61 2.61 0 0 0 .013-.337c0-1.207-.987-1.672-1.92-1.672-.934 0-1.775.784-1.777 1.672.016 1.027 0 2.046 0 3.07l1.732-.012c1.352-.028 1.368 2.009.016 1.998l-1.748.013c-.004.826.006.677.002 1.093 0 0 .015 1.01-.016 1.776-.209 2.25-2.124 4.046-4.424 4.046-2.438 0-4.448-1.993-4.448-4.437.073-2.515 2.078-4.492 4.603-4.469l1.409-.01v1.996l-1.409.013h-.007c-1.388.04-2.577.984-2.6 2.47a2.438 2.438 0 0 0 2.452 2.439c1.356 0 2.441-.987 2.441-2.437l-.001-7.557c0-.14.005-.252.02-.407.23-1.848 1.883-3.256 3.754-3.256z"/></g>' },
        "SUSE": { color: "#73BA25", svg: '<circle cx="12" cy="12" r="10" fill="#73BA25"/><g transform="translate(3.6,3.6) scale(0.7)" fill="#fff"><path d="M10.724 0a12 12 0 0 0-9.448 4.623c1.464.391 2.5.727 2.81.832.005-.19.037-1.893.037-1.893s.004-.04.025-.06c.026-.026.065-.018.065-.018.385.056 8.602 1.274 12.066 3.292.427.25.638.517.902.786.958.99 2.223 5.108 2.359 5.957.005.033-.036.07-.054.083a5.177 5.177 0 0 1-.313.228c-.82.55-2.708 1.872-5.13 1.656-2.176-.193-5.018-1.44-8.445-3.699.336.79.668 1.58 1 2.371.497.258 5.287 2.7 7.651 2.651 1.904-.04 3.941-.968 4.756-1.458 0 0 .179-.108.257-.048.085.066.061.167.041.27-.05.234-.164.66-.242.863l-.065.165c-.093.25-.183.482-.356.625-.48.436-1.246.784-2.446 1.305-1.855.812-4.865 1.328-7.66 1.31-1.001-.022-1.968-.133-2.817-.232-1.743-.197-3.161-.357-4.026.269A12 12 0 0 0 10.724 24a12 12 0 0 0 12-12 12 12 0 0 0-12-12zM13.4 6.963a3.503 3.503 0 0 0-2.521.942 3.498 3.498 0 0 0-1.114 2.449 3.528 3.528 0 0 0 3.39 3.64 3.48 3.48 0 0 0 2.524-.946 3.504 3.504 0 0 0 1.114-2.446 3.527 3.527 0 0 0-3.393-3.64zm-.03 1.035a2.458 2.458 0 0 1 2.368 2.539 2.43 2.43 0 0 1-.774 1.706 2.456 2.456 0 0 1-1.762.659 2.461 2.461 0 0 1-2.364-2.542c.02-.655.3-1.26.777-1.707a2.419 2.419 0 0 1 1.756-.655zm.402 1.23c-.602 0-1.087.325-1.087.727 0 .4.485.725 1.087.725.6 0 1.088-.326 1.088-.725 0-.402-.487-.726-1.088-.726Z"/></g>' },
        "FreeBSD": { color: "#AB2B28", svg: '<circle cx="12" cy="12" r="10" fill="#AB2B28"/><g transform="translate(3.6,3.6) scale(0.7)" fill="#fff"><path d="M23.682 2.406c-.001-.149-.097-.187-.24-.189h-.25v.659h.108v-.282h.102l.17.282h.122l-.184-.29c.102-.012.175-.065.172-.18zm-.382.096v-.193h.13c.06-.002.145.011.143.089.005.09-.08.107-.153.103h-.12zM21.851 1.49c1.172 1.171-2.077 6.319-2.626 6.869-.549.548-1.944.044-3.115-1.128-1.172-1.171-1.676-2.566-1.127-3.115.549-.55 5.697-3.798 6.868-2.626zM1.652 6.61C.626 4.818-.544 2.215.276 1.395c.81-.81 3.355.319 5.144 1.334A11.003 11.003 0 0 0 1.652 6.61zm18.95.418a10.584 10.584 0 0 1 1.368 5.218c0 5.874-4.762 10.636-10.637 10.636C5.459 22.882.697 18.12.697 12.246.697 6.371 5.459 1.61 11.333 1.61c1.771 0 3.441.433 4.909 1.199-.361.201-.69.398-.969.574-.428-.077-.778-.017-.998.202-.402.402-.269 1.245.263 2.2.273.539.701 1.124 1.25 1.674.103.104.208.202.315.297 1.519 1.446 3.205 2.111 3.829 1.486.267-.267.297-.728.132-1.287.167-.27.35-.584.538-.927z"/></g>' },
        "macOS": { color: "#555", svg: '<circle cx="12" cy="12" r="10" fill="#555"/><g transform="translate(4.2,3.6) scale(0.65)" fill="#fff"><path d="M12.152 6.896c-.948 0-2.415-1.078-3.96-1.04-2.04.027-3.91 1.183-4.961 3.014-2.117 3.675-.546 9.103 1.519 12.09 1.013 1.454 2.208 3.09 3.792 3.039 1.52-.065 2.09-.987 3.935-.987 1.831 0 2.35.987 3.96.948 1.637-.026 2.676-1.48 3.676-2.948 1.156-1.688 1.636-3.325 1.662-3.415-.039-.013-3.182-1.221-3.22-4.857-.026-3.04 2.48-4.494 2.597-4.559-1.429-2.09-3.623-2.324-4.39-2.376-2-.156-3.675 1.09-4.61 1.09zM15.53 3.83c.843-1.012 1.4-2.427 1.245-3.83-1.207.052-2.662.805-3.532 1.818-.78.896-1.454 2.338-1.273 3.714 1.338.104 2.715-.688 3.559-1.701"/></g>' },
        "Windows": { color: "#0078D6", svg: '<circle cx="12" cy="12" r="10" fill="#0078D6"/><g transform="translate(4.2,4.2) scale(0.65)" fill="#fff"><path d="M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801"/></g>' },
        "Linux": { color: "#FCC624", svg: '<circle cx="12" cy="12" r="10" fill="#FCC624"/><g transform="translate(4.8,3.6) scale(0.6)" fill="#333"><path d="M12.504 0c-.155 0-.315.008-.48.021-4.226.333-3.105 4.807-3.17 6.298-.076 1.092-.3 1.953-1.05 3.02-.885 1.051-2.127 2.75-2.716 4.521-.278.832-.41 1.684-.287 2.489a.424.424 0 00-.11.135c-.26.268-.45.6-.663.839-.199.199-.485.267-.797.4-.313.136-.658.269-.864.68-.09.189-.136.394-.132.602 0 .199.027.4.055.536.058.399.116.728.04.97-.249.68-.28 1.145-.106 1.484.174.334.535.47.94.601.81.2 1.91.135 2.774.6.926.466 1.866.67 2.616.47.526-.116.97-.464 1.208-.946.587-.003 1.23-.269 2.26-.334.699-.058 1.574.267 2.577.2.025.134.063.198.114.333l.003.003c.391.778 1.113 1.132 1.884 1.071.771-.06 1.592-.536 2.257-1.306.631-.765 1.683-1.084 2.378-1.503.348-.199.629-.469.649-.853.023-.4-.2-.811-.714-1.376v-.097l-.003-.003c-.17-.2-.25-.535-.338-.926-.085-.401-.182-.786-.492-1.046h-.003c-.059-.054-.123-.067-.188-.135a.357.357 0 00-.19-.064c.431-1.278.264-2.55-.173-3.694-.533-1.41-1.465-2.638-2.175-3.483-.796-1.005-1.576-1.957-1.56-3.368.026-2.152.236-6.133-3.544-6.139z"/></g>' },
    };

    function osIconMatch(osName) {
        if (!osName) return null;
        const lower = osName.toLowerCase();
        for (const key in OS_ICONS) {
            if (lower.includes(key.toLowerCase())) return OS_ICONS[key];
        }
        return null;
    }

    function osIcon(osName, size) {
        size = size || 16;
        const entry = osIconMatch(osName);
        if (!entry) {
            return `<span style="display:inline-block;width:${size}px;height:${size}px;border-radius:50%;background:#888;flex-shrink:0"></span>`;
        }
        return `<svg width="${size}" height="${size}" viewBox="0 0 24 24">${entry.svg}</svg>`;
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

    window.newSessionFor = async function (host, customName) {
        const name = customName || "s-" + Date.now().toString(36);
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
        const osOptionsList = ["", "Ubuntu", "Debian", "CentOS", "RHEL", "Oracle Linux", "Fedora", "Alpine", "Arch Linux", "SUSE", "FreeBSD", "macOS", "Windows", "Linux"];

        const currentHostIcon = h.config.icon || "terminal";
        const currentHostColor = h.config.icon_color || "default";

        modalFields.innerHTML = `
            <label for="m-addr">Host Address (FQDN)</label>
            <input type="text" id="m-addr" value="${ea(h.config.address)}" required>
            <label for="m-hname">Display Name (optional)</label>
            <input type="text" id="m-hname" value="${ea(h.config.name !== h.config.address ? h.config.name : '')}" placeholder="Defaults to address">
            <label for="m-port">SSH Port</label>
            <input type="number" id="m-port" value="${h.config.port || 22}" min="1" max="65535">
            <label for="m-user">User</label>
            <input type="text" id="m-user" value="${ea(h.config.user)}">
            ${keypairHTML}
            <label>Operating System</label>
            <div id="m-os-dropdown"></div>
            <input type="hidden" id="m-os" value="${ea(currentOS)}">
            ${iconSelectorHTML("m-icon", currentHostIcon, currentHostColor)}
            <hr style="border:0;border-top:1px solid var(--border);margin:16px 0 8px">
            <button type="button" class="btn btn-danger" id="m-delete-host" style="width:100%">Delete Host</button>
        `;

        setTimeout(() => {
            attachIconSelector("m-icon");
            buildOsDropdown("m-os-dropdown", "m-os", osOptionsList, currentOS, detectedOS);
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
                icon: document.getElementById("m-icon").value || "",
                icon_color: document.getElementById("m-icon-color").value || "",
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
            // Trigger re-scan to refresh detected OS
            authFetch(`/api/hosts/${eu(hostName)}/scan`, { method: "POST" }).catch(() => {});
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
            <label for="m-sname">Session Name (optional)</label>
            <input type="text" id="m-sname" placeholder="auto-generated if blank">
        `;

        modalHandler = async () => {
            const host = document.getElementById("m-host").value;
            if (!host) return;
            const sname = document.getElementById("m-sname").value.trim();
            await window.newSessionFor(host, sname || null);
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
            <label for="m-addr">Host Address (FQDN)</label>
            <input type="text" id="m-addr" placeholder="myserver.example.com" required>
            <label for="m-hname">Display Name (optional)</label>
            <input type="text" id="m-hname" placeholder="Defaults to address">
            <label for="m-port">SSH Port</label>
            <input type="number" id="m-port" placeholder="22" min="1" max="65535">
            <label for="m-user">User (leave blank for default)</label>
            <input type="text" id="m-user" placeholder="">
            ${keypairHTML}
            ${iconSelectorHTML("m-icon", "terminal")}
        `;

        setTimeout(() => attachIconSelector("m-icon"), 0);

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
            const iconVal = document.getElementById("m-icon")?.value;
            if (iconVal && iconVal !== "terminal") body.icon = iconVal;
            const iconColorVal = document.getElementById("m-icon-color")?.value;
            if (iconColorVal && iconColorVal !== "default") body.icon_color = iconColorVal;

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

    // ── Custom OS dropdown with icons ──
    function buildOsDropdown(containerId, hiddenInputId, options, currentValue, detectedOS) {
        const container = document.getElementById(containerId);
        const hidden = document.getElementById(hiddenInputId);
        if (!container || !hidden) return;

        function labelFor(val) {
            if (val === "") return detectedOS ? `Auto-detect (${detectedOS})` : "Auto-detect";
            return val;
        }

        const selected = currentValue;
        container.innerHTML = `
            <div class="custom-select" id="${containerId}-cs">
                <button type="button" class="custom-select-trigger" id="${containerId}-trigger">
                    ${osIcon(selected || detectedOS || "Linux", 16)}
                    <span>${esc(labelFor(selected))}</span>
                    <svg class="custom-select-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"></polyline></svg>
                </button>
                <div class="custom-select-menu" id="${containerId}-menu">
                    ${options.map(o => {
                        const lbl = labelFor(o);
                        const iconOs = o || detectedOS || "Linux";
                        const sel = o === selected ? " selected" : "";
                        return `<div class="custom-select-option${sel}" data-value="${ea(o)}">${osIcon(iconOs, 16)}<span>${esc(lbl)}</span></div>`;
                    }).join("")}
                </div>
            </div>`;

        const trigger = document.getElementById(`${containerId}-trigger`);
        const menu = document.getElementById(`${containerId}-menu`);

        trigger.addEventListener("click", (e) => {
            e.preventDefault();
            e.stopPropagation();
            menu.classList.toggle("open");
        });

        menu.addEventListener("click", (e) => {
            const opt = e.target.closest(".custom-select-option");
            if (!opt) return;
            const val = opt.dataset.value;
            hidden.value = val;
            trigger.innerHTML = `${osIcon(val || detectedOS || "Linux", 16)}<span>${esc(labelFor(val))}</span><svg class="custom-select-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"></polyline></svg>`;
            menu.querySelectorAll(".custom-select-option").forEach(o => o.classList.toggle("selected", o.dataset.value === val));
            menu.classList.remove("open");
        });

        document.addEventListener("click", (e) => {
            if (!container.contains(e.target)) menu.classList.remove("open");
        });
    }

    // Build OS filter dropdown with icons
    function buildOsFilterDropdown() {
        const wrapper = document.getElementById("host-os-select-wrap");
        if (!wrapper) return;
        const currentVal = document.getElementById("host-os-select")?.value || "";
        const osNames = [...new Set(hosts.map(h => h.config.os || h.detected_os || "").filter(Boolean))].sort();
        const options = ["", ...osNames];

        wrapper.innerHTML = `
            <div class="custom-select custom-select-sm" id="os-filter-cs">
                <button type="button" class="custom-select-trigger" id="os-filter-trigger">
                    ${currentVal ? osIcon(currentVal, 14) : ''}
                    <span>${currentVal ? esc(currentVal) : 'All OS'}</span>
                    <svg class="custom-select-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"></polyline></svg>
                </button>
                <div class="custom-select-menu" id="os-filter-menu">
                    ${options.map(o => {
                        const sel = o === currentVal ? " selected" : "";
                        return `<div class="custom-select-option${sel}" data-value="${ea(o)}">${o ? osIcon(o, 14) : ''}<span>${o ? esc(o) : 'All OS'}</span></div>`;
                    }).join("")}
                </div>
            </div>`;

        // Hidden input to hold the value
        let hidden = document.getElementById("host-os-select");
        if (!hidden) {
            hidden = document.createElement("input");
            hidden.type = "hidden";
            hidden.id = "host-os-select";
            wrapper.appendChild(hidden);
        }
        hidden.value = currentVal;

        const trigger = document.getElementById("os-filter-trigger");
        const menu = document.getElementById("os-filter-menu");

        trigger.addEventListener("click", (e) => {
            e.preventDefault();
            e.stopPropagation();
            menu.classList.toggle("open");
        });

        menu.addEventListener("click", (e) => {
            const opt = e.target.closest(".custom-select-option");
            if (!opt) return;
            const val = opt.dataset.value;
            hidden.value = val;
            trigger.innerHTML = `${val ? osIcon(val, 14) : ''}<span>${val ? esc(val) : 'All OS'}</span><svg class="custom-select-arrow" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"></polyline></svg>`;
            menu.querySelectorAll(".custom-select-option").forEach(o => o.classList.toggle("selected", o.dataset.value === val));
            menu.classList.remove("open");
            renderHosts();
        });

        document.addEventListener("click", (e) => {
            if (!wrapper.contains(e.target)) menu.classList.remove("open");
        });
    }

    // Also expose add host from hosts view if needed
    window.showAddHostModal = showAddHostModal;

    // ── Star toggle ──
    window.toggleStar = function (hostName, sessionName) {
        toggleSessionStar(hostName, sessionName);
        renderCurrentView();
    };

    // ── Session icon picker ──
    window.pickSessionIcon = function (btn, hostName, sessionName) {
        const host = hosts.find(h => h.config.name === hostName);
        const hostIcon = host ? (host.config.icon || "terminal") : "terminal";
        const hostColor = host ? (host.config.icon_color || "default") : "default";
        const currentIcon = getSessionIcon(hostName, sessionName) || hostIcon;
        const currentColor = getSessionIconColor(hostName, sessionName) || hostColor;
        showIconPicker(btn, currentIcon, function (iconName, colorName) {
            setSessionIcon(hostName, sessionName, iconName);
            setSessionIconColor(hostName, sessionName, colorName);
            renderCurrentView();
        }, currentColor);
    };

    // ── Search ──
    window.onSearch = function () {
        renderCurrentView();
        updateURL();
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

    function sessionStatusTip(attached) {
        return attached ? "Client attached to this session" : "Session running, no client attached";
    }

    function hostStatusTip(host) {
        if (!host.online) return "Offline — cannot reach host via SSH" + (host.error ? ": " + host.error : "");
        if (!host.tmux_detected) return "Online but tmux not found — install tmux to manage sessions";
        return "Online — tmux " + (host.tmux_version || "") + ", " + (host.sessions ? host.sessions.length : 0) + " session(s)";
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
    loadFromURL();
    loadSessionIcons(function() { fetchAll(); });
    fetchVersion();
    setInterval(fetchAll, 5000);

    // Handle browser back/forward
    window.addEventListener("hashchange", () => {
        loadFromURL();
        renderCurrentView();
    });
})();
