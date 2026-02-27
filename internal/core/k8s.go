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
	var config *rest.Config

	if token != "" {
		// 使用Token认证
		config = &rest.Config{
			Host: apiServer,
			TLSClientConfig: rest.TLSClientConfig{
				CAData:   []byte(caCert),
				CertData: []byte(clientCert),
				KeyData:  []byte(clientKey),
			},
			BearerToken: token,
		}
	} else {
		// 使用证书认证
		config = &rest.Config{
			Host: apiServer,
			TLSClientConfig: rest.TLSClientConfig{
				CAData:   []byte(caCert),
				CertData: []byte(clientCert),
				KeyData:  []byte(clientKey),
			},
		}
	}

	// 根据SkipVerify字段决定是否跳过证书验证
	if skipVerify {
		config.TLSClientConfig.Insecure = true
	}

	return kubernetes.NewForConfig(config)
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
