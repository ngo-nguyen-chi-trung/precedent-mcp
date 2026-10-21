// Test plan: env.go
//
// Logic flow
// | Function          | Step | Action                                                 | Effect                                                                                  |
// |-------------------|------|--------------------------------------------------------|-----------------------------------------------------------------------------------------|
// | EnvKey.GetValue   | 1    | Read the process environment variable named by the key | Unset: ""                                                                               |
// | GetAllowedOrigins | 1    | ALLOWED_ORIGINS empty → OriginDefault                  | Loopback origins on any port                                                            |
// | GetAllowedOrigins | 2    | Split on OriginSeparator "," into origins              | Entries kept exactly as given; an invalid list is rejected at gateway startup           |
// | LoadEnv           | 1    | Load .env from the working directory                   | Missing or malformed file: error; cmd/server and cmd/gateway log a warning and go on    |
// | LoadEnv           | 2    | Set each key that is not already in the environment    | Existing variables keep their values                                                    |
//
// Scenarios
// | Number | Function          | Result  | Value class    | Input                                          | Expected                                                        |
// |--------|-------------------|---------|----------------|------------------------------------------------|-----------------------------------------------------------------|
// | 1      | EnvKey.GetValue   | Success | Middle value   | GRPC_PORT=50051                                | "50051" (step 1)                                                |
// | 2      | EnvKey.GetValue   | Success | Boundary value | GRPC_PORT unset                                | "" (step 1)                                                     |
// | 3      | GetAllowedOrigins | Success | Boundary value | ALLOWED_ORIGINS unset                          | http://localhost:*, http://127.0.0.1:*, http://[::1]:* (step 1) |
// | 4      | GetAllowedOrigins | Success | Boundary value | ALLOWED_ORIGINS=https://example.com            | One origin (step 2)                                             |
// | 5      | GetAllowedOrigins | Success | Middle value   | ALLOWED_ORIGINS=https://a.com,https://b.com    | Both origins (step 2)                                           |
// | 6      | GetAllowedOrigins | Success | Middle value   | ALLOWED_ORIGINS=https://a.com, https://b.com   | "https://a.com", " https://b.com"; the space is kept (step 2)   |
// | 7      | GetAllowedOrigins | Success | Boundary value | ALLOWED_ORIGINS=https://a.com,https://b.com,   | "https://a.com", "https://b.com", "" (step 2)                   |
// | 8      | LoadEnv           | Success | Middle value   | .env holds HTTP_PORT=9999; HTTP_PORT unset     | HTTP_PORT becomes "9999" (steps 1, 2)                           |
// | 9      | LoadEnv           | Success | Boundary value | .env holds HTTP_PORT=9999; HTTP_PORT=8080 set  | HTTP_PORT stays "8080" (step 2)                                 |
// | 10     | LoadEnv           | Failure | Middle value   | No .env in the working directory               | Error "open .env: no such file or directory" (step 1)           |
// | 11     | LoadEnv           | Failure | Middle value   | .env holds the malformed line "HTTP PORT 9999" | godotenv parse error (step 1)                                   |
//
// Outside env.go: a bad environment value is rejected, never repaired; cmd/gateway stops right after LoadEnv when gateway.CORSConfig().Validate fails, and gateway/rest_test.go covers CORSConfig.
//
// Branch coverage
// | Function          | Condition                            | True | False  |
// |-------------------|--------------------------------------|------|--------|
// | GetAllowedOrigins | ALLOWED_ORIGINS empty                | 3    | 4 to 7 |
// | GetAllowedOrigins | An entry carries a space or is empty | 6, 7 | 4, 5   |
// | LoadEnv           | Variable already set                 | 9    | 8      |
// | LoadEnv           | .env readable and valid              | 8, 9 | 10, 11 |
//
// Test mapping
// | Test                                                      | Scenarios |
// |-----------------------------------------------------------|-----------|
// | TestEnvKeyGetValue/set variable returns its value         | 1         |
// | TestEnvKeyGetValue/unset variable returns empty           | 2         |
// | TestGetAllowedOrigins/unset value yields the defaults     | 3         |
// | TestGetAllowedOrigins/single origin                       | 4         |
// | TestGetAllowedOrigins/multiple origins                    | 5         |
// | TestGetAllowedOrigins/space after a comma is kept         | 6         |
// | TestGetAllowedOrigins/trailing comma keeps an empty entry | 7         |
// | TestLoadEnv/sets an unset variable                        | 8         |
// | TestLoadEnv/keeps an existing variable                    | 9         |
// | TestLoadEnvErrors/missing file                            | 10        |
// | TestLoadEnvErrors/malformed line                          | 11        |
// | No test                                                   | None      |

package env_test

import . "github.com/ngo/precedent_mcp/internal/env"

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvKeyGetValue(t *testing.T) {
	tests := []struct {
		name             string
		givenEnvironment map[EnvKey]string
		want             string
	}{
		{"set variable returns its value", map[EnvKey]string{GRPCPort: "50051"}, "50051"},
		{"unset variable returns empty", map[EnvKey]string{}, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(string(GRPCPort), "")
			require.NoError(t, os.Unsetenv(string(GRPCPort)))
			for key, value := range test.givenEnvironment {
				t.Setenv(string(key), value)
			}

			got := GRPCPort.GetValue()

			assert.Equal(t, test.want, got)
		})
	}
}

func TestGetAllowedOrigins(t *testing.T) {
	tests := []struct {
		name             string
		givenEnvironment map[EnvKey]string
		want             []string
	}{
		{"unset value yields the defaults", map[EnvKey]string{},
			[]string{"http://localhost:*", "http://127.0.0.1:*", "http://[::1]:*"}},
		{"single origin", map[EnvKey]string{AllowedOrigins: "https://example.com"},
			[]string{"https://example.com"}},
		{"multiple origins", map[EnvKey]string{AllowedOrigins: "https://a.com,https://b.com"},
			[]string{"https://a.com", "https://b.com"}},
		{"space after a comma is kept", map[EnvKey]string{AllowedOrigins: "https://a.com, https://b.com"},
			[]string{"https://a.com", " https://b.com"}},
		{"trailing comma keeps an empty entry", map[EnvKey]string{AllowedOrigins: "https://a.com,https://b.com,"},
			[]string{"https://a.com", "https://b.com", ""}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(string(AllowedOrigins), "")
			require.NoError(t, os.Unsetenv(string(AllowedOrigins)))
			for key, value := range test.givenEnvironment {
				t.Setenv(string(key), value)
			}

			got := GetAllowedOrigins()

			assert.Equal(t, test.want, got)
		})
	}
}

func TestLoadEnv(t *testing.T) {
	tests := []struct {
		name             string
		givenFiles       map[string]string
		givenEnvironment map[EnvKey]string
		want             string
	}{
		{"sets an unset variable", map[string]string{".env": "HTTP_PORT=9999\n"}, map[EnvKey]string{}, "9999"},
		{"keeps an existing variable", map[string]string{".env": "HTTP_PORT=9999\n"}, map[EnvKey]string{HTTPPort: "8080"}, "8080"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			for file, content := range test.givenFiles {
				require.NoError(t, os.WriteFile(filepath.Join(directory, file), []byte(content), 0o600))
			}
			t.Chdir(directory)
			t.Setenv(string(HTTPPort), "")
			require.NoError(t, os.Unsetenv(string(HTTPPort)))
			for key, value := range test.givenEnvironment {
				t.Setenv(string(key), value)
			}

			err := LoadEnv()

			require.NoError(t, err)
			got := HTTPPort.GetValue()
			assert.Equal(t, test.want, got)
		})
	}
}

func TestLoadEnvErrors(t *testing.T) {
	tests := []struct {
		name       string
		givenFiles map[string]string
		wantError  string
	}{
		{"missing file", map[string]string{}, "open .env: no such file or directory"},
		{"malformed line", map[string]string{".env": "HTTP PORT 9999\n"},
			`unexpected character "\n" in variable name near "HTTP PORT 9999\n"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			for file, content := range test.givenFiles {
				require.NoError(t, os.WriteFile(filepath.Join(directory, file), []byte(content), 0o600))
			}
			t.Chdir(directory)

			err := LoadEnv()

			assert.EqualError(t, err, test.wantError)
		})
	}
}
