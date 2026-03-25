(function () {
    "use strict";

    const keypairList = document.getElementById("keypair-list");
    const modal = document.getElementById("modal-overlay");
    const modalTitle = document.getElementById("modal-title");
    const modalFields = document.getElementById("modal-fields");
    const modalForm = document.getElementById("modal-form");
    const modalSubmit = document.getElementById("modal-submit");
    const authEnabled = window.__authEnabled;

    let keypairs = [];
    let settings = {};
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

    async function fetchAll() {
        const [kRes, sRes] = await Promise.all([
            authFetch("/api/keypairs"),
            authFetch("/api/settings"),
        ]);
        keypairs = await kRes.json();
        settings = await sRes.json();
        render();

        if (authEnabled) {
            fetchTokens();
        }
    }

    // ── Render ──

    function render() {
        document.getElementById("default-username").value = settings.default_username || "";

        const sel = document.getElementById("default-keypair");
        sel.innerHTML = keypairs.map(kp =>
            `<option value="${ea(kp.name)}" ${kp.name === settings.default_keypair ? "selected" : ""}>${esc(kp.name)}</option>`
        ).join("");

        document.getElementById("tmux-window-size").value = settings.tmux_window_size || "largest";

        if (keypairs.length === 0) {
            keypairList.innerHTML = `<div class="no-sessions">No keypairs. Generate or import one.</div>`;
            return;
        }

        keypairList.innerHTML = keypairs.map(kp => {
            const isDefault = kp.name === settings.default_keypair;
            return `<div class="keypair-card">
                <div class="keypair-header">
                    <div class="keypair-title">
                        <strong>${esc(kp.name)}</strong>
                        ${isDefault ? '<span class="badge badge-attached">default</span>' : ""}
                        <span class="host-meta-text">${esc(kp.type)} &middot; ${esc(kp.source)} &middot; ${esc(kp.fingerprint)}</span>
                    </div>
                    <div class="action-group">
                        <button class="btn btn-sm" onclick="viewPubKey('${ea(kp.name)}')">Public Key</button>
                        ${!isDefault ? `<button class="btn btn-sm btn-danger" onclick="deleteKeypair('${ea(kp.name)}')">Delete</button>` : ""}
                    </div>
                </div>
                <div id="pubkey-${ea(kp.name)}" class="keypair-pubkey hidden"></div>
            </div>`;
        }).join("");
    }

    // ── Actions ──

    window.viewPubKey = async function (name) {
        const el = document.getElementById("pubkey-" + name);
        if (!el.classList.contains("hidden")) {
            el.classList.add("hidden");
            return;
        }
        try {
            const res = await authFetch(`/api/keypairs/${encodeURIComponent(name)}`);
            const data = await res.json();
            el.innerHTML = `<code class="pubkey-code">${esc(data.public_key)}</code>
                <button class="btn btn-sm" onclick="copyText('${ea(data.public_key)}')">Copy</button>`;
            el.classList.remove("hidden");
        } catch (e) {
            alert("Failed to load public key");
        }
    };

    window.copyText = async function (text) {
        try {
            await navigator.clipboard.writeText(text);
            toast("Copied", "success");
        } catch (e) {
            toast("Copy failed", "error");
        }
    };

    window.deleteKeypair = async function (name) {
        if (!confirm(`Delete keypair "${name}"?`)) return;
        try {
            const res = await authFetch(`/api/keypairs/${encodeURIComponent(name)}`, { method: "DELETE" });
            if (!res.ok) throw new Error(await res.text());
            toast("Deleted", "success");
            fetchAll();
        } catch (e) {
            toast("Error: " + e.message, "error");
        }
    };

    // ── Save settings ──

    document.getElementById("save-settings-btn").addEventListener("click", async () => {
        const defaultUsername = document.getElementById("default-username").value.trim();
        const defaultKeypair = document.getElementById("default-keypair").value;
        const tmuxWindowSize = document.getElementById("tmux-window-size").value;

        try {
            const res = await authFetch("/api/settings", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    default_username: defaultUsername,
                    default_keypair: defaultKeypair,
                    tmux_window_size: tmuxWindowSize,
                }),
            });
            if (!res.ok) throw new Error(await res.text());
            toast("Settings saved", "success");
            fetchAll();
        } catch (e) {
            toast("Error: " + e.message, "error");
        }
    });

    // ── Generate ──

    document.getElementById("generate-keypair-btn").addEventListener("click", () => {
        modalTitle.textContent = "Generate Keypair";
        modalSubmit.textContent = "Generate";
        modalFields.innerHTML = `
            <label for="m-name">Name</label>
            <input type="text" id="m-name" placeholder="my-key" required>
        `;
        modalHandler = async () => {
            const name = document.getElementById("m-name").value.trim();
            if (!name) return;
            const res = await authFetch("/api/keypairs", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name }),
            });
            if (!res.ok) throw new Error(await res.text());
            toast(`Keypair "${name}" generated`, "success");
            fetchAll();
        };
        modal.classList.remove("hidden");
    });

    // ── Import ──

    document.getElementById("import-keypair-btn").addEventListener("click", () => {
        modalTitle.textContent = "Import Keypair";
        modalSubmit.textContent = "Import";
        modalFields.innerHTML = `
            <label for="m-name">Name</label>
            <input type="text" id="m-name" placeholder="my-key" required>
            <label for="m-method">Import Method</label>
            <select id="m-method">
                <option value="paste">Paste Private Key</option>
                <option value="path">Server File Path</option>
            </select>
            <div id="m-paste-fields">
                <label for="m-key">Private Key (PEM)</label>
                <textarea id="m-key" rows="6" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"></textarea>
            </div>
            <div id="m-path-fields" class="hidden">
                <label for="m-path">Path on this server</label>
                <input type="text" id="m-path" placeholder="/home/user/.ssh/id_ed25519">
            </div>
        `;
        // Toggle fields
        setTimeout(() => {
            document.getElementById("m-method").addEventListener("change", (e) => {
                document.getElementById("m-paste-fields").classList.toggle("hidden", e.target.value !== "paste");
                document.getElementById("m-path-fields").classList.toggle("hidden", e.target.value !== "path");
            });
        }, 0);

        modalHandler = async () => {
            const name = document.getElementById("m-name").value.trim();
            const method = document.getElementById("m-method").value;
            if (!name) return;

            const body = { name };
            if (method === "paste") {
                body.private_key = document.getElementById("m-key").value;
                if (!body.private_key) throw new Error("Paste your private key");
            } else {
                body.server_path = document.getElementById("m-path").value.trim();
                if (!body.server_path) throw new Error("Enter the key path");
            }

            const res = await authFetch("/api/keypairs/import", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(body),
            });
            if (!res.ok) throw new Error(await res.text());
            toast(`Keypair "${name}" imported`, "success");
            fetchAll();
        };
        modal.classList.remove("hidden");
    });

    // ── Modal plumbing ──

    document.getElementById("modal-cancel").addEventListener("click", () => modal.classList.add("hidden"));
    modal.addEventListener("click", (e) => { if (e.target === modal) modal.classList.add("hidden"); });
    modalForm.addEventListener("submit", async (e) => {
        e.preventDefault();
        if (!modalHandler) return;
        try {
            await modalHandler();
            modal.classList.add("hidden");
        } catch (err) {
            toast("Error: " + err.message, "error");
        }
    });

    // ── Password change ──

    if (authEnabled) {
        document.getElementById("change-password-btn").addEventListener("click", async () => {
            const currentPassword = document.getElementById("current-password").value;
            const newPassword = document.getElementById("new-password").value;
            const confirmPassword = document.getElementById("confirm-password").value;

            if (newPassword.length < 4) {
                toast("Password must be at least 4 characters", "error");
                return;
            }
            if (newPassword !== confirmPassword) {
                toast("Passwords do not match", "error");
                return;
            }

            try {
                const res = await authFetch("/api/auth/password", {
                    method: "PUT",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({
                        current_password: currentPassword,
                        new_password: newPassword,
                    }),
                });
                if (!res.ok) throw new Error(await res.text());
                toast("Password changed", "success");
                document.getElementById("current-password").value = "";
                document.getElementById("new-password").value = "";
                document.getElementById("confirm-password").value = "";
            } catch (e) {
                toast("Error: " + e.message, "error");
            }
        });

        // ── Logout ──

        document.getElementById("logout-btn").addEventListener("click", async () => {
            try {
                await fetch("/api/auth/logout", { method: "POST" });
            } catch (e) { /* ignore */ }
            window.location.href = "/login";
        });

        // ── API Tokens ──

        const tokenList = document.getElementById("token-list");

        async function fetchTokens() {
            try {
                const res = await authFetch("/api/auth/tokens");
                const tokens = await res.json();
                renderTokens(tokens);
            } catch (e) { /* ignore */ }
        }

        function renderTokens(tokens) {
            if (!tokens || tokens.length === 0) {
                tokenList.innerHTML = `<div class="no-sessions">No API tokens created yet.</div>`;
                return;
            }
            tokenList.innerHTML = tokens.map(t => {
                const created = new Date(t.created).toLocaleDateString();
                return `<div class="keypair-card">
                    <div class="keypair-header">
                        <div class="keypair-title">
                            <strong>${esc(t.name)}</strong>
                            <span class="host-meta-text">created ${esc(created)}</span>
                        </div>
                        <div class="action-group">
                            <button class="btn btn-sm btn-danger" onclick="deleteToken('${ea(t.name)}')">Revoke</button>
                        </div>
                    </div>
                </div>`;
            }).join("");
        }

        window.deleteToken = async function (name) {
            if (!confirm(`Revoke API token "${name}"?`)) return;
            try {
                const res = await authFetch(`/api/auth/tokens/${encodeURIComponent(name)}`, { method: "DELETE" });
                if (!res.ok) throw new Error(await res.text());
                toast("Token revoked", "success");
                fetchTokens();
            } catch (e) {
                toast("Error: " + e.message, "error");
            }
        };

        document.getElementById("create-token-btn").addEventListener("click", () => {
            modalTitle.textContent = "Create API Token";
            modalSubmit.textContent = "Create";
            modalSubmit.style.display = "";
            modalFields.innerHTML = `
                <label for="m-name">Token Name</label>
                <input type="text" id="m-name" placeholder="my-script" required>
            `;
            modalHandler = async () => {
                const name = document.getElementById("m-name").value.trim();
                if (!name) return;
                const res = await authFetch("/api/auth/tokens", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ name }),
                });
                if (!res.ok) throw new Error(await res.text());
                const data = await res.json();

                // Show the token once
                modalTitle.textContent = "Token Created";
                modalFields.innerHTML = `
                    <p style="color:#888;font-size:13px;margin-bottom:12px">Copy this token now — it won't be shown again.</p>
                    <code class="pubkey-code" id="new-token-value">${esc(data.token)}</code>
                    <button type="button" class="btn btn-sm" style="margin-top:8px" onclick="copyText('${ea(data.token)}')">Copy Token</button>
                `;
                modalSubmit.textContent = "Done";
                modalHandler = async () => {
                    fetchTokens();
                };
            };
            modal.classList.remove("hidden");
        });
    }

    // ── Helpers ──

    function esc(s) {
        const d = document.createElement("div");
        d.textContent = s || "";
        return d.innerHTML;
    }
    function ea(s) { return (s || "").replace(/'/g, "\\'").replace(/"/g, "&quot;"); }
    function toast(msg, type) {
        const el = document.createElement("div");
        el.className = `toast toast-${type}`;
        el.textContent = msg;
        document.body.appendChild(el);
        setTimeout(() => el.remove(), 3000);
    }

    // ── Init ──
    fetchAll();
})();
