package client_manager

import (
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"k8s.io/client-go/kubernetes"
)

type FakeClientManager struct {
	k8sClient      kubernetes.Interface
	metadataClient metadata.ClientInterface
}

// Ensure FakeClientManager implements ClientManagerInterface
var _ ClientManagerInterface = (*FakeClientManager)(nil)

func (f *FakeClientManager) K8sClient() kubernetes.Interface {
	return f.k8sClient
}

func (f *FakeClientManager) MetadataClient() metadata.ClientInterface {
	return f.metadataClient
}

func NewFakeClientManager(k8sClient kubernetes.Interface, metadataClient metadata.ClientInterface) *FakeClientManager {
	return &FakeClientManager{
		k8sClient:      k8sClient,
		metadataClient: metadataClient,
	}
}
