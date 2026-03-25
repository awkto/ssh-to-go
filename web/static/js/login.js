(function () {
    "use strict";

    const form = document.getElementById("login-form");
    const passwordInput = document.getElementById("password");
    const errorEl = document.getElementById("login-error");

    form.addEventListener("submit", async (e) => {
        e.preventDefault();
        errorEl.classList.add("hidden");

        const password = passwordInput.value;
        if (!password) return;

        try {
            const res = await fetch("/api/auth/login", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ password }),
            });

            if (!res.ok) {
                const data = await res.json().catch(() => ({}));
                throw new Error(data.error || "Login failed");
            }

            // Redirect to dashboard (or wherever they were going)
            const params = new URLSearchParams(window.location.search);
            window.location.href = params.get("next") || "/";
        } catch (err) {
            errorEl.textContent = err.message;
            errorEl.classList.remove("hidden");
            passwordInput.select();
        }
    });
})();
