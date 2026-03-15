package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration.
type Config struct {
	Files   []string                `json:"files" yaml:"files"`
	Servers map[string]ServerConfig `json:"servers" yaml:"servers"`

	// configDir is the directory of the config file, used to resolve relative paths.
	// Not serialized.
	configDir string
}

// ConfigDir returns the directory of the config file.
func (c *Config) ConfigDir() string {
	return c.configDir
}

// ServerConfig defines either a local (command-based) or remote (URL-based) MCP server.
type ServerConfig struct {
	// Human-readable description shown in --help and skill documents.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Local server fields
	Command string            `json:"command,omitempty" yaml:"command,omitempty"`
	Args    []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Cwd     string            `json:"cwd,omitempty" yaml:"cwd,omitempty"`

	// Remote server fields
	URL     string            `json:"url,omitempty" yaml:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`

	// Tool filtering
	AllowTools []string `json:"allow_tools,omitempty" yaml:"allow_tools,omitempty"`
	DenyTools  []string `json:"deny_tools,omitempty" yaml:"deny_tools,omitempty"`

	// OAuth2 authentication
	OAuth *OAuthConfig `json:"oauth,omitempty" yaml:"oauth,omitempty"`

	// cwdAutoSet is true when Cwd was defaulted to the config file directory,
	// not explicitly set by the user. Not serialized.
	cwdAutoSet bool

	// rawEnv and rawHeaders store original values before env var resolution.
	// Used by BuildCompiledConfig to preserve ${VAR} patterns.
	rawEnv     map[string]string
	rawHeaders map[string]string
	rawOAuth   *OAuthConfig
}

// OAuthConfig holds optional OAuth2 settings for remote servers.
// All endpoints are auto-discovered from the server URL via RFC 9728/8414.
type OAuthConfig struct {
	ClientID     string   `json:"client_id,omitempty" yaml:"client_id,omitempty"`
	ClientSecret string   `json:"client_secret,omitempty" yaml:"client_secret,omitempty"`
	Scopes       []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

// ExplicitCwd returns the cwd only if it was explicitly set in the config file.
func (s ServerConfig) ExplicitCwd() string {
	if s.cwdAutoSet {
		return ""
	}
	return s.Cwd
}

func (s ServerConfig) IsLocal() bool {
	return s.Command != ""
}

func (s ServerConfig) IsRemote() bool {
	return s.URL != ""
}

// EnvValue stores a resolved value alongside its original variable name,
// allowing runtime override via os.Getenv.
type EnvValue struct {
	Value    string `json:"value" yaml:"value"`
	EnvVar   string `json:"envVar,omitempty" yaml:"envVar,omitempty"`
	Template string `json:"template,omitempty" yaml:"template,omitempty"`
}

// CompiledConfig is the config format stored inside the compiled binary's zip.
// It preserves env var metadata for runtime override.
type CompiledConfig struct {
	Files   []string                        `json:"files" yaml:"files"`
	Servers map[string]CompiledServerConfig `json:"servers" yaml:"servers"`
	BaseDir string                          `json:"baseDir,omitempty" yaml:"baseDir,omitempty"`
}

type CompiledServerConfig struct {
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Command     string               `json:"command,omitempty" yaml:"command,omitempty"`
	Args        []string             `json:"args,omitempty" yaml:"args,omitempty"`
	Env         map[string]EnvValue  `json:"env,omitempty" yaml:"env,omitempty"`
	Cwd         string               `json:"cwd,omitempty" yaml:"cwd,omitempty"`
	URL         string               `json:"url,omitempty" yaml:"url,omitempty"`
	Headers     map[string]EnvValue  `json:"headers,omitempty" yaml:"headers,omitempty"`
	AllowTools  []string             `json:"allow_tools,omitempty" yaml:"allow_tools,omitempty"`
	DenyTools   []string             `json:"deny_tools,omitempty" yaml:"deny_tools,omitempty"`
	OAuth       *CompiledOAuthConfig `json:"oauth,omitempty" yaml:"oauth,omitempty"`
}

// CompiledOAuthConfig preserves env var metadata for OAuth fields.
type CompiledOAuthConfig struct {
	ClientID     EnvValue `json:"client_id,omitempty" yaml:"client_id,omitempty"`
	ClientSecret EnvValue `json:"client_secret,omitempty" yaml:"client_secret,omitempty"`
	Scopes       []string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

// LoadFromFile reads and parses a config file (JSON or YAML, detected by extension).
// The config is validated against the embedded JSON Schema before parsing.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	isYAML := ext == ".yaml" || ext == ".yml"

	if err := validateAgainstSchema(data, isYAML); err != nil {
		return nil, err
	}

	var cfg Config
	if isYAML {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing YAML config: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing JSON config: %w", err)
		}
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.ResolveEnvVars()
	absDir, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("resolving config directory: %w", err)
	}
	cfg.configDir = absDir
	cfg.resolveFilePaths(absDir)
	return &cfg, nil
}

// LoadFromBytes parses config from raw JSON bytes (used for compiled config).
func LoadFromBytes(data []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// LoadCompiledConfig parses a CompiledConfig and resolves env vars with runtime override.
func LoadCompiledConfig(data []byte) (*Config, error) {
	var cc CompiledConfig
	if err := json.Unmarshal(data, &cc); err != nil {
		return nil, fmt.Errorf("parsing compiled config: %w", err)
	}

	cfg := &Config{
		Files:     cc.Files,
		Servers:   make(map[string]ServerConfig),
		configDir: cc.BaseDir,
	}

	for name, srv := range cc.Servers {
		sc := ServerConfig{
			Description: srv.Description,
			Command:     srv.Command,
			Args:        srv.Args,
			Cwd:         srv.Cwd,
			URL:         srv.URL,
			AllowTools:  srv.AllowTools,
			DenyTools:   srv.DenyTools,
		}
		if srv.Env != nil {
			sc.Env = make(map[string]string)
			for k, ev := range srv.Env {
				sc.Env[k] = resolveEnvValue(ev)
			}
		}
		if srv.Headers != nil {
			sc.Headers = make(map[string]string)
			for k, ev := range srv.Headers {
				sc.Headers[k] = resolveEnvValue(ev)
			}
		}
		if srv.OAuth != nil {
			sc.OAuth = &OAuthConfig{
				ClientID:     resolveEnvValue(srv.OAuth.ClientID),
				ClientSecret: resolveEnvValue(srv.OAuth.ClientSecret),
				Scopes:       srv.OAuth.Scopes,
			}
		}
		cfg.Servers[name] = sc
	}

	return cfg, nil
}

func resolveEnvValue(ev EnvValue) string {
	if ev.Template != "" {
		// Check if all referenced env vars are set at runtime
		matches := EnvVarPattern.FindAllStringSubmatch(ev.Template, -1)
		allSet := true
		for _, m := range matches {
			if _, ok := os.LookupEnv(m[1]); !ok {
				allSet = false
				break
			}
		}
		if allSet {
			return resolveEnvString(ev.Template)
		}
		return ev.Value
	}
	if ev.EnvVar != "" {
		if v, ok := os.LookupEnv(ev.EnvVar); ok {
			return v
		}
	}
	return ev.Value
}

// Validate checks config invariants not expressible in JSON Schema.
func (c *Config) Validate() error {
	for name, srv := range c.Servers {
		for _, pattern := range srv.AllowTools {
			if _, err := filepath.Match(pattern, ""); err != nil {
				return fmt.Errorf("server %q: invalid allow_tools pattern %q: %w", name, pattern, err)
			}
		}
		for _, pattern := range srv.DenyTools {
			if _, err := filepath.Match(pattern, ""); err != nil {
				return fmt.Errorf("server %q: invalid deny_tools pattern %q: %w", name, pattern, err)
			}
		}
	}
	return nil
}

// resolveFilePaths makes relative file paths absolute and sets default cwd
// for local servers, both relative to the config file's directory.
func (c *Config) resolveFilePaths(absDir string) {
	for i, f := range c.Files {
		if !filepath.IsAbs(f) {
			c.Files[i] = filepath.Join(absDir, f)
		}
	}
	for name, srv := range c.Servers {
		if srv.IsLocal() && srv.Cwd == "" {
			srv.Cwd = absDir
			srv.cwdAutoSet = true
			c.Servers[name] = srv
		}
	}
}

// PatchFiles updates embedded file paths and sets local server cwd
// to the extracted dirs root so relative paths in args resolve correctly.
// The baseDir parameter is the original config directory used during compilation
// to compute relative paths for absolute file entries.
func (c *Config) PatchFiles(dirsRoot string, baseDir string) {
	for i, f := range c.Files {
		if filepath.IsAbs(f) {
			if baseDir != "" {
				if rel, err := filepath.Rel(baseDir, f); err == nil && !strings.HasPrefix(rel, "..") {
					c.Files[i] = filepath.Join(dirsRoot, rel)
					continue
				}
			}
			c.Files[i] = filepath.Join(dirsRoot, filepath.Base(f))
		} else {
			c.Files[i] = filepath.Join(dirsRoot, f)
		}
	}

	for name, srv := range c.Servers {
		if !srv.IsLocal() {
			continue
		}
		// Only change cwd when there are embedded files to resolve.
		// Without embedded files, the dirs directory doesn't exist.
		if len(c.Files) == 0 {
			continue
		}
		if srv.Cwd != "" {
			if baseDir != "" {
				if rel, err := filepath.Rel(baseDir, srv.Cwd); err == nil && !strings.HasPrefix(rel, "..") {
					srv.Cwd = filepath.Join(dirsRoot, rel)
					c.Servers[name] = srv
					continue
				}
			}
			srv.Cwd = filepath.Join(dirsRoot, filepath.Base(srv.Cwd))
		} else {
			srv.Cwd = dirsRoot
		}
		c.Servers[name] = srv
	}
}

// SortedServerNames returns the server names from the config in sorted order.
func SortedServerNames(cfg *Config) []string {
	names := make([]string, 0, len(cfg.Servers))
	for name := range cfg.Servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
