package config

import (
	"os"
	"testing"
)

func TestResolveEnvVars(t *testing.T) {
	t.Setenv("TEST_API_KEY", "my-key")
	t.Setenv("TEST_TOKEN", "my-token")

	cfg := &Config{
		Servers: map[string]ServerConfig{
			"s1": {
				Command: "node",
				Env: map[string]string{
					"API_KEY": "${TEST_API_KEY}",
					"STATIC":  "no-var-here",
				},
				Headers: map[string]string{
					"Auth": "Bearer ${TEST_TOKEN}",
				},
			},
		},
	}

	cfg.ResolveEnvVars()

	if cfg.Servers["s1"].Env["API_KEY"] != "my-key" {
		t.Errorf("expected my-key, got %s", cfg.Servers["s1"].Env["API_KEY"])
	}
	if cfg.Servers["s1"].Env["STATIC"] != "no-var-here" {
		t.Errorf("expected no-var-here, got %s", cfg.Servers["s1"].Env["STATIC"])
	}
	if cfg.Servers["s1"].Headers["Auth"] != "Bearer my-token" {
		t.Errorf("expected Bearer my-token, got %s", cfg.Servers["s1"].Headers["Auth"])
	}
}

func TestResolveEnvVarsUnset(t *testing.T) {
	_ = os.Unsetenv("TOTALLY_UNSET_VAR_XYZ")

	cfg := &Config{
		Servers: map[string]ServerConfig{
			"s1": {
				Command: "node",
				Env: map[string]string{
					"KEY": "${TOTALLY_UNSET_VAR_XYZ}",
				},
			},
		},
	}

	cfg.ResolveEnvVars()

	// Should keep original pattern when env var is not set
	if cfg.Servers["s1"].Env["KEY"] != "${TOTALLY_UNSET_VAR_XYZ}" {
		t.Errorf("expected original pattern, got %s", cfg.Servers["s1"].Env["KEY"])
	}
}

func TestResolveEnvVarsNilMaps(t *testing.T) {
	cfg := &Config{
		Servers: map[string]ServerConfig{
			"s1": {Command: "node"},
		},
	}

	// Should not panic
	cfg.ResolveEnvVars()
}

func TestBuildCompiledConfig(t *testing.T) {
	t.Setenv("TEST_BUILD_KEY", "resolved")

	cfg := &Config{
		Files: []string{"/dir1", "/dir2"},
		Servers: map[string]ServerConfig{
			"local": {
				Command: "node",
				Args:    []string{"server.js"},
				Cwd:     "/path/to/dir",
				Env: map[string]string{
					"KEY": "${TEST_BUILD_KEY}",
				},
			},
			"remote": {
				URL: "https://example.com",
				Headers: map[string]string{
					"Auth": "${TEST_BUILD_KEY}",
				},
			},
			"plain": {
				Command: "python",
			},
		},
	}

	// Simulate the real flow: ResolveEnvVars stores raw values, then BuildCompiledConfig uses them
	cfg.ResolveEnvVars()
	cc := BuildCompiledConfig(cfg)

	if len(cc.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(cc.Files))
	}

	// Check local server - env var should be captured even though it was resolved
	local := cc.Servers["local"]
	if local.Command != "node" {
		t.Errorf("expected node, got %s", local.Command)
	}
	if local.Env["KEY"].Value != "resolved" {
		t.Errorf("expected resolved, got %s", local.Env["KEY"].Value)
	}
	if local.Env["KEY"].EnvVar != "TEST_BUILD_KEY" {
		t.Errorf("expected TEST_BUILD_KEY, got %s", local.Env["KEY"].EnvVar)
	}

	// Check remote server
	remote := cc.Servers["remote"]
	if remote.URL != "https://example.com" {
		t.Errorf("expected url, got %s", remote.URL)
	}
	if remote.Headers["Auth"].EnvVar != "TEST_BUILD_KEY" {
		t.Errorf("expected TEST_BUILD_KEY, got %s", remote.Headers["Auth"].EnvVar)
	}

	// Check plain server (no env/headers)
	plain := cc.Servers["plain"]
	if plain.Env != nil {
		t.Error("expected nil env")
	}
	if plain.Headers != nil {
		t.Error("expected nil headers")
	}
}

func TestBuildEnvValueNoPattern(t *testing.T) {
	ev := buildEnvValue("static-value")
	if ev.Value != "static-value" {
		t.Errorf("expected static-value, got %s", ev.Value)
	}
	if ev.EnvVar != "" {
		t.Errorf("expected empty envVar, got %s", ev.EnvVar)
	}
}

func TestBuildEnvValueWithPattern(t *testing.T) {
	t.Setenv("TEST_EV_VAR", "resolved-val")

	ev := buildEnvValue("${TEST_EV_VAR}")
	if ev.Value != "resolved-val" {
		t.Errorf("expected resolved-val, got %s", ev.Value)
	}
	if ev.EnvVar != "TEST_EV_VAR" {
		t.Errorf("expected TEST_EV_VAR, got %s", ev.EnvVar)
	}
}

func TestBuildEnvValueUnsetPattern(t *testing.T) {
	_ = os.Unsetenv("UNSET_BUILD_VAR_XYZ")

	ev := buildEnvValue("${UNSET_BUILD_VAR_XYZ}")
	if ev.Value != "" {
		t.Errorf("expected empty value, got %s", ev.Value)
	}
	if ev.EnvVar != "UNSET_BUILD_VAR_XYZ" {
		t.Errorf("expected UNSET_BUILD_VAR_XYZ, got %s", ev.EnvVar)
	}
}

func TestBuildCompiledConfigPreservesToolFilters(t *testing.T) {
	cfg := &Config{
		Files: []string{},
		Servers: map[string]ServerConfig{
			"s1": {
				Command:    "node",
				AllowTools: []string{"read_*"},
				DenyTools:  []string{"read_secret"},
			},
		},
	}

	cc := BuildCompiledConfig(cfg)

	s1 := cc.Servers["s1"]
	if len(s1.AllowTools) != 1 || s1.AllowTools[0] != "read_*" {
		t.Errorf("expected allow_tools [read_*], got %v", s1.AllowTools)
	}
	if len(s1.DenyTools) != 1 || s1.DenyTools[0] != "read_secret" {
		t.Errorf("expected deny_tools [read_secret], got %v", s1.DenyTools)
	}
}

func TestResolveEnvStringMultipleVars(t *testing.T) {
	t.Setenv("TEST_HOST", "localhost")
	t.Setenv("TEST_PORT", "8080")

	result := resolveEnvString("http://${TEST_HOST}:${TEST_PORT}/api")
	if result != "http://localhost:8080/api" {
		t.Errorf("expected http://localhost:8080/api, got %s", result)
	}
}

func TestBuildEnvValueMixedLiteral(t *testing.T) {
	t.Setenv("TEST_MIX_TOKEN", "my-token")

	ev := buildEnvValue("Bearer ${TEST_MIX_TOKEN}")
	if ev.Template != "Bearer ${TEST_MIX_TOKEN}" {
		t.Errorf("expected template, got %q", ev.Template)
	}
	if ev.Value != "Bearer my-token" {
		t.Errorf("expected 'Bearer my-token', got %q", ev.Value)
	}
	if ev.EnvVar != "" {
		t.Errorf("expected empty envVar for template, got %q", ev.EnvVar)
	}
}

func TestBuildEnvValueMultipleVars(t *testing.T) {
	t.Setenv("TEST_MV_HOST", "localhost")
	t.Setenv("TEST_MV_PORT", "8080")

	ev := buildEnvValue("http://${TEST_MV_HOST}:${TEST_MV_PORT}/api")
	if ev.Template != "http://${TEST_MV_HOST}:${TEST_MV_PORT}/api" {
		t.Errorf("expected template, got %q", ev.Template)
	}
	if ev.Value != "http://localhost:8080/api" {
		t.Errorf("expected resolved value, got %q", ev.Value)
	}
}

func TestBuildCompiledConfigRuntimeOverride(t *testing.T) {
	// Simulate: at compile time TOKEN=compile-val
	t.Setenv("TEST_OVERRIDE_TOKEN", "compile-val")

	cfg := &Config{
		Servers: map[string]ServerConfig{
			"s1": {
				Command: "node",
				Env: map[string]string{
					"KEY": "${TEST_OVERRIDE_TOKEN}",
				},
			},
		},
	}

	// Simulate real compile flow
	cfg.ResolveEnvVars()
	cc := BuildCompiledConfig(cfg)

	// Verify env var name is captured even though the value was resolved
	if cc.Servers["s1"].Env["KEY"].EnvVar != "TEST_OVERRIDE_TOKEN" {
		t.Errorf("expected TEST_OVERRIDE_TOKEN, got %s", cc.Servers["s1"].Env["KEY"].EnvVar)
	}
	if cc.Servers["s1"].Env["KEY"].Value != "compile-val" {
		t.Errorf("expected compile-val, got %s", cc.Servers["s1"].Env["KEY"].Value)
	}
}

func TestBuildCompiledConfigMixedLiteralPreservesPrefix(t *testing.T) {
	t.Setenv("TEST_MLP_TOKEN", "abc123")

	cfg := &Config{
		Servers: map[string]ServerConfig{
			"s1": {
				Command: "node",
				Headers: map[string]string{
					"Auth": "Bearer ${TEST_MLP_TOKEN}",
				},
			},
		},
	}

	cfg.ResolveEnvVars()
	cc := BuildCompiledConfig(cfg)

	ev := cc.Servers["s1"].Headers["Auth"]
	if ev.Template != "Bearer ${TEST_MLP_TOKEN}" {
		t.Errorf("expected template 'Bearer ${TEST_MLP_TOKEN}', got %q", ev.Template)
	}
	if ev.Value != "Bearer abc123" {
		t.Errorf("expected 'Bearer abc123', got %q", ev.Value)
	}
}

func TestResolveEnvValueTemplate(t *testing.T) {
	t.Setenv("TEST_RET_TOKEN", "runtime-val")

	ev := EnvValue{
		Value:    "Bearer compile-val",
		Template: "Bearer ${TEST_RET_TOKEN}",
	}
	result := resolveEnvValue(ev)
	if result != "Bearer runtime-val" {
		t.Errorf("expected 'Bearer runtime-val', got %q", result)
	}
}

func TestResolveEnvValueTemplateFallback(t *testing.T) {
	_ = os.Unsetenv("TEST_RETF_UNSET")

	ev := EnvValue{
		Value:    "Bearer compile-val",
		Template: "Bearer ${TEST_RETF_UNSET}",
	}
	result := resolveEnvValue(ev)
	if result != "Bearer compile-val" {
		t.Errorf("expected 'Bearer compile-val', got %q", result)
	}
}

func TestResolveEnvValueMultiVarTemplatePartialSet(t *testing.T) {
	t.Setenv("TEST_RMVT_HOST", "newhost")
	_ = os.Unsetenv("TEST_RMVT_PORT")

	ev := EnvValue{
		Value:    "http://oldhost:8080/api",
		Template: "http://${TEST_RMVT_HOST}:${TEST_RMVT_PORT}/api",
	}
	// When not all vars are set, falls back to compile-time value
	result := resolveEnvValue(ev)
	if result != "http://oldhost:8080/api" {
		t.Errorf("expected compile-time fallback, got %q", result)
	}
}
