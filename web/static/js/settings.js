(function () {
    "use strict";

    const keypairList = document.getElementById("keypair-list");
    const modal = document.getElementById("modal-overlay");
    const modalTitle = document.getElementById("modal-title");
    const modalFields = document.getElementById("modal-fields");
    const modalForm = document.getElementById("modal-form");
    const modalSubmit = document.getElementById("modal-submit");

    let keypairs = [];
    let settings = {};
    let modalHandler = null;

    // ── Fetch ──

    async function fetchAll() {
        const [kRes, sRes] = await Promise.all([
            fetch("/api/keypairs"),
            fetch("/api/settings"),
        ]);
        keypairs = await kRes.json();
        settings = await sRes.json();
        render();
    }

    // ── Render ──

    function render() {
        document.getElementById("default-username").value = settings.default_username || "";

        const sel = document.getElementById("default-keypair");
        sel.innerHTML = keypairs.map(kp =>
            `<option value="${ea(kp.name)}" ${kp.name === settings.default_keypair ? "selected" : ""}>${esc(kp.name)}</option>`
        ).join("");

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
            const res = await fetch(`/api/keypairs/${encodeURIComponent(name)}`);
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
            const res = await fetch(`/api/keypairs/${encodeURIComponent(name)}`, { method: "DELETE" });
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

        try {
            const res = await fetch("/api/settings", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    default_username: defaultUsername,
                    default_keypair: defaultKeypair,
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
            const res = await fetch("/api/keypairs", {
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

            const res = await fetch("/api/keypairs/import", {
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
