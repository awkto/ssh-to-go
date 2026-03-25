package sshutil

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// EnsureKeypair generates an ed25519 keypair in dataDir if one doesn't
// already exist. Returns the path to the private key.
func EnsureKeypair(dataDir string) (string, error) {
	privPath := filepath.Join(dataDir, "id_ed25519")
	pubPath := privPath + ".pub"

	// Already exists
	if _, err := os.Stat(privPath); err == nil {
		return privPath, nil
	}

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}

	privBytes, err := ssh.MarshalPrivateKey(priv, "ssh-to-go server key")
	if err != nil {
		return "", fmt.Errorf("marshal private key: %w", err)
	}

	if err := os.WriteFile(privPath, pem.EncodeToMemory(privBytes), 0600); err != nil {
		return "", fmt.Errorf("write private key: %w", err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("convert public key: %w", err)
	}

	authorizedKey := ssh.MarshalAuthorizedKey(sshPub)
	// Add a comment
	pubLine := fmt.Sprintf("%s ssh-to-go\n", string(authorizedKey[:len(authorizedKey)-1]))
	if err := os.WriteFile(pubPath, []byte(pubLine), 0644); err != nil {
		return "", fmt.Errorf("write public key: %w", err)
	}

	return privPath, nil
}

// ReadPublicKey reads the public key file from the data dir.
func ReadPublicKey(dataDir string) (string, error) {
	pubPath := filepath.Join(dataDir, "id_ed25519.pub")
	data, err := os.ReadFile(pubPath)
	if err != nil {
		return "", fmt.Errorf("read public key: %w", err)
	}
	return string(data), nil
}
