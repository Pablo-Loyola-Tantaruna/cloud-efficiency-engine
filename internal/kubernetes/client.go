package kubernetes

import (
	"fmt"
	"os"
	"strings"

	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var ErrNotConfigured = fmt.Errorf("kubernetes client is not configured")

// NewClientFromEnv builds a Kubernetes client from an explicit kubeconfig or
// the in-cluster service account. It deliberately does not fall back to a
// developer's default kubeconfig path so production startup is deterministic.
func NewClientFromEnv() (kubeclient.Interface, error) {
	kubeconfig := strings.TrimSpace(os.Getenv("FINOPS_KUBECONFIG"))
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else if strings.TrimSpace(os.Getenv("KUBERNETES_SERVICE_HOST")) != "" {
		config, err = rest.InClusterConfig()
	} else {
		return nil, ErrNotConfigured
	}
	if err != nil {
		return nil, fmt.Errorf("build Kubernetes config: %w", err)
	}

	client, err := kubeclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create Kubernetes client: %w", err)
	}
	return client, nil
}
