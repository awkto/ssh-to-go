package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

func cmdAuth(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: stogo auth login|logout")
	}
	switch args[0] {
	case "login":
		return authLogin(args[1:])
	case "logout":
		return authLogout()
	default:
		return fmt.Errorf("unknown auth subcommand %q (want login or logout)", args[0])
	}
}

func authLogin(args []string) error {
	fs := flag.NewFlagSet("auth login", flag.ContinueOnError)
	urlFlag := fs.String("url", "", "server URL (e.g. https://sshtogo.example.com)")
	nameFlag := fs.String("name", "", "API token name to create (default stogo-<hostname>)")
	tokenFlag := fs.String("token", "", "use an existing API token instead of password login")
	if err := fs.Parse(args); err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)

	serverURL := *urlFlag
	if serverURL == "" {
		fmt.Print("Server URL: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		serverURL = strings.TrimSpace(line)
	}
	if serverURL == "" {
		return fmt.Errorf("server URL is required")
	}
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		serverURL = "https://" + serverURL
	}
	serverURL = strings.TrimRight(serverURL, "/")

	// An existing token just needs verifying.
	if *tokenFlag != "" {
		cfg := &cliConfig{URL: serverURL, Token: *tokenFlag}
		if err := verifyAndSave(cfg); err != nil {
			return err
		}
		fmt.Printf("Logged in to %s (existing token)\n", serverURL)
		return nil
	}

	// A server running with auth disabled needs no token at all.
	probe := newClient(&cliConfig{URL: serverURL})
	var me struct {
		Authenticated bool `json:"authenticated"`
		NoAuth        bool `json:"no_auth"`
	}
	if err := probe.do("GET", "/api/me", nil, &me); err == nil && me.NoAuth {
		if err := saveConfig(&cliConfig{URL: serverURL, NoAuth: true}); err != nil {
			return err
		}
		fmt.Printf("Logged in to %s (server has authentication disabled)\n", serverURL)
		return nil
	}

	fmt.Print("Password: ")
	var pw []byte
	var err error
	if term.IsTerminal(int(os.Stdin.Fd())) {
		pw, err = term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
	} else {
		// Piped stdin (scripts, tests): read one line instead.
		line, rerr := reader.ReadString('\n')
		pw, err = []byte(strings.TrimRight(line, "\r\n")), rerr
	}
	if err != nil {
		return err
	}

	// Password login gives a browser session (in-memory on the server); use
	// it once to mint a durable named API token, then discard the session.
	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Timeout: 30 * time.Second, Jar: jar}

	loginBody, _ := json.Marshal(map[string]string{"password": string(pw)})
	resp, err := hc.Post(serverURL+"/api/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid password")
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp.StatusCode, body)
	}

	tokenName := *nameFlag
	if tokenName == "" {
		host, _ := os.Hostname()
		if host == "" {
			host = "cli"
		}
		tokenName = "stogo-" + host
	}

	token, err := createToken(hc, serverURL, tokenName)
	if err != nil && strings.Contains(err.Error(), "already exists") && *nameFlag == "" {
		// Auto-generated name collided (e.g. re-login from the same machine);
		// retry with a timestamp suffix rather than failing.
		tokenName = fmt.Sprintf("%s-%s", tokenName, time.Now().Format("20060102-150405"))
		token, err = createToken(hc, serverURL, tokenName)
	}
	if err != nil {
		return err
	}

	// Best-effort: drop the temporary browser session.
	if req, rerr := http.NewRequest("POST", serverURL+"/api/auth/logout", nil); rerr == nil {
		if r, derr := hc.Do(req); derr == nil {
			r.Body.Close()
		}
	}

	cfg := &cliConfig{URL: serverURL, Token: token, TokenName: tokenName}
	if err := verifyAndSave(cfg); err != nil {
		return err
	}
	fmt.Printf("Logged in to %s (created API token %q)\n", serverURL, tokenName)
	return nil
}

func createToken(hc *http.Client, serverURL, name string) (string, error) {
	body, _ := json.Marshal(map[string]string{"name": name})
	resp, err := hc.Post(serverURL+"/api/auth/tokens", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated {
		return "", apiError(resp.StatusCode, data)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &out); err != nil || out.Token == "" {
		return "", fmt.Errorf("server did not return a token")
	}
	return out.Token, nil
}

func verifyAndSave(cfg *cliConfig) error {
	c := newClient(cfg)
	var me struct {
		Authenticated bool   `json:"authenticated"`
		Version       string `json:"version"`
	}
	if err := c.do("GET", "/api/me", nil, &me); err != nil {
		return fmt.Errorf("verify login: %w", err)
	}
	return saveConfig(cfg)
}

func authLogout() error {
	cfg, err := loadConfig()
	if err != nil {
		// Nothing configured — nothing to do.
		return removeConfig()
	}

	// Revoke the token server-side when we know its name; a token pasted via
	// -token or env has no recorded name, so it can only be forgotten locally.
	if cfg.TokenName != "" && cfg.Token != "" {
		c := newClient(cfg)
		if err := c.do("DELETE", "/api/auth/tokens/"+cfg.TokenName, nil, nil); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not revoke token %q on server: %v\n", cfg.TokenName, err)
		} else {
			fmt.Printf("Revoked API token %q on %s\n", cfg.TokenName, cfg.URL)
		}
	}
	if err := removeConfig(); err != nil {
		return err
	}
	fmt.Println("Logged out")
	return nil
}
