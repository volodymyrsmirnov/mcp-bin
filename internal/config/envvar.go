package config

import (
	"os"
	"regexp"
)

// EnvVarPattern matches ${VAR} env var references in config values.
var EnvVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// ResolveEnvVars replaces ${VAR} patterns in the config with environment variable values.
// It preserves the original (raw) values for use by BuildCompiledConfig.
func (c *Config) ResolveEnvVars() {
	for name, srv := range c.Servers {
		if srv.Env != nil {
			srv.rawEnv = make(map[string]string, len(srv.Env))
			for k, v := range srv.Env {
				srv.rawEnv[k] = v
				srv.Env[k] = resolveEnvString(v)
			}
		}
		if srv.Headers != nil {
			srv.rawHeaders = make(map[string]string, len(srv.Headers))
			for k, v := range srv.Headers {
				srv.rawHeaders[k] = v
				srv.Headers[k] = resolveEnvString(v)
			}
		}
		c.Servers[name] = srv
	}
}

func resolveEnvString(s string) string {
	return EnvVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		varName := EnvVarPattern.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return match // keep original if not set
	})
}

// BuildCompiledConfig creates a CompiledConfig from a Config,
// preserving env var metadata for runtime override.
// It uses the raw (pre-resolution) values stored by ResolveEnvVars
// so that ${VAR} patterns are available for runtime override.
func BuildCompiledConfig(c *Config) *CompiledConfig {
	cc := &CompiledConfig{
		Files:   c.Files,
		Servers: make(map[string]CompiledServerConfig),
		BaseDir: c.configDir,
	}

	for name, srv := range c.Servers {
		cs := CompiledServerConfig{
			Description: srv.Description,
			Command:     srv.Command,
			Args:        srv.Args,
			Cwd:         srv.ExplicitCwd(),
			URL:         srv.URL,
			AllowTools:  srv.AllowTools,
			DenyTools:   srv.DenyTools,
		}
		if srv.Env != nil {
			cs.Env = make(map[string]EnvValue)
			rawEnv := srv.rawEnv
			if rawEnv == nil {
				rawEnv = srv.Env
			}
			for k, raw := range rawEnv {
				cs.Env[k] = buildEnvValue(raw)
			}
		}
		if srv.Headers != nil {
			cs.Headers = make(map[string]EnvValue)
			rawHeaders := srv.rawHeaders
			if rawHeaders == nil {
				rawHeaders = srv.Headers
			}
			for k, raw := range rawHeaders {
				cs.Headers[k] = buildEnvValue(raw)
			}
		}
		cc.Servers[name] = cs
	}

	return cc
}

func buildEnvValue(raw string) EnvValue {
	matches := EnvVarPattern.FindAllStringSubmatch(raw, -1)
	if len(matches) == 0 {
		return EnvValue{Value: raw}
	}

	// Simple case: entire value is a single env var like "${TOKEN}"
	if len(matches) == 1 && raw == matches[0][0] {
		return EnvValue{
			Value:  os.Getenv(matches[0][1]),
			EnvVar: matches[0][1],
		}
	}

	// Mixed case: literals + env vars (e.g., "Bearer ${TOKEN}")
	// or multiple env vars (e.g., "${HOST}:${PORT}")
	resolved := resolveEnvString(raw)
	return EnvValue{
		Value:    resolved,
		Template: raw,
	}
}
