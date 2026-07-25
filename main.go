package main

import (
	_ "embed"
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/awkto/ssh-to-go/internal/api"
	"github.com/awkto/ssh-to-go/internal/auth"
	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/execjob"
	"github.com/awkto/ssh-to-go/internal/hub"
	"github.com/awkto/ssh-to-go/internal/keystore"
	"github.com/awkto/ssh-to-go/internal/sessionreg"
	"github.com/awkto/ssh-to-go/internal/sshutil"
	"github.com/awkto/ssh-to-go/internal/tmux"
	"github.com/awkto/ssh-to-go/web"
)

//go:embed VERSION
var version string

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Initialize keystore
	ks, err := keystore.NewStore(cfg.DataDir)
	if err != nil {
		log.Fatalf("keystore: %v", err)
	}

	// Migrate legacy keypair if present
	if err := ks.Migrate(cfg.DataDir); err != nil {
		log.Printf("warning: keypair migration failed: %v", err)
	}

	// Initialize settings
	sm, err := keystore.NewSettingsManager(cfg.DataDir)
	if err != nil {
		log.Fatalf("settings: %v", err)
	}

	// Initialize session icon store
	sis, err := keystore.NewSessionIconStore(cfg.DataDir)
	if err != nil {
		log.Fatalf("session icons: %v", err)
	}

	// Remembers the commands sessions were launched with, so the New
	// Session form can offer them back as chips.
	rcs, err := keystore.NewRecentCommandStore(cfg.DataDir)
	if err != nil {
		log.Fatalf("recent commands: %v", err)
	}

	// Initialize tracked-session registry (records app-created sessions
	// so we can offer "recreate" after a host reboot).
	reg, err := sessionreg.NewStore(cfg.DataDir)
	if err != nil {
		log.Fatalf("session registry: %v", err)
	}

	// In-memory index for one-off exec jobs launched via /api/exec.
	execJobs := execjob.NewStore()

	// Initialize auth
	noAuth := os.Getenv("SSH_TO_GO_NO_AUTH") == "1"
	am, err := auth.NewManager(cfg.DataDir, noAuth)
	if err != nil {
		log.Fatalf("auth: %v", err)
	}
	if noAuth {
		log.Printf("authentication disabled (SSH_TO_GO_NO_AUTH=1)")
	}

	// Set global default key path (used as fallback)
	if len(ks.List()) > 0 {
		sshutil.DefaultKeyPath = ks.PrivateKeyPath(sm.DefaultKeypairName())
		pubKey, err := ks.PublicKey(sm.DefaultKeypairName())
		if err == nil {
			log.Printf("default keypair: %s", sm.DefaultKeypairName())
			log.Printf("public key: %s", pubKey)
		}
	} else {
		log.Printf("no keypairs configured — visit /setup to get started")
	}

	log.Printf("loaded %d hosts, listening on %s", len(cfg.Hosts), cfg.ListenAddr)

	// Parse templates
	dashboardTmpl, err := template.ParseFS(web.TemplateFS, "templates/index.html")
	if err != nil {
		log.Fatalf("parse dashboard template: %v", err)
	}
	terminalTmpl, err := template.ParseFS(web.TemplateFS, "templates/terminal.html")
	if err != nil {
		log.Fatalf("parse terminal template: %v", err)
	}
	setupTmpl, err := template.ParseFS(web.TemplateFS, "templates/setup.html")
	if err != nil {
		log.Fatalf("parse setup template: %v", err)
	}
	loginTmpl, err := template.ParseFS(web.TemplateFS, "templates/login.html")
	if err != nil {
		log.Fatalf("parse login template: %v", err)
	}
	api.SetTemplates(dashboardTmpl, terminalTmpl, setupTmpl, loginTmpl)

	// Set up hub and pollers
	h := hub.New(cfg.Hosts)
	tm := tmux.NewManager()

	pollResults := make(chan tmux.PollResult, max(len(cfg.Hosts)*2, 4))
	done := make(chan struct{})
	defer close(done)

	resolveKey := func(host config.Host) string {
		return keystore.ResolveKeyPath(host, ks, sm)
	}

	// Restore the incognito hide-list before the first poll lands, so a
	// hidden session never flashes into the dashboard after a restart.
	for _, host := range cfg.Hosts {
		if hidden := reg.HiddenNames(host.Name); len(hidden) > 0 {
			h.SetHidden(host.Name, hidden)
			log.Printf("hiding %d incognito session(s) on %s", len(hidden), host.Name)
		}
	}

	for _, host := range cfg.Hosts {
		log.Printf("starting poller for %s (%s@%s)", host.Name, host.User, host.DialAddress())
		tmux.StartPoller(host, cfg.PollInterval, resolveKey, reg, pollResults, done)
	}

	// Process poll results in background
	go func() {
		for result := range pollResults {
			h.Update(result)
			if result.Error != nil {
				log.Printf("poll %s: %v", result.HostName, result.Error)
			} else {
				log.Printf("poll %s: %d sessions", result.HostName, len(result.Sessions))
			}
		}
	}()

	// Static file server
	staticSub, err := fs.Sub(web.StaticFS, ".")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}

	router := api.NewRouter(api.RouterConfig{
		Hub:          h,
		Tmux:         tm,
		KeyStore:     ks,
		Settings:     sm,
		SessionIcons: sis,
		RecentCmds:   rcs,
		Registry:     reg,
		Auth:         am,
		ExecJobs:     execJobs,
		StaticFS:     http.FS(staticSub),
		ConfigPath:   *configPath,
		PollInterval: cfg.PollInterval,
		PollResults:  pollResults,
		Done:         done,
		Version:      strings.TrimSpace(version),
	})

	log.Printf("server starting at http://%s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, router); err != nil {
		log.Fatalf("server: %v", err)
	}
}
