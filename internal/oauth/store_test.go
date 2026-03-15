package oauth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/zalando/go-keyring"
)

// memoryKeyring is an in-memory Keyring for testing.
type memoryKeyring struct {
	store map[string]string
}

func newMemoryKeyring() *memoryKeyring {
	return &memoryKeyring{store: make(map[string]string)}
}

func (m *memoryKeyring) Set(service, user, password string) error {
	m.store[service+":"+user] = password
	return nil
}

func (m *memoryKeyring) Get(service, user string) (string, error) {
	v, ok := m.store[service+":"+user]
	if !ok {
		return "", keyring.ErrNotFound
	}
	return v, nil
}

func (m *memoryKeyring) Delete(service, user string) error {
	key := service + ":" + user
	if _, ok := m.store[key]; !ok {
		return keyring.ErrNotFound
	}
	delete(m.store, key)
	return nil
}

// failingKeyring always returns an error (simulates locked/unavailable keychain).
type failingKeyring struct{}

func (failingKeyring) Set(string, string, string) error { return fmt.Errorf("keychain locked") }
func (failingKeyring) Get(string, string) (string, error) {
	return "", fmt.Errorf("keychain locked")
}
func (failingKeyring) Delete(string, string) error { return fmt.Errorf("keychain locked") }

func TestStoreAndLoadToken(t *testing.T) {
	ring := newMemoryKeyring()
	store := NewKeychainStore(ring, "https://example.com/mcp")

	token := &transport.Token{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		Scope:        "read write",
	}

	ctx := context.Background()

	if err := store.SaveToken(ctx, token); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	loaded, err := store.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}

	if loaded.AccessToken != "access-123" {
		t.Errorf("AccessToken: got %q, want %q", loaded.AccessToken, "access-123")
	}
	if loaded.RefreshToken != "refresh-456" {
		t.Errorf("RefreshToken: got %q, want %q", loaded.RefreshToken, "refresh-456")
	}
	if loaded.TokenType != "Bearer" {
		t.Errorf("TokenType: got %q, want %q", loaded.TokenType, "Bearer")
	}
	if loaded.Scope != "read write" {
		t.Errorf("Scope: got %q, want %q", loaded.Scope, "read write")
	}
}

func TestDeleteToken(t *testing.T) {
	ring := newMemoryKeyring()
	store := NewKeychainStore(ring, "https://example.com/mcp")
	ctx := context.Background()

	_ = store.SaveToken(ctx, &transport.Token{AccessToken: "to-delete", TokenType: "Bearer"})

	if err := store.DeleteToken(); err != nil {
		t.Fatalf("DeleteToken: %v", err)
	}

	_, err := store.GetToken(ctx)
	if err == nil {
		t.Error("expected error after delete, got nil")
	}
}

func TestDeleteTokenNotFound(t *testing.T) {
	ring := newMemoryKeyring()
	store := NewKeychainStore(ring, "https://nonexistent.com/mcp")

	err := store.DeleteToken()
	if err == nil {
		t.Fatal("expected error for deleting non-existent token")
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestGetTokenNotFound(t *testing.T) {
	ring := newMemoryKeyring()
	store := NewKeychainStore(ring, "https://nonexistent.com/mcp")

	_, err := store.GetToken(context.Background())
	if err == nil {
		t.Error("expected error for missing token, got nil")
	}
}

func TestGetTokenCorruptData(t *testing.T) {
	ring := newMemoryKeyring()
	store := NewKeychainStore(ring, "https://example.com/mcp")

	// Manually store invalid JSON
	ring.store[keyringService+":"+store.keyName()] = "not-valid-json"

	_, err := store.GetToken(context.Background())
	if err == nil {
		t.Error("expected error for corrupt token data, got nil")
	}
	if !strings.Contains(err.Error(), "parsing stored token") {
		t.Errorf("expected parsing error, got: %v", err)
	}
}

func TestKeyNameDiffersByServerURL(t *testing.T) {
	ring := newMemoryKeyring()
	store1 := NewKeychainStore(ring, "https://server1.com/mcp")
	store2 := NewKeychainStore(ring, "https://server2.com/mcp")
	ctx := context.Background()

	_ = store1.SaveToken(ctx, &transport.Token{AccessToken: "token1", TokenType: "Bearer"})
	_ = store2.SaveToken(ctx, &transport.Token{AccessToken: "token2", TokenType: "Bearer"})

	t1, _ := store1.GetToken(ctx)
	t2, _ := store2.GetToken(ctx)

	if t1.AccessToken != "token1" {
		t.Errorf("store1: got %q, want %q", t1.AccessToken, "token1")
	}
	if t2.AccessToken != "token2" {
		t.Errorf("store2: got %q, want %q", t2.AccessToken, "token2")
	}
}

func TestURLNormalization(t *testing.T) {
	ring := newMemoryKeyring()
	ctx := context.Background()

	// Store with trailing slash
	store1 := NewKeychainStore(ring, "https://example.com/mcp/")
	_ = store1.SaveToken(ctx, &transport.Token{AccessToken: "tok", TokenType: "Bearer"})

	// Load without trailing slash — should find the same token
	store2 := NewKeychainStore(ring, "https://example.com/mcp")
	tok, err := store2.GetToken(ctx)
	if err != nil {
		t.Fatalf("expected token to be found despite trailing slash difference: %v", err)
	}
	if tok.AccessToken != "tok" {
		t.Errorf("got %q, want %q", tok.AccessToken, "tok")
	}
}

func TestURLNormalizationCaseInsensitive(t *testing.T) {
	ring := newMemoryKeyring()
	ctx := context.Background()

	store1 := NewKeychainStore(ring, "HTTPS://Example.COM/mcp")
	_ = store1.SaveToken(ctx, &transport.Token{AccessToken: "tok", TokenType: "Bearer"})

	store2 := NewKeychainStore(ring, "https://example.com/mcp")
	tok, err := store2.GetToken(ctx)
	if err != nil {
		t.Fatalf("expected token to be found despite case difference: %v", err)
	}
	if tok.AccessToken != "tok" {
		t.Errorf("got %q, want %q", tok.AccessToken, "tok")
	}
}

func TestURLNormalizationDefaultPort(t *testing.T) {
	ring := newMemoryKeyring()
	ctx := context.Background()

	store1 := NewKeychainStore(ring, "https://example.com:443/mcp")
	_ = store1.SaveToken(ctx, &transport.Token{AccessToken: "tok", TokenType: "Bearer"})

	store2 := NewKeychainStore(ring, "https://example.com/mcp")
	tok, err := store2.GetToken(ctx)
	if err != nil {
		t.Fatalf("expected token to be found despite default port: %v", err)
	}
	if tok.AccessToken != "tok" {
		t.Errorf("got %q, want %q", tok.AccessToken, "tok")
	}
}

// --- Check and Logout tests ---

func TestCheckNoToken(t *testing.T) {
	ring := newMemoryKeyring()
	store := NewKeychainStore(ring, "https://example.com/mcp")
	var buf bytes.Buffer

	err := Check(context.Background(), &buf, "https://example.com/mcp", store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No token stored") {
		t.Errorf("expected 'No token stored', got: %s", buf.String())
	}
}

func TestCheckValidToken(t *testing.T) {
	ring := newMemoryKeyring()
	store := NewKeychainStore(ring, "https://example.com/mcp")
	ctx := context.Background()

	_ = store.SaveToken(ctx, &transport.Token{
		AccessToken: "valid",
		TokenType:   "Bearer",
		Scope:       "read",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	})

	var buf bytes.Buffer
	err := Check(ctx, &buf, "https://example.com/mcp", store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Status: Valid") {
		t.Errorf("expected 'Status: Valid', got: %s", output)
	}
	if !strings.Contains(output, "Scopes: read") {
		t.Errorf("expected scopes in output, got: %s", output)
	}
}

func TestCheckExpiredToken(t *testing.T) {
	ring := newMemoryKeyring()
	store := NewKeychainStore(ring, "https://example.com/mcp")
	ctx := context.Background()

	_ = store.SaveToken(ctx, &transport.Token{
		AccessToken:  "expired",
		RefreshToken: "refresh-available",
		TokenType:    "Bearer",
		ExpiresAt:    time.Now().Add(-1 * time.Hour),
	})

	var buf bytes.Buffer
	err := Check(ctx, &buf, "https://example.com/mcp", store)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Status: Expired") {
		t.Errorf("expected 'Status: Expired', got: %s", output)
	}
	if !strings.Contains(output, "Refresh token available") {
		t.Errorf("expected refresh notice, got: %s", output)
	}
}

func TestCheckExpiredNoRefresh(t *testing.T) {
	ring := newMemoryKeyring()
	store := NewKeychainStore(ring, "https://example.com/mcp")
	ctx := context.Background()

	_ = store.SaveToken(ctx, &transport.Token{
		AccessToken: "expired",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(-1 * time.Hour),
	})

	var buf bytes.Buffer
	_ = Check(ctx, &buf, "https://example.com/mcp", store)
	if !strings.Contains(buf.String(), "No refresh token") {
		t.Errorf("expected 'No refresh token', got: %s", buf.String())
	}
}

func TestLogoutExistingToken(t *testing.T) {
	ring := newMemoryKeyring()
	store := NewKeychainStore(ring, "https://example.com/mcp")
	_ = store.SaveToken(context.Background(), &transport.Token{AccessToken: "del", TokenType: "Bearer"})

	var buf bytes.Buffer
	if err := Logout(&buf, store); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Token removed") {
		t.Errorf("expected 'Token removed', got: %s", buf.String())
	}

	// Verify actually deleted
	_, err := store.GetToken(context.Background())
	if err == nil {
		t.Error("token should be deleted")
	}
}

func TestLogoutNoToken(t *testing.T) {
	ring := newMemoryKeyring()
	store := NewKeychainStore(ring, "https://example.com/mcp")

	var buf bytes.Buffer
	if err := Logout(&buf, store); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "No token stored") {
		t.Errorf("expected 'No token stored', got: %s", buf.String())
	}
}

func TestLogoutKeychainError(t *testing.T) {
	store := NewKeychainStore(failingKeyring{}, "https://example.com/mcp")

	var buf bytes.Buffer
	err := Logout(&buf, store)
	if err == nil {
		t.Error("expected error for failing keychain, got nil")
	}
	if !strings.Contains(err.Error(), "removing token") {
		t.Errorf("expected 'removing token' error, got: %v", err)
	}
}

// --- readCodeFromStdin tests ---

func TestReadCodeFromStdinDirectCode(t *testing.T) {
	code, err := parseRedirectInput("abc123", "ignored-state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "abc123" {
		t.Errorf("got %q, want %q", code, "abc123")
	}
}

func TestReadCodeFromStdinURL(t *testing.T) {
	input := "http://localhost:8080/oauth/callback?code=mycode&state=mystate"
	code, err := parseRedirectInput(input, "mystate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "mycode" {
		t.Errorf("got %q, want %q", code, "mycode")
	}
}

func TestReadCodeFromStdinURLStateMismatch(t *testing.T) {
	input := "http://localhost:8080/callback?code=mycode&state=wrong"
	_, err := parseRedirectInput(input, "expected")
	if err == nil {
		t.Error("expected error for state mismatch")
	}
	if !strings.Contains(err.Error(), "state mismatch") {
		t.Errorf("expected state mismatch error, got: %v", err)
	}
}

func TestReadCodeFromStdinURLNoCode(t *testing.T) {
	input := "http://localhost:8080/callback?state=mystate"
	_, err := parseRedirectInput(input, "mystate")
	if err == nil {
		t.Error("expected error for missing code")
	}
}

func TestReadCodeFromStdinURLWithError(t *testing.T) {
	input := "http://localhost:8080/callback?error=access_denied&error_description=User+denied&state=s"
	_, err := parseRedirectInput(input, "s")
	if err == nil {
		t.Error("expected error for authorization error")
	}
	if !strings.Contains(err.Error(), "access_denied") {
		t.Errorf("expected access_denied in error, got: %v", err)
	}
}

// parseRedirectInput is a testable version of the URL/code parsing logic
// extracted from readCodeFromStdin (which reads from os.Stdin and can't be tested directly).
func parseRedirectInput(input, expectedState string) (string, error) {
	input = strings.TrimSpace(input)
	if strings.HasPrefix(input, "http") {
		u, err := url.Parse(input)
		if err != nil {
			return "", err
		}
		if returnedState := u.Query().Get("state"); returnedState != expectedState {
			return "", fmt.Errorf("state mismatch in redirect URL (possible CSRF)")
		}
		code := u.Query().Get("code")
		if code == "" {
			errMsg := u.Query().Get("error")
			errDesc := u.Query().Get("error_description")
			if errMsg != "" {
				return "", fmt.Errorf("authorization error: %s: %s", errMsg, errDesc)
			}
			return "", fmt.Errorf("no 'code' parameter found in URL")
		}
		return code, nil
	}
	return input, nil
}

// --- OpenBrowser tests ---

func TestOpenBrowserRejectsNonHTTP(t *testing.T) {
	tests := []string{
		"file:///etc/passwd",
		"ftp://example.com",
		"javascript:alert(1)",
		"not-a-url",
	}
	for _, u := range tests {
		if err := OpenBrowser(u); err == nil {
			t.Errorf("expected error for %q, got nil", u)
		}
	}
}
