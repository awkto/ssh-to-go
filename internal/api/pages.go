package api

import (
	"html/template"
	"log"
	"net/http"
)

var (
	dashboardTmpl *template.Template
	terminalTmpl  *template.Template
)

func SetTemplates(dashboard, terminal *template.Template) {
	dashboardTmpl = dashboard
	terminalTmpl = terminal
}

func (h *Handlers) DashboardPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if err := dashboardTmpl.Execute(w, nil); err != nil {
		log.Printf("render dashboard: %v", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

type terminalData struct {
	Host    string
	Session string
}

func (h *Handlers) TerminalPage(w http.ResponseWriter, r *http.Request) {
	data := terminalData{
		Host:    r.PathValue("host"),
		Session: r.PathValue("session"),
	}
	if err := terminalTmpl.Execute(w, data); err != nil {
		log.Printf("render terminal: %v", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
