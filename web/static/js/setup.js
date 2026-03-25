(function () {
    "use strict";

    // Tab switching
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
