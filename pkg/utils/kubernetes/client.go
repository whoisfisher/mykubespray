package kubernetes

import (
	"fmt"
	"k8s.io/client-go/informers"
	"os"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// K8sClient wraps the various Kubernetes clients
type K8sClient struct {
	Clientset       *kubernetes.Clientset
	DynamicClient   dynamic.Interface
	DiscoveryClient *discovery.DiscoveryClient
	InformerFactory informers.SharedInformerFactory
}

type K8sConfig struct {
	Kubeconfig string
	ApiServer  string
	Token      string
	Cacert     string
}

// NewK8sClient creates a new K8sClient based on the provided configuration.
func NewK8sClient(config K8sConfig) (*K8sClient, error) {
	var cfg *rest.Config
	var err error

	// Determine which configuration to use
	if config.Kubeconfig != "" {
		// Use kubeconfig file
		cfg, err = clientcmd.BuildConfigFromFlags("", config.Kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build kubeconfig: %v", err)
		}
	} else if config.ApiServer != "" && config.Token != "" {
		// Use API server URL and token
		cfg = &rest.Config{
			Host:        config.ApiServer,
			BearerToken: config.Token,
			TLSClientConfig: rest.TLSClientConfig{
				CAFile: config.Cacert,
			},
		}
	} else {
		// Use in-cluster config
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %v", err)
	}

	informerFactory := informers.NewSharedInformerFactory(clientset, 0)

	return &K8sClient{
		Clientset:       clientset,
		DynamicClient:   dynamicClient,
		DiscoveryClient: discoveryClient,
		InformerFactory: informerFactory,
	}, nil
}

// NewDefaultClient creates a new K8sClient using the default kubeconfig file.
func NewDefaultClient() (*K8sClient, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" && homedir.HomeDir() != "" {
		kubeconfig = fmt.Sprintf("%s/.kube/config", homedir.HomeDir())
	}
	config := K8sConfig{
		Kubeconfig: kubeconfig,
	}
	return NewK8sClient(config)
}
