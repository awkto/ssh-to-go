package api

import (
	"net/http"
	"time"

	"github.com/awkto/ssh-to-go/internal/hub"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

// RouterConfig holds the extra state needed to construct handlers.
type RouterConfig struct {
	Hub          *hub.Hub
	Tmux         *tmux.Manager
	StaticFS     http.FileSystem
	ConfigPath   string
	PollInterval time.Duration
	PollResults  chan<- tmux.PollResult
	Done         <-chan struct{}
}

func NewRouter(rc RouterConfig) *http.ServeMux {
	handlers := &Handlers{
		Hub:          rc.Hub,
		Tmux:         rc.Tmux,
		ConfigPath:   rc.ConfigPath,
		PollInterval: rc.PollInterval,
		PollResults:  rc.PollResults,
		Done:         rc.Done,
	}
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/sessions", handlers.ListSessions)
	mux.HandleFunc("GET /api/hosts", handlers.ListHosts)
	mux.HandleFunc("POST /api/hosts", handlers.AddHost)
	mux.HandleFunc("POST /api/hosts/{host}/sessions", handlers.CreateSession)
	mux.HandleFunc("POST /api/hosts/{host}/scan", handlers.ScanHost)
	mux.HandleFunc("DELETE /api/hosts/{host}/sessions/{session}", handlers.KillSession)
	mux.HandleFunc("GET /api/hosts/{host}/sessions/{session}/handoff", handlers.Handoff)
	mux.HandleFunc("POST /api/scan", handlers.ScanAll)

	// WebSocket
	mux.HandleFunc("GET /ws/{host}/{session}", handlers.WebSocket)

	// Static files and pages
	mux.Handle("GET /static/", http.FileServer(rc.StaticFS))
	mux.HandleFunc("GET /terminal/{host}/{session}", handlers.TerminalPage)
	mux.HandleFunc("GET /", handlers.DashboardPage)

	return mux
}
