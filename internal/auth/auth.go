package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// APIToken is a named API token (hash stored, plain token shown once at creation).
type APIToken struct {
	Name      string    `json:"name"`
	TokenHash string    `json:"token_hash"`
	Created   time.Time `json:"created"`
}

// storedState is the persistent auth state saved to disk.
type storedState struct {
	PasswordHash string     `json:"password_hash"`
	APITokens    []APIToken `json:"api_tokens"`
}

// session is an in-memory browser session.
type session struct {
	expiresAt time.Time
}

const (
	sessionTTL  = 7 * 24 * time.Hour
	bcryptCost  = 12
	tokenBytes  = 32
	sessionFile = "auth.json"
)

// Manager handles authentication state.
type Manager struct {
	mu       sync.RWMutex
	path     string
	state    storedState
	sessions map[string]session // token -> session
	noAuth   bool
}

// NewManager creates an auth manager that persists to dataDir/auth.json.
// If noAuth is true, authentication is disabled entirely.
func NewManager(dataDir string, noAuth bool) (*Manager, error) {
	m := &Manager{
		path:     filepath.Join(dataDir, sessionFile),
		sessions: make(map[string]session),
		noAuth:   noAuth,
	}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

// NoAuth returns true if authentication is disabled.
func (m *Manager) NoAuth() bool {
	return m.noAuth
}

// HasPassword returns true if an admin password has been set.
func (m *Manager) HasPassword() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.PasswordHash != ""
}

// SetPassword sets the admin password (first-run or change).
// If currentPassword is non-empty, it must match the existing password.
func (m *Manager) SetPassword(currentPassword, newPassword string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If a password is already set, verify current password
	if m.state.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(m.state.PasswordHash), []byte(currentPassword)); err != nil {
			return fmt.Errorf("current password is incorrect")
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	m.state.PasswordHash = string(hash)
	return m.save()
}

// CheckPassword verifies the given password against the stored hash.
func (m *Manager) CheckPassword(password string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state.PasswordHash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(m.state.PasswordHash), []byte(password)) == nil
}

// CreateSession creates a new browser session and returns the session token.
func (m *Manager) CreateSession() (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[token] = session{expiresAt: time.Now().Add(sessionTTL)}
	return token, nil
}

// ValidSession checks if a session token is valid and not expired.
func (m *Manager) ValidSession(token string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(s.expiresAt) {
		return false
	}
	return true
}

// DeleteSession removes a browser session.
func (m *Manager) DeleteSession(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
}

// CreateAPIToken creates a new API token and returns the plain token (shown once).
func (m *Manager) CreateAPIToken(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("token name is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate name
	for _, t := range m.state.APITokens {
		if t.Name == name {
			return "", fmt.Errorf("token %q already exists", name)
		}
	}

	plain, err := randomToken()
	if err != nil {
		return "", err
	}

	m.state.APITokens = append(m.state.APITokens, APIToken{
		Name:      name,
		TokenHash: hashToken(plain),
		Created:   time.Now(),
	})

	if err := m.save(); err != nil {
		return "", err
	}
	return plain, nil
}

// ValidAPIToken checks if the given bearer token matches any stored API token.
func (m *Manager) ValidAPIToken(plain string) bool {
	h := hashToken(plain)
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.state.APITokens {
		if t.TokenHash == h {
			return true
		}
	}
	return false
}

// ListAPITokens returns API token metadata (no hashes).
func (m *Manager) ListAPITokens() []APIToken {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]APIToken, len(m.state.APITokens))
	for i, t := range m.state.APITokens {
		out[i] = APIToken{Name: t.Name, Created: t.Created}
	}
	return out
}

// DeleteAPIToken removes an API token by name.
func (m *Manager) DeleteAPIToken(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, t := range m.state.APITokens {
		if t.Name == name {
			m.state.APITokens = append(m.state.APITokens[:i], m.state.APITokens[i+1:]...)
			return m.save()
		}
	}
	return fmt.Errorf("token %q not found", name)
}

// load reads auth state from disk.
func (m *Manager) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read auth: %w", err)
	}
	return json.Unmarshal(data, &m.state)
}

// save writes auth state to disk.
func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0600)
}

func randomToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func hashToken(plain string) string {
	h := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(h[:])
}
