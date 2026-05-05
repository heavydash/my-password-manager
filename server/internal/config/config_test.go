package config

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestConfig_New(t *testing.T) {
	t.Run("default_config", func(t *testing.T) {
		cfg, err := newWithFlags(flag.NewFlagSet("test", flag.ContinueOnError))
		assert.NoError(t, err)
		assert.Equal(t, "8080", cfg.Server.Port)
	})

	t.Run("env_override", func(t *testing.T) {
		os.Setenv("SERVER_PORT", "3000")
		os.Setenv("JWT_SECRET", "my-super-secret-jwt-key-32-chars-minimum")
		defer os.Clearenv()

		cfg, err := newWithFlags(flag.NewFlagSet("test", flag.ContinueOnError))
		assert.NoError(t, err)
		assert.Equal(t, "3000", cfg.Server.Port)
	})
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "valid config",
			cfg: &Config{
				JWT: JWTConfig{Secret: "very-long-secret-key-at-least-32-chars"},
				Database: DatabaseConfig{
					DSN:      "postgres://user:pass@localhost/db",
					MaxConns: 25,
					MinConns: 5,
				},
				Server: ServerConfig{Port: "8080",
					GRPCPort: "9090",
					Env:      "development"},
			},
		},
		{
			name: "short jwt secret",
			cfg: &Config{
				JWT:      JWTConfig{Secret: "short"},
				Database: DatabaseConfig{DSN: "postgres://..."},
				Server:   ServerConfig{Port: "8080", GRPCPort: "9090", Env: "development"},
			},
			wantErr: "JWT_SECRET must be at least 32 characters long",
		},
		{
			name: "missing DB_DSN",
			cfg: &Config{
				JWT:    JWTConfig{Secret: "very-long-secret-key-at-least-32-chars"},
				Server: ServerConfig{Port: "8080", GRPCPort: "9090", Env: "development"},
			},
			wantErr: "DB_DSN is required",
		},
		{
			name: "production without OAuth",
			cfg: &Config{
				JWT: JWTConfig{Secret: "very-long-secret-key-at-least-32-chars"},
				Database: DatabaseConfig{
					DSN:      "postgres://user:pass@localhost/db",
					MaxConns: 25,
					MinConns: 5,
				},
				Server: ServerConfig{
					Port:     "8080",
					GRPCPort: "9090",
					Env:      "production",
				},
				OAuth: OAuth{},
			},
			wantErr: "OAuth credentials required in production",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
