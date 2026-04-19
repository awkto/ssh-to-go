package keystore

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

type KeypairMeta struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
	Source      string    `json:"source"` // "generated" or "imported"
}

type keypairIndex struct {
	Keypairs []KeypairMeta `json:"keypairs"`
}

type Store struct {
	mu      sync.RWMutex
	baseDir string // data_dir/keys/
	index   keypairIndex
}

func NewStore(dataDir string) (*Store, error) {
	baseDir := filepath.Join(dataDir, "keys")
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("create keys dir: %w", err)
	}

	s := &Store{baseDir: baseDir}
	if err := s.loadIndex(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) indexPath() string {
	return filepath.Join(s.baseDir, "keypairs.json")
}

func (s *Store) privPath(name string) string {
	return filepath.Join(s.baseDir, name)
}

func (s *Store) pubPath(name string) string {
	return filepath.Join(s.baseDir, name+".pub")
}

func (s *Store) loadIndex() error {
	data, err := os.ReadFile(s.indexPath())
	if err != nil {
		if os.IsNotExist(err) {
			s.index = keypairIndex{}
			return nil
		}
		return fmt.Errorf("read keypair index: %w", err)
	}
	return json.Unmarshal(data, &s.index)
}

func (s *Store) saveIndex() error {
	data, err := json.MarshalIndent(s.index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.indexPath(), data, 0600)
}

// List returns all keypair metadata.
func (s *Store) List() []KeypairMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]KeypairMeta, len(s.index.Keypairs))
	copy(out, s.index.Keypairs)
	return out
}

// Get returns metadata for a single keypair.
func (s *Store) Get(name string) (*KeypairMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, kp := range s.index.Keypairs {
		if kp.Name == name {
			return &kp, nil
		}
	}
	return nil, fmt.Errorf("keypair %q not found", name)
}

// Exists checks if a keypair with the given name exists.
func (s *Store) Exists(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, kp := range s.index.Keypairs {
		if kp.Name == name {
			return true
		}
	}
	return false
}

// Generate creates a new ed25519 keypair.
func (s *Store) Generate(name string) (*KeypairMeta, error) {
	if !validName.MatchString(name) {
		return nil, fmt.Errorf("invalid keypair name %q (must match %s)", name, validName.String())
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, kp := range s.index.Keypairs {
		if kp.Name == name {
			return nil, fmt.Errorf("keypair %q already exists", name)
		}
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	if err := s.writeKeyFiles(name, priv, pub); err != nil {
		return nil, err
	}

	sshPub, _ := ssh.NewPublicKey(pub)
	meta := KeypairMeta{
		Name:        name,
		Type:        "ed25519",
		Fingerprint: ssh.FingerprintSHA256(sshPub),
		CreatedAt:   time.Now(),
		Source:      "generated",
	}
	s.index.Keypairs = append(s.index.Keypairs, meta)
	if err := s.saveIndex(); err != nil {
		return nil, err
	}
	return &meta, nil
}

// Import imports a keypair from PEM-encoded private key bytes.
func (s *Store) Import(name string, privateKeyPEM []byte) (*KeypairMeta, error) {
	if !validName.MatchString(name) {
		return nil, fmt.Errorf("invalid keypair name %q", name)
	}

	// Validate the key parses
	signer, err := ssh.ParsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, kp := range s.index.Keypairs {
		if kp.Name == name {
			return nil, fmt.Errorf("keypair %q already exists", name)
		}
	}

	// Write private key as-is
	if err := os.WriteFile(s.privPath(name), privateKeyPEM, 0600); err != nil {
		return nil, fmt.Errorf("write private key: %w", err)
	}

	// Derive and write public key
	pubKey := signer.PublicKey()
	authorizedKey := ssh.MarshalAuthorizedKey(pubKey)
	pubLine := fmt.Sprintf("%s ssh-to-go:%s\n", string(authorizedKey[:len(authorizedKey)-1]), name)
	if err := os.WriteFile(s.pubPath(name), []byte(pubLine), 0644); err != nil {
		return nil, fmt.Errorf("write public key: %w", err)
	}

	keyType := pubKey.Type()
	meta := KeypairMeta{
		Name:        name,
		Type:        keyType,
		Fingerprint: ssh.FingerprintSHA256(pubKey),
		CreatedAt:   time.Now(),
		Source:      "imported",
	}
	s.index.Keypairs = append(s.index.Keypairs, meta)
	if err := s.saveIndex(); err != nil {
		return nil, err
	}
	return &meta, nil
}

// ImportFromPath imports a keypair from a file path on the server.
func (s *Store) ImportFromPath(name, serverPath string) (*KeypairMeta, error) {
	data, err := os.ReadFile(serverPath)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}
	return s.Import(name, data)
}

// Delete removes a keypair from the store.
func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, kp := range s.index.Keypairs {
		if kp.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("keypair %q not found", name)
	}

	os.Remove(s.privPath(name))
	os.Remove(s.pubPath(name))

	s.index.Keypairs = append(s.index.Keypairs[:idx], s.index.Keypairs[idx+1:]...)
	return s.saveIndex()
}

// Rename renames a keypair, updating files and index.
func (s *Store) Rename(oldName, newName string) error {
	if !validName.MatchString(newName) {
		return fmt.Errorf("invalid keypair name %q (must match %s)", newName, validName.String())
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, kp := range s.index.Keypairs {
		if kp.Name == oldName {
			idx = i
		}
		if kp.Name == newName {
			return fmt.Errorf("keypair %q already exists", newName)
		}
	}
	if idx == -1 {
		return fmt.Errorf("keypair %q not found", oldName)
	}

	// Rename key files
	if err := os.Rename(s.privPath(oldName), s.privPath(newName)); err != nil {
		return fmt.Errorf("rename private key: %w", err)
	}
	if err := os.Rename(s.pubPath(oldName), s.pubPath(newName)); err != nil {
		// Try to roll back
		_ = os.Rename(s.privPath(newName), s.privPath(oldName))
		return fmt.Errorf("rename public key: %w", err)
	}

	s.index.Keypairs[idx].Name = newName
	return s.saveIndex()
}

// PublicKey returns the public key string for a keypair.
func (s *Store) PublicKey(name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(s.pubPath(name))
	if err != nil {
		return "", fmt.Errorf("read public key: %w", err)
	}
	return string(data), nil
}

// PrivateKeyPath returns the absolute path to a keypair's private key file.
func (s *Store) PrivateKeyPath(name string) string {
	return s.privPath(name)
}

// Migrate moves a legacy id_ed25519 keypair from dataDir into the store.
func (s *Store) Migrate(dataDir string) error {
	legacyPriv := filepath.Join(dataDir, "id_ed25519")
	legacyPub := filepath.Join(dataDir, "id_ed25519.pub")

	if _, err := os.Stat(legacyPriv); err != nil {
		return nil // nothing to migrate
	}

	// Already migrated?
	if s.Exists("server") {
		return nil
	}

	// Move files
	if err := os.Rename(legacyPriv, s.privPath("server")); err != nil {
		return fmt.Errorf("migrate private key: %w", err)
	}
	if err := os.Rename(legacyPub, s.pubPath("server")); err != nil {
		// Non-fatal, we can derive it
		_ = err
	}

	// Read the key to get metadata
	data, err := os.ReadFile(s.privPath("server"))
	if err != nil {
		return fmt.Errorf("read migrated key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return fmt.Errorf("parse migrated key: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	meta := KeypairMeta{
		Name:        "server",
		Type:        signer.PublicKey().Type(),
		Fingerprint: ssh.FingerprintSHA256(signer.PublicKey()),
		CreatedAt:   time.Now(),
		Source:      "generated",
	}
	s.index.Keypairs = append(s.index.Keypairs, meta)
	return s.saveIndex()
}

// EnsureDefault generates a "server" keypair if no keypairs exist.
func (s *Store) EnsureDefault() error {
	if len(s.List()) > 0 {
		return nil
	}
	_, err := s.Generate("server")
	return err
}

func (s *Store) writeKeyFiles(name string, priv ed25519.PrivateKey, pub ed25519.PublicKey) error {
	privBytes, err := ssh.MarshalPrivateKey(priv, "ssh-to-go:"+name)
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}

	if err := os.WriteFile(s.privPath(name), pem.EncodeToMemory(privBytes), 0600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	sshPub, _ := ssh.NewPublicKey(pub)
	authorizedKey := ssh.MarshalAuthorizedKey(sshPub)
	pubLine := fmt.Sprintf("%s ssh-to-go:%s\n", string(authorizedKey[:len(authorizedKey)-1]), name)
	if err := os.WriteFile(s.pubPath(name), []byte(pubLine), 0644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}

	return nil
}
