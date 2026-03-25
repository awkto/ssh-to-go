package main

import (
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/awkto/ssh-to-go/internal/api"
	"github.com/awkto/ssh-to-go/internal/config"
	"github.com/awkto/ssh-to-go/internal/hub"
	"github.com/awkto/ssh-to-go/internal/sshutil"
	"github.com/awkto/ssh-to-go/internal/tmux"
	"github.com/awkto/ssh-to-go/web"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Generate or load SSH keypair
	keyPath, err := sshutil.EnsureKeypair(cfg.DataDir)
	if err != nil {
		log.Fatalf("ssh keypair: %v", err)
	}
	sshutil.DefaultKeyPath = keyPath
	log.Printf("SSH key: %s", keyPath)

	pubKey, err := sshutil.ReadPublicKey(cfg.DataDir)
	if err != nil {
		log.Fatalf("read public key: %v", err)
	}
	log.Printf("Public key: %s", pubKey)

	log.Printf("loaded %d hosts, listening on %s", len(cfg.Hosts), cfg.ListenAddr)

	// Parse templates from embedded FS
	dashboardTmpl, err := template.ParseFS(web.TemplateFS, "templates/index.html")
	if err != nil {
		log.Fatalf("parse dashboard template: %v", err)
	}
	terminalTmpl, err := template.ParseFS(web.TemplateFS, "templates/terminal.html")
	if err != nil {
		log.Fatalf("parse terminal template: %v", err)
	}
	api.SetTemplates(dashboardTmpl, terminalTmpl)

	// Set up hub and pollers
	h := hub.New(cfg.Hosts)
	tm := tmux.NewManager()

	pollResults := make(chan tmux.PollResult, max(len(cfg.Hosts)*2, 4))
	done := make(chan struct{})
	defer close(done)

	for _, host := range cfg.Hosts {
		log.Printf("starting poller for %s (%s@%s)", host.Name, host.User, host.Address)
		tmux.StartPoller(host, cfg.PollInterval, pollResults, done)
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

	// Static file server from embedded FS
	staticSub, err := fs.Sub(web.StaticFS, ".")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}

	router := api.NewRouter(api.RouterConfig{
		Hub:          h,
		Tmux:         tm,
		StaticFS:     http.FS(staticSub),
		ConfigPath:   *configPath,
		PollInterval: cfg.PollInterval,
		PollResults:  pollResults,
		Done:         done,
		DataDir:      cfg.DataDir,
	})

	log.Printf("server starting at http://%s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, router); err != nil {
		log.Fatalf("server: %v", err)
	}
}
