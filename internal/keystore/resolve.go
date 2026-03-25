package keystore

import "github.com/awkto/ssh-to-go/internal/config"

// ResolveKeyPath returns the private key file path for a host.
// Priority: host.KeyName -> host.KeyPath -> settings default keypair.
func ResolveKeyPath(host config.Host, ks *Store, sm *SettingsManager) string {
	if host.KeyName != "" && ks.Exists(host.KeyName) {
		return ks.PrivateKeyPath(host.KeyName)
	}
	if host.KeyPath != "" {
		return host.KeyPath
	}
	return ks.PrivateKeyPath(sm.DefaultKeypairName())
}

// ResolveUser returns the user for a host, falling back to the default username.
func ResolveUser(host config.Host, sm *SettingsManager) string {
	if host.User != "" {
		return host.User
	}
	defaultUser := sm.DefaultUsername()
	if defaultUser != "" {
		return defaultUser
	}
	return "root"
}
