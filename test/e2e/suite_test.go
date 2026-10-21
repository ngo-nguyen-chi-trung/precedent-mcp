//go:build e2e

// Test plan: e2e suite context
//
// The suite is a client of the wire contract: it imports nothing from the module and speaks JSON maps only.
//
// Logic flow
// | Function   | Step | Action                                                                                        | Effect                                                 |
// |------------|------|-----------------------------------------------------------------------------------------------|--------------------------------------------------------|
// | SetupSuite | 1    | Build ./cmd/server and ./cmd/gateway into a temporary directory                               | The binaries under test, never ./bin/server            |
// | SetupSuite | 2    | Start the server with GRPC_PORT on a free port                                                | gRPC services backed by Weaviate and assets            |
// | SetupSuite | 3    | Start the gateway with GRPC_ADDRESS and HTTP_PORT on free ports                               | Every request in the suite goes through the gateway    |
// | SetupSuite | 4    | Post server/discover until HTTP 200 or a process exits                                        | Both processes are ready; a crash fails the suite      |
// | SetupSuite | 5    | Compile every outputSchema of the decoded tools/list with format assertion                    | Each test validates structuredContent against its tool |
// | SetupTest  | 1    | Generate a topic word unique to the test                                                      | Searches match only what the test stored               |
// | Each test  | 1    | Marshal the request map; post it with the header through s.client; decode the body into a map | HTTP status and response map compared whole            |

package e2e

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/suite"
)

const (
	readyTimeout   = 60 * time.Second
	readyTick      = 200 * time.Millisecond
	stopTimeout    = 5 * time.Second
	requestTimeout = 30 * time.Second
	searchTimeout  = 10 * time.Second
	searchTick     = 200 * time.Millisecond
)

var meta = map[string]any{
	"io.modelcontextprotocol/protocolVersion":    "2026-07-28",
	"io.modelcontextprotocol/clientCapabilities": map[string]any{},
}

var serverMeta = map[string]any{
	"io.modelcontextprotocol/serverInfo": map[string]any{"name": "precedent-mcp", "version": "1.0.0"},
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type E2ESuite struct {
	suite.Suite
	topic   string
	url     string
	client  *http.Client
	exited  []chan struct{}
	logs    syncBuffer
	schemas map[string]*jsonschema.Schema
}

func TestE2E(t *testing.T) {
	suite.Run(t, new(E2ESuite))
}

func (s *E2ESuite) SetupSuite() {
	t := s.T()
	t.Cleanup(func() {
		if t.Failed() {
			t.Log(s.logs.String())
		}
	})

	root, err := filepath.Abs(filepath.Join("..", ".."))
	s.Require().NoError(err)
	s.client = &http.Client{Timeout: requestTimeout}

	ports := s.freePorts(2)
	grpcPort, httpPort := ports[0], ports[1]
	s.url = "http://127.0.0.1:" + httpPort + "/mcp"

	s.start(root, "server", "GRPC_PORT="+grpcPort)
	s.start(root, "gateway", "GRPC_ADDRESS=127.0.0.1:"+grpcPort, "HTTP_PORT="+httpPort, "GIN_MODE=release")

	s.Require().Eventually(s.ready, readyTimeout, readyTick)
	s.Require().False(s.hasExited())

	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": map[string]any{"_meta": meta}})
	s.Require().NoError(err)
	req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
	s.Require().NoError(err)
	req.Header = http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"tools/list"}}
	res, err := s.client.Do(req)
	s.Require().NoError(err)
	defer func() { s.NoError(res.Body.Close()) }()
	s.Require().Equal(http.StatusOK, res.StatusCode)
	var got map[string]any
	s.Require().NoError(json.NewDecoder(res.Body).Decode(&got))
	s.schemas = map[string]*jsonschema.Schema{}
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	for _, tool := range got["result"].(map[string]any)["tools"].([]any) {
		name := tool.(map[string]any)["name"].(string)
		s.Require().NoError(c.AddResource(name+".json", tool.(map[string]any)["outputSchema"]), name)
		schema, err := c.Compile(name + ".json")
		s.Require().NoError(err, name)
		s.schemas[name] = schema
	}
}

func (s *E2ESuite) SetupTest() {
	s.topic = "e2e" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

func (s *E2ESuite) freePorts(count int) []string {
	ports := make([]string, count)
	for i := range ports {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		s.Require().NoError(err)
		defer func() { s.NoError(listener.Close()) }()
		ports[i] = strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	}
	return ports
}

func (s *E2ESuite) start(root, name string, environ ...string) {
	bin := filepath.Join(s.T().TempDir(), name)
	args := []string{"build"}
	if os.Getenv("E2E_COVERDIR") != "" {
		args = append(args, "-cover", "-coverpkg=./...")
	}
	build := exec.Command("go", append(args, "-o", bin, "./cmd/"+name)...)
	build.Dir = root
	build.Stdout, build.Stderr = &s.logs, &s.logs
	s.Require().NoError(build.Run(), name)

	cmd := exec.Command(bin)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), environ...)
	if dir := os.Getenv("E2E_COVERDIR"); dir != "" {
		cmd.Env = append(cmd.Env, "GOCOVERDIR="+dir)
	}
	cmd.Stdout, cmd.Stderr = &s.logs, &s.logs
	s.Require().NoError(cmd.Start(), name)
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()
	s.exited = append(s.exited, exited)
	s.T().Cleanup(func() {
		_ = cmd.Process.Signal(os.Interrupt)
		select {
		case <-exited:
		case <-time.After(stopTimeout):
			_ = cmd.Process.Kill()
			<-exited
		}
	})
}

func (s *E2ESuite) ready() bool {
	if s.hasExited() {
		return true
	}
	payload, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "server/discover", "params": map[string]any{"_meta": meta}})
	if err != nil {
		return false
	}
	req, err := http.NewRequest(http.MethodPost, s.url, bytes.NewReader(payload))
	if err != nil {
		return false
	}
	req.Header = http.Header{"Content-Type": {"application/json"}, "Mcp-Protocol-Version": {"2026-07-28"}, "Mcp-Method": {"server/discover"}}
	res, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer func() { s.NoError(res.Body.Close()) }()
	return res.StatusCode == http.StatusOK
}

func (s *E2ESuite) hasExited() bool {
	for _, exited := range s.exited {
		select {
		case <-exited:
			return true
		default:
		}
	}
	return false
}
