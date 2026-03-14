package server

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/priyanshujain/openbotkit/internal/servertest"
	"github.com/priyanshujain/openbotkit/internal/testutil"
	"github.com/priyanshujain/openbotkit/remote"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestServerAPI_Docker runs the server API contract tests against a Docker
// container built from the repo's Dockerfile. Requires Docker to be running.
func TestServerAPI_Docker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Docker test in short mode")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("skipping: Docker is not running")
	}

	testutil.LoadEnv(t)
	geminiKey := testutil.RequireGeminiKey(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// The server requires a config with models configured.
	configYAML := `models:
  default: gemini/gemini-2.5-flash
  providers:
    gemini:
      provider: gemini
`

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "../../",
			Dockerfile: "infrastructure/docker/Dockerfile",
		},
		ExposedPorts: []string{"8443/tcp"},
		Env: map[string]string{
			"OBK_CONFIG_DIR":    "/data",
			"OBK_AUTH_USERNAME": "testuser",
			"OBK_AUTH_PASSWORD": "testpass",
			"GEMINI_API_KEY":    geminiKey,
		},
		Files: []testcontainers.ContainerFile{
			{
				Reader:            strings.NewReader(configYAML),
				ContainerFilePath: "/data/config.yaml",
				FileMode:          0644,
			},
		},
		Cmd: []string{"server", "run", "--addr", ":8443"},
		WaitingFor: wait.ForHTTP("/api/health").
			WithPort("8443/tcp").
			WithStartupTimeout(3 * time.Minute),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start container: %v", err)
	}
	defer container.Terminate(ctx)

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	port, err := container.MappedPort(ctx, "8443")
	if err != nil {
		t.Fatalf("get port: %v", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	b := servertest.Backend{
		Client:       remote.NewClient(baseURL, "testuser", "testpass"),
		NoAuthClient: remote.NewClient(baseURL, "", ""),
	}
	servertest.Run(t, b)
}
