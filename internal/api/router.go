package api

import (
	"net/http"
	"time"

	"github.com/awkto/ssh-to-go/internal/auth"
	"github.com/awkto/ssh-to-go/internal/hub"
	"github.com/awkto/ssh-to-go/internal/keystore"
	"github.com/awkto/ssh-to-go/internal/tmux"
)

type RouterConfig struct {
	Hub          *hub.Hub
	Tmux         *tmux.Manager
	KeyStore     *keystore.Store
	Settings     *keystore.SettingsManager
	Auth         *auth.Manager
	StaticFS     http.FileSystem
	ConfigPath   string
	PollInterval time.Duration
	PollResults  chan<- tmux.PollResult
	Done         <-chan struct{}
}

func NewRouter(rc RouterConfig) http.Handler {
	handlers := &Handlers{
		Hub:          rc.Hub,
		Tmux:         rc.Tmux,
		KeyStore:     rc.KeyStore,
		Settings:     rc.Settings,
		Auth:         rc.Auth,
		ConfigPath:   rc.ConfigPath,
		PollInterval: rc.PollInterval,
		PollResults:  rc.PollResults,
		Done:         rc.Done,
	}
	mux := http.NewServeMux()

	// Auth API (some routes are public, gated by middleware)
	mux.HandleFunc("POST /api/auth/setup", handlers.AuthSetup)
	mux.HandleFunc("POST /api/auth/login", handlers.AuthLogin)
	mux.HandleFunc("POST /api/auth/logout", handlers.AuthLogout)
	mux.HandleFunc("PUT /api/auth/password", handlers.AuthChangePassword)
	mux.HandleFunc("GET /api/auth/tokens", handlers.AuthListTokens)
	mux.HandleFunc("POST /api/auth/tokens", handlers.AuthCreateToken)
	mux.HandleFunc("DELETE /api/auth/tokens/{name}", handlers.AuthDeleteToken)

	// Session/host API
	mux.HandleFunc("GET /api/sessions", handlers.ListSessions)
	mux.HandleFunc("GET /api/hosts", handlers.ListHosts)
	mux.HandleFunc("POST /api/hosts", handlers.AddHost)
	mux.HandleFunc("POST /api/hosts/{host}/sessions", handlers.CreateSession)
	mux.HandleFunc("POST /api/hosts/{host}/scan", handlers.ScanHost)
	mux.HandleFunc("DELETE /api/hosts/{host}/sessions/{session}", handlers.KillSession)
	mux.HandleFunc("PUT /api/hosts/{host}/sessions/{session}", handlers.RenameSession)
	mux.HandleFunc("GET /api/hosts/{host}/sessions/{session}/handoff", handlers.Handoff)
	mux.HandleFunc("POST /api/scan", handlers.ScanAll)
	mux.HandleFunc("GET /api/pubkey", handlers.PubKey)

	// Keypair API
	mux.HandleFunc("GET /api/keypairs", handlers.ListKeypairs)
	mux.HandleFunc("POST /api/keypairs", handlers.CreateKeypair)
	mux.HandleFunc("POST /api/keypairs/import", handlers.ImportKeypair)
	mux.HandleFunc("GET /api/keypairs/{name}", handlers.GetKeypair)
	mux.HandleFunc("DELETE /api/keypairs/{name}", handlers.DeleteKeypair)

	// Settings API
	mux.HandleFunc("GET /api/settings", handlers.GetSettings)
	mux.HandleFunc("PUT /api/settings", handlers.UpdateSettings)

	// WebSocket
	mux.HandleFunc("GET /ws/{host}/{session}", handlers.WebSocket)

	// Static files and pages
	mux.Handle("GET /static/", http.FileServer(rc.StaticFS))
	mux.HandleFunc("GET /login", handlers.LoginPage)
	mux.HandleFunc("GET /terminal/{host}/{session}", handlers.TerminalPage)
	mux.HandleFunc("GET /settings", handlers.SettingsPage)
	mux.HandleFunc("GET /setup", handlers.SetupPage)
	mux.HandleFunc("GET /", handlers.DashboardPage)

	return auth.Middleware(rc.Auth, mux)
}
