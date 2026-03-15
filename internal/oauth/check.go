package oauth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/zalando/go-keyring"
)

// Check loads the stored OAuth token and reports its status.
func Check(ctx context.Context, w io.Writer, serverURL string, store *KeychainStore) error {
	token, err := store.GetToken(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(w, "Status: No token stored")
		_, _ = fmt.Fprintln(w, "Run 'mcp-bin oauth login <server>' to authenticate.")
		return nil
	}

	_, _ = fmt.Fprintf(w, "Server: %s\n", serverURL)
	_, _ = fmt.Fprintf(w, "Token type: %s\n", token.TokenType)
	if token.Scope != "" {
		_, _ = fmt.Fprintf(w, "Scopes: %s\n", token.Scope)
	}

	if !token.ExpiresAt.IsZero() {
		_, _ = fmt.Fprintf(w, "Expires: %s\n", token.ExpiresAt.Format(time.RFC3339))
	}

	if token.IsExpired() {
		_, _ = fmt.Fprintln(w, "Status: Expired")
		if token.RefreshToken != "" {
			_, _ = fmt.Fprintln(w, "Refresh token available. Token will be refreshed on next use.")
		} else {
			_, _ = fmt.Fprintln(w, "No refresh token. Run 'mcp-bin oauth login <server>' to re-authenticate.")
		}
	} else {
		_, _ = fmt.Fprintln(w, "Status: Valid")
		if !token.ExpiresAt.IsZero() {
			remaining := time.Until(token.ExpiresAt)
			_, _ = fmt.Fprintf(w, "Expires in: %s\n", remaining.Truncate(time.Second))
		}
	}

	return nil
}

// Logout deletes the stored OAuth token for a server.
func Logout(w io.Writer, store *KeychainStore) error {
	if err := store.DeleteToken(); err != nil {
		// Only treat "not found" as success — real errors must surface
		if errors.Is(err, keyring.ErrNotFound) {
			_, _ = fmt.Fprintln(w, "No token stored for this server.")
			return nil
		}
		return fmt.Errorf("removing token: %w", err)
	}
	_, _ = fmt.Fprintln(w, "Token removed from system keychain.")
	return nil
}
