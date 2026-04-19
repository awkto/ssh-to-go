package api

import (
	"html/template"
	"log"
	"net/http"
)

var (
	dashboardTmpl *template.Template
	terminalTmpl  *template.Template
	setupTmpl     *template.Template
	loginTmpl     *template.Template
)

func SetTemplates(dashboard, terminal, setup, login *template.Template) {
	dashboardTmpl = dashboard
	terminalTmpl = terminal
	setupTmpl = setup
	loginTmpl = login
}

func (h *Handlers) DashboardPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	// Redirect to setup if no keypairs exist
	if len(h.KeyStore.List()) == 0 {
		http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
		return
	}
	if err := dashboardTmpl.Execute(w, nil); err != nil {
		log.Printf("render dashboard: %v", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

type terminalData struct {
	Host           string
	Session        string
	TabTitleFormat string
}

func (h *Handlers) TerminalPage(w http.ResponseWriter, r *http.Request) {
	host := r.PathValue("host")
	session := r.PathValue("session")
	titleFmt := h.Settings.Get().TabTitleFormat
	if titleFmt == "" {
		titleFmt = "host-session"
	}
	data := terminalData{
		Host:           host,
		Session:        session,
		TabTitleFormat: titleFmt,
	}
	// Record last-accessed timestamp (best-effort).
	_ = h.SessionIcons.Touch(host, session)
	if err := terminalTmpl.Execute(w, data); err != nil {
		log.Printf("render terminal: %v", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func (h *Handlers) SetupPage(w http.ResponseWriter, r *http.Request) {
	// If already set up (has keypairs AND password), redirect to dashboard
	if len(h.KeyStore.List()) > 0 && (h.Auth.NoAuth() || h.Auth.HasPassword()) {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	type setupData struct {
		NeedsPassword bool
	}
	data := setupData{NeedsPassword: !h.Auth.NoAuth()}
	if err := setupTmpl.Execute(w, data); err != nil {
		log.Printf("render setup: %v", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func (h *Handlers) LoginPage(w http.ResponseWriter, r *http.Request) {
	// If already authenticated, redirect to dashboard
	if h.Auth.NoAuth() {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	if cookie, err := r.Cookie("session"); err == nil {
		if h.Auth.ValidSession(cookie.Value) {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
	}
	if err := loginTmpl.Execute(w, nil); err != nil {
		log.Printf("render login: %v", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
