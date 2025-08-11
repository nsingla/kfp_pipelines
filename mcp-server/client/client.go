package client

import (
	"github.com/kubeflow/model-registry/pkg/openapi"
	"net/http"
	"sync"
)

// OpenAPIClientSingleton manages a singleton instance of the OpenAPI client
type OpenAPIClientSingleton struct {
	client     *openapi.APIClient
	httpClient *http.Client
	mu         sync.RWMutex
}

var (
	instance *OpenAPIClientSingleton
	initOnce sync.Once
)

// GetInstance returns the singleton instance of OpenAPIClientSingleton
func GetInstance() *OpenAPIClientSingleton {
	initOnce.Do(func() {
		instance = &OpenAPIClientSingleton{
			httpClient: http.DefaultClient,
		}
	})
	return instance
}

// SetHTTPClient sets the HTTP client to use for API calls
func (s *OpenAPIClientSingleton) SetHTTPClient(client *http.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.httpClient = client
	// Reset the client so it gets recreated with the new HTTP client
	s.client = nil
}

// GetClient returns an OpenAPI client configured for the specified URL
// If the URL changes, a new client instance will be created
func (s *OpenAPIClientSingleton) GetClient(url string) *openapi.APIClient {
	s.mu.RLock()
	// Check if we have a client and if the URL matches
	if s.client != nil {
		currentURL := s.getCurrentURL(s.client)
		if currentURL == url {
			defer s.mu.RUnlock()
			return s.client
		}
	}
	s.mu.RUnlock()

	// Need to create/recreate client
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check pattern - another goroutine might have created it
	if s.client != nil {
		currentURL := s.getCurrentURL(s.client)
		if currentURL == url {
			return s.client
		}
	}

	// Create new client with the specified URL
	cfg := &openapi.Configuration{
		HTTPClient: s.httpClient,
		Servers: openapi.ServerConfigurations{
			{
				URL: url,
			},
		},
	}

	s.client = openapi.NewAPIClient(cfg)
	return s.client
}

// getCurrentURL extracts the current URL from the client configuration
func (s *OpenAPIClientSingleton) getCurrentURL(client *openapi.APIClient) string {
	if client == nil || client.GetConfig() == nil || len(client.GetConfig().Servers) == 0 {
		return ""
	}
	return client.GetConfig().Servers[0].URL
}

// ResetClient forces recreation of the client on next GetClient call
func (s *OpenAPIClientSingleton) ResetClient() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = nil
}
