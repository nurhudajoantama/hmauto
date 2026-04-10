package config

import "testing"

func TestSecurityValidateAuthTokens(t *testing.T) {
	tests := []struct {
		name     string
		security Security
		wantErr  string
	}{
		{
			name: "valid tokens",
			security: Security{
				BearerToken: "http-token",
				MCPToken:    "mcp-token",
			},
		},
		{
			name: "missing bearer token",
			security: Security{
				MCPToken: "mcp-token",
			},
			wantErr: "security.bearerToken must be set",
		},
		{
			name: "missing mcp token",
			security: Security{
				BearerToken: "http-token",
			},
			wantErr: "security.mcpToken must be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.security.ValidateAuthTokens()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateAuthTokens() error = %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("ValidateAuthTokens() error = nil, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("ValidateAuthTokens() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}
