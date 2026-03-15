package oauth

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

// OpenBrowser opens the given URL in the user's default browser.
// Only HTTP and HTTPS URLs are allowed for security.
func OpenBrowser(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return fmt.Errorf("refusing to open non-HTTP URL in browser: %s", rawURL)
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
