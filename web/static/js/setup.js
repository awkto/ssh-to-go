(function () {
    "use strict";

    // ── Password setup ──

    const passwordCard = document.getElementById("password-card");
    const keyCard = document.getElementById("key-card");

    if (passwordCard) {
        const passwordBtn = document.getElementById("set-password-btn");
        const passwordError = document.getElementById("password-error");

        passwordBtn.addEventListener("click", async () => {
            const password = document.getElementById("setup-password").value;
            const confirm = document.getElementById("setup-password-confirm").value;
            passwordError.classList.add("hidden");

            if (password.length < 4) {
                passwordError.textContent = "Password must be at least 4 characters";
                passwordError.classList.remove("hidden");
                return;
            }
            if (password !== confirm) {
                passwordError.textContent = "Passwords do not match";
                passwordError.classList.remove("hidden");
                return;
            }

            passwordBtn.disabled = true;
            passwordBtn.textContent = "Setting password...";

            try {
                const res = await fetch("/api/auth/setup", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ password }),
                });
                if (!res.ok) throw new Error(await res.text());

                // If keypairs already exist (existing instance), go to dashboard
                const kpRes = await fetch("/api/keypairs");
                const kps = await kpRes.json();
                if (kps && kps.length > 0) {
                    window.location.href = "/";
                    return;
                }

                // Fresh instance — show the key setup card
                passwordCard.style.display = "none";
                keyCard.style.display = "";
            } catch (e) {
                passwordError.textContent = "Error: " + e.message;
                passwordError.classList.remove("hidden");
            } finally {
                passwordBtn.disabled = false;
                passwordBtn.textContent = "Set Password & Continue";
            }
        });
    }

    // ── Tab switching ──

    document.querySelectorAll(".tab-btn").forEach(btn => {
        btn.addEventListener("click", () => {
            document.querySelectorAll(".tab-btn").forEach(b => b.classList.remove("active"));
            document.querySelectorAll(".tab-content").forEach(c => c.classList.remove("active"));
            btn.classList.add("active");
            document.getElementById("tab-" + btn.dataset.tab).classList.add("active");
        });
    });

    function showResult(pubKey) {
        document.getElementById("setup-pubkey").textContent = pubKey;
        document.getElementById("setup-result").classList.remove("hidden");
    }

    // Generate
    document.getElementById("gen-btn").addEventListener("click", async () => {
        const name = document.getElementById("gen-name").value.trim();
        if (!name) return alert("Enter a name");

        const btn = document.getElementById("gen-btn");
        btn.disabled = true;
        btn.textContent = "Generating...";

        try {
            const res = await fetch("/api/keypairs", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name }),
            });
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();

            // Set as default
            await fetch("/api/settings", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ default_keypair: name }),
            });

            showResult(data.public_key);
        } catch (e) {
            alert("Error: " + e.message);
        } finally {
            btn.disabled = false;
            btn.textContent = "Generate";
        }
    });

    // Import (paste)
    document.getElementById("import-paste-btn").addEventListener("click", async () => {
        const name = document.getElementById("import-name-paste").value.trim();
        const key = document.getElementById("import-key").value;
        if (!name || !key) return alert("Enter a name and paste your private key");

        try {
            const res = await fetch("/api/keypairs/import", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name, private_key: key }),
            });
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();

            await fetch("/api/settings", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ default_keypair: name }),
            });

            showResult(data.public_key);
        } catch (e) {
            alert("Error: " + e.message);
        }
    });

    // Import (server path)
    document.getElementById("import-path-btn").addEventListener("click", async () => {
        const name = document.getElementById("import-name-path").value.trim();
        const serverPath = document.getElementById("import-path").value.trim();
        if (!name || !serverPath) return alert("Enter a name and path");

        try {
            const res = await fetch("/api/keypairs/import", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name, server_path: serverPath }),
            });
            if (!res.ok) throw new Error(await res.text());
            const data = await res.json();

            await fetch("/api/settings", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ default_keypair: name }),
            });

            showResult(data.public_key);
        } catch (e) {
            alert("Error: " + e.message);
        }
    });

    // Copy
    document.getElementById("copy-setup-key").addEventListener("click", async () => {
        const key = document.getElementById("setup-pubkey").textContent;
        try {
            await navigator.clipboard.writeText(key);
            document.getElementById("copy-setup-key").textContent = "Copied!";
            setTimeout(() => {
                document.getElementById("copy-setup-key").textContent = "Copy Public Key";
            }, 2000);
        } catch (e) {
            alert("Copy failed");
        }
    });
})();
