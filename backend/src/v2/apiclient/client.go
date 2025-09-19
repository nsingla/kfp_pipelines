package apiclient

import (
	"fmt"
	"os"
	"time"

	gc "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"google.golang.org/grpc"
)

// Client provides typed clients for KFP v2beta1 API services used by driver/launcher.
type Client struct {
	Run       gc.RunServiceClient
	Artifact  gc.ArtifactServiceClient
	Conn      *grpc.ClientConn
	Endpoint  string
}

// Config holds connection options.
type Config struct {
	// Endpoint in host:port form, e.g. ml-pipeline.kubeflow:8887
	Endpoint string
}

// FromEnv builds a Config from environment with sensible defaults.
// KFP_API_ADDRESS and KFP_API_PORT are used; default is ml-pipeline.kubeflow:8887.
func FromEnv() *Config {
	addr := os.Getenv("KFP_API_ADDRESS")
	port := os.Getenv("KFP_API_PORT")
	endpoint := "ml-pipeline.kubeflow:8887"
	if addr != "" && port != "" {
		endpoint = fmt.Sprintf("%s:%s", addr, port)
	}
	return &Config{Endpoint: endpoint}
}

// New creates a new API client connection.
func New(cfg *Config) (*Client, error) {
	if cfg == nil || cfg.Endpoint == "" {
		return nil, fmt.Errorf("invalid config: missing endpoint")
	}
	conn, err := grpc.Dial(cfg.Endpoint,
		grpc.WithInsecure(), // In-cluster traffic; mTLS/secure options can be added later
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to KFP API at %s: %w", cfg.Endpoint, err)
	}
	return &Client{
		Run:      gc.NewRunServiceClient(conn),
		Artifact: gc.NewArtifactServiceClient(conn),
		Conn:     conn,
		Endpoint: cfg.Endpoint,
	}, nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	if c == nil || c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}
