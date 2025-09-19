package client_manager

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ClientManagerInterface interface {
	K8sClient() kubernetes.Interface
}

// Ensure ClientManager implements ClientManagerInterface
var _ ClientManagerInterface = (*ClientManager)(nil)

// ClientManager is a container for various service clients.
type ClientManager struct {
	k8sClient kubernetes.Interface
}

type Options struct {
	MLMDServerAddress string
	MLMDServerPort    string
	CacheDisabled     bool
}

// NewClientManager creates and Init a new instance of ClientManager.
func NewClientManager(options *Options) (*ClientManager, error) {
	clientManager := &ClientManager{}
	err := clientManager.init(options)
	if err != nil {
		return nil, err
	}

	return clientManager, nil
}

func (cm *ClientManager) K8sClient() kubernetes.Interface {
	return cm.k8sClient
}

func (cm *ClientManager) init(opts *Options) error {
	k8sClient, err := initK8sClient()
	if err != nil {
		return err
	}
	cm.k8sClient = k8sClient
	return nil
}

func initK8sClient() (kubernetes.Interface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
	}
	return k8sClient, nil
}

