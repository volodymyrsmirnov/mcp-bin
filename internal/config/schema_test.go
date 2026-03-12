package config

import "testing"

func TestValidateAgainstSchema(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		isYAML  bool
		wantErr bool
	}{
		// Valid JSON configs
		{
			name: "valid JSON local server",
			data: `{"servers": {"s1": {"command": "node"}}}`,
		},
		{
			name: "valid JSON remote server",
			data: `{"servers": {"s1": {"url": "https://example.com"}}}`,
		},
		{
			name: "valid JSON with files",
			data: `{"files": ["/tmp/dir"], "servers": {"s1": {"command": "node"}}}`,
		},
		{
			name: "valid JSON all server fields",
			data: `{"servers": {"s1": {"command": "node", "args": ["x"], "env": {"K": "V"}, "cwd": "/tmp", "allow_tools": ["*"], "deny_tools": ["bad"]}}}`,
		},
		{
			name: "valid JSON remote with headers",
			data: `{"servers": {"s1": {"url": "https://example.com", "headers": {"Authorization": "Bearer token"}}}}`,
		},
		{
			name: "valid JSON empty servers map",
			data: `{"servers": {}}`,
		},
		{
			name: "valid JSON multiple servers",
			data: `{"servers": {"local": {"command": "node"}, "remote": {"url": "https://example.com"}}}`,
		},

		// Valid YAML configs
		{
			name:   "valid YAML local server",
			data:   "servers:\n  s1:\n    command: node\n",
			isYAML: true,
		},
		{
			name:   "valid YAML remote server",
			data:   "servers:\n  s1:\n    url: https://example.com\n",
			isYAML: true,
		},

		// Missing servers
		{
			name:    "JSON missing servers",
			data:    `{"files": []}`,
			wantErr: true,
		},
		{
			name:    "YAML missing servers",
			data:    "files:\n  - /tmp\n",
			isYAML:  true,
			wantErr: true,
		},

		// Both command and url
		{
			name:    "JSON both command and url",
			data:    `{"servers": {"s1": {"command": "node", "url": "https://x.com"}}}`,
			wantErr: true,
		},
		{
			name:    "YAML both command and url",
			data:    "servers:\n  s1:\n    command: node\n    url: https://x.com\n",
			isYAML:  true,
			wantErr: true,
		},

		// Neither command nor url
		{
			name:    "JSON neither command nor url",
			data:    `{"servers": {"s1": {"args": ["x"]}}}`,
			wantErr: true,
		},

		// Unknown top-level field
		{
			name:    "JSON unknown top-level field",
			data:    `{"servers": {"s1": {"command": "node"}}, "extra": true}`,
			wantErr: true,
		},
		{
			name:    "YAML unknown top-level field",
			data:    "servers:\n  s1:\n    command: node\ndirectories:\n  - /tmp\n",
			isYAML:  true,
			wantErr: true,
		},

		// Unknown server field
		{
			name:    "JSON unknown server field",
			data:    `{"servers": {"s1": {"command": "node", "timeout": 30}}}`,
			wantErr: true,
		},
		{
			name:    "YAML unknown server field",
			data:    "servers:\n  s1:\n    command: node\n    timeout: 30\n",
			isYAML:  true,
			wantErr: true,
		},

		// Wrong types
		{
			name:    "JSON args wrong type",
			data:    `{"servers": {"s1": {"command": "node", "args": "not-array"}}}`,
			wantErr: true,
		},
		{
			name:    "JSON env wrong value type",
			data:    `{"servers": {"s1": {"command": "node", "env": {"K": 123}}}}`,
			wantErr: true,
		},
		{
			name:    "JSON files wrong type",
			data:    `{"files": "not-array", "servers": {"s1": {"command": "node"}}}`,
			wantErr: true,
		},
		{
			name:    "JSON servers wrong type",
			data:    `{"servers": ["not", "object"]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgainstSchema([]byte(tt.data), tt.isYAML)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAgainstSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
