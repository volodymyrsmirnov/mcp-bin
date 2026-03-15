package oauth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client/transport"

	"github.com/volodymyrsmirnov/mcp-bin/internal/config"
)

// FlowOptions controls the interactive OAuth login flow.
type FlowOptions struct {
	Port      int  // Local callback server port. 0 = OS-assigned.
	NoBrowser bool // Skip auto-opening browser; prompt user to paste URL.
}

// Login performs an interactive OAuth2 authorization code flow with PKCE.
// It discovers endpoints from the server URL, opens the browser for consent,
// handles the callback, exchanges the code for tokens, and stores them.
func Login(ctx context.Context, w io.Writer, serverURL string, oauthCfg *config.OAuthConfig, store *KeychainStore, opts FlowOptions) error {
	// Start local callback server
	addr := fmt.Sprintf("localhost:%d", opts.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("starting callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/oauth/callback", port)

	// Build OAuth config for mcp-go
	transportCfg := transport.OAuthConfig{
		RedirectURI: redirectURI,
		TokenStore:  store,
		PKCEEnabled: true,
	}
	if oauthCfg != nil {
		transportCfg.ClientID = oauthCfg.ClientID
		transportCfg.ClientSecret = oauthCfg.ClientSecret
		transportCfg.Scopes = oauthCfg.Scopes
	}

	// Create an OAuthHandler directly to drive the login flow.
	// We don't use NewOAuthStreamableHttpClient + Start because the server
	// may not return 401 during initial connection, but we still need to
	// perform the OAuth flow when the user explicitly runs "login".
	handler := transport.NewOAuthHandler(transportCfg)
	handler.SetBaseURL(serverURL)

	// Discover server metadata (authorization endpoint, token endpoint, etc.)
	_, _ = fmt.Fprintln(w, "Discovering OAuth endpoints...")
	metadata, err := handler.GetServerMetadata(ctx)
	if err != nil {
		_ = listener.Close()
		return fmt.Errorf("discovering OAuth endpoints: %w", err)
	}

	// Register client dynamically if no client ID was provided
	if handler.GetClientID() == "" {
		_, _ = fmt.Fprintln(w, "Registering OAuth client...")
		if metadata.RegistrationEndpoint == "" {
			_ = listener.Close()
			return fmt.Errorf("server does not support dynamic client registration; set client_id in config")
		}
		if regErr := handler.RegisterClient(ctx, "mcp-bin"); regErr != nil {
			_ = listener.Close()
			return fmt.Errorf("registering OAuth client: %w", regErr)
		}
	}

	// Generate PKCE and state
	state, err := transport.GenerateState()
	if err != nil {
		_ = listener.Close()
		return fmt.Errorf("generating state: %w", err)
	}
	codeVerifier, err := transport.GenerateCodeVerifier()
	if err != nil {
		_ = listener.Close()
		return fmt.Errorf("generating code verifier: %w", err)
	}
	codeChallenge := transport.GenerateCodeChallenge(codeVerifier)

	// Get authorization URL
	authURL, err := handler.GetAuthorizationURL(ctx, state, codeChallenge)
	if err != nil {
		_ = listener.Close()
		return fmt.Errorf("building authorization URL: %w", err)
	}

	// Set up callback handler with non-blocking sends to prevent goroutine leaks.
	// sync.Once ensures only the first callback request is processed.
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	var callbackOnce sync.Once

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", func(rw http.ResponseWriter, r *http.Request) {
		callbackOnce.Do(func() {
			code := r.URL.Query().Get("code")
			returnedState := r.URL.Query().Get("state")
			if returnedState != state {
				// Do not include the expected state value in the error — it is a CSRF secret.
				select {
				case errCh <- fmt.Errorf("state mismatch in callback (possible CSRF)"):
				default:
				}
				http.Error(rw, "State mismatch", http.StatusBadRequest)
				return
			}
			if code == "" {
				errMsg := r.URL.Query().Get("error")
				errDesc := r.URL.Query().Get("error_description")
				select {
				case errCh <- fmt.Errorf("authorization error: %s: %s", errMsg, errDesc):
				default:
				}
				http.Error(rw, "Authorization failed: "+errMsg, http.StatusBadRequest)
				return
			}
			select {
			case codeCh <- code:
			default:
			}
			_, _ = fmt.Fprint(rw, "<html><body><h1>Authorization successful!</h1><p>You can close this tab and return to the terminal.</p></body></html>")
		})
	})

	server := &http.Server{Handler: mux}
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			select {
			case errCh <- serveErr:
			default:
			}
		}
	}()
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutCtx)
	}()

	// Direct user to authorization URL
	_, _ = fmt.Fprintf(w, "Open this URL in your browser to authorize:\n\n  %s\n\n", authURL)

	if opts.NoBrowser {
		_, _ = fmt.Fprintln(w, "After authorizing, paste the redirect URL here and press Enter:")
	} else {
		_, _ = fmt.Fprintln(w, "Opening browser...")
		if browserErr := OpenBrowser(authURL); browserErr != nil {
			_, _ = fmt.Fprintf(w, "Could not open browser: %v\nPlease open the URL above manually.\n", browserErr)
		}
	}

	// Wait for callback or manual paste
	var code string
	if opts.NoBrowser {
		code, err = readCodeFromStdin(state)
		if err != nil {
			return err
		}
	} else {
		// Wait for callback with timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		select {
		case code = <-codeCh:
		case err := <-errCh:
			return err
		case <-timeoutCtx.Done():
			return fmt.Errorf("authorization timed out (5 minutes)")
		}
	}

	// Exchange code for tokens
	_, _ = fmt.Fprintln(w, "Exchanging authorization code for tokens...")
	if err := handler.ProcessAuthorizationResponse(ctx, code, state, codeVerifier); err != nil {
		return fmt.Errorf("exchanging authorization code: %w", err)
	}

	_, _ = fmt.Fprintln(w, "Authentication successful! Tokens stored in system keychain.")
	return nil
}

// readCodeFromStdin reads a redirect URL or auth code from stdin.
// If a URL is provided, it validates the state parameter against expectedState.
func readCodeFromStdin(expectedState string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(input)

	// If it's a URL, extract and validate
	if strings.HasPrefix(input, "http") {
		u, err := url.Parse(input)
		if err != nil {
			return "", fmt.Errorf("parsing redirect URL: %w", err)
		}
		// Validate state to prevent CSRF
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

	// Assume it's the code directly
	return input, nil
}
