package core

import (
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type K8sClient struct {
	config *rest.Config
}

// NewK8sClientFromClusterConfig 根据集群配置创建K8s客户端
func NewK8sClientFromClusterConfig(apiServer, caCert, clientCert, clientKey, token string, skipVerify bool) (*kubernetes.Clientset, error) {
	config := NewK8sRestConfig(apiServer, caCert, clientCert, clientKey, token, skipVerify)
	return kubernetes.NewForConfig(config)
}

// NewK8sRestConfig 根据集群配置构建 *rest.Config（供 exec/logs 等场景复用）
func NewK8sRestConfig(apiServer, caCert, clientCert, clientKey, token string, skipVerify bool) *rest.Config {
	var config *rest.Config

	if token != "" {
		config = &rest.Config{
			Host:        apiServer,
			BearerToken: token,
			TLSClientConfig: rest.TLSClientConfig{
				CAData:   []byte(caCert),
				CertData: []byte(clientCert),
				KeyData:  []byte(clientKey),
			},
		}
	} else {
		config = &rest.Config{
			Host: apiServer,
			TLSClientConfig: rest.TLSClientConfig{
				CAData:   []byte(caCert),
				CertData: []byte(clientCert),
				KeyData:  []byte(clientKey),
			},
		}
	}

	if skipVerify {
		config.TLSClientConfig.Insecure = true
	}

	return config
}

// NewK8sClientFromKubeConfig 从kubeconfig文件创建K8s客户端
func NewK8sClientFromKubeConfig(kubeconfigPath string) (*kubernetes.Clientset, error) {
	if kubeconfigPath == "" {
		// 尝试使用默认路径
		homeDir := homedir.HomeDir()
		if homeDir != "" {
			kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
		}
	}

	// 构建配置加载规则
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfigPath

	// 获取客户端配置
	configOverrides := &clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	)

	// 获取REST配置
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}
