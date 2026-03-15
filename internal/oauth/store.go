package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/zalando/go-keyring"
)

const keyringService = "mcp-bin"

// Keyring abstracts system keychain operations for testability.
type Keyring interface {
	Set(service, user, password string) error
	Get(service, user string) (string, error)
	Delete(service, user string) error
}

// systemKeyring wraps go-keyring for production use.
type systemKeyring struct{}

func (systemKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}
func (systemKeyring) Get(service, user string) (string, error) { return keyring.Get(service, user) }
func (systemKeyring) Delete(service, user string) error        { return keyring.Delete(service, user) }

// SystemKeyring returns a Keyring backed by the system keychain.
func SystemKeyring() Keyring { return systemKeyring{} }

// KeychainStore implements transport.TokenStore using the system keychain.
type KeychainStore struct {
	ring      Keyring
	serverURL string
}

// NewKeychainStore creates a KeychainStore for the given server URL.
// The URL is normalized (lowercased scheme/host, trailing slash stripped)
// to avoid key collisions from cosmetic URL differences.
func NewKeychainStore(ring Keyring, serverURL string) *KeychainStore {
	return &KeychainStore{ring: ring, serverURL: normalizeURL(serverURL)}
}

// normalizeURL lowercases scheme and host, strips trailing slashes and default ports,
// so that cosmetically different URLs that refer to the same server produce the same key.
func normalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	// Remove default ports
	if (u.Scheme == "https" && u.Port() == "443") || (u.Scheme == "http" && u.Port() == "80") {
		u.Host = u.Hostname()
	}
	return u.String()
}

func (s *KeychainStore) keyName() string {
	return "oauth:" + s.serverURL
}

// GetToken loads a token from the system keychain.
func (s *KeychainStore) GetToken(_ context.Context) (*transport.Token, error) {
	data, err := s.ring.Get(keyringService, s.keyName())
	if err != nil {
		return nil, fmt.Errorf("no stored token: %w", err)
	}
	var token transport.Token
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, fmt.Errorf("parsing stored token: %w", err)
	}
	return &token, nil
}

// SaveToken stores a token in the system keychain.
func (s *KeychainStore) SaveToken(_ context.Context, token *transport.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("serializing token: %w", err)
	}
	return s.ring.Set(keyringService, s.keyName(), string(data))
}

// DeleteToken removes a token from the system keychain.
func (s *KeychainStore) DeleteToken() error {
	return s.ring.Delete(keyringService, s.keyName())
}
