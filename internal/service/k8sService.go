package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ZebraOps/ZebraCICD/internal/core"
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8SService struct {
	clusterRepo *handler.K8SClusterRepository
}

func NewK8SService(clusterRepo *handler.K8SClusterRepository) *K8SService {
	return &K8SService{
		clusterRepo: clusterRepo,
	}
}

// CreateCluster 创建K8s集群凭证
func (s *K8SService) CreateCluster(cluster *model.K8SCluster) error {
	return s.clusterRepo.Create(cluster)
}

// TestConnection 测试连接K8s集群
func (s *K8SService) TestConnection(clusterID uint) error {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return err
	}

	// 创建K8s客户端
	clientset, err := s.createK8sClient(cluster)
	if err != nil {
		return err
	}

	// 尝试获取节点列表以测试连接
	_, err = clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	return err
}

// ListPods 获取Pod列表
func (s *K8SService) ListPods(clusterID uint, namespace string) ([]types.PodInfo, error) {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return nil, err
	}

	clientset, err := s.createK8sClient(cluster)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var pods []types.PodInfo
	for _, pod := range podList.Items {
		// 获取更精确的 Pod 状态
		podStatus := getPodDetailedStatus(&pod)

		var startTime *time.Time
		if pod.Status.StartTime != nil {
			startTime = &pod.Status.StartTime.Time
		}

		pods = append(pods, types.PodInfo{
			Name:      pod.Name,
			Status:    podStatus,
			NodeName:  pod.Spec.NodeName,
			Namespace: pod.Namespace,
			StartTime: startTime,
		})
	}

	return pods, nil
}

// getPodDetailedStatus 获取详细的 Pod 状态
func getPodDetailedStatus(pod *corev1.Pod) string {
	// 首先检查 Pod 状态条件
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodScheduled && condition.Status == corev1.ConditionFalse {
			return string(condition.Reason)
		}
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionFalse {
			// 检查是否有更具体的错误原因
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.State.Waiting != nil {
					return containerStatus.State.Waiting.Reason
				}
				if containerStatus.State.Terminated != nil {
					return containerStatus.State.Terminated.Reason
				}
			}
		}
	}

	// 检查容器状态以获取更详细的信息
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil {
			waitingReason := containerStatus.State.Waiting.Reason
			// 特殊处理常见的错误状态
			if waitingReason == "CrashLoopBackOff" ||
				waitingReason == "ImagePullBackOff" ||
				waitingReason == "ErrImagePull" {
				return waitingReason
			}
		}

		if containerStatus.State.Terminated != nil {
			terminatedReason := containerStatus.State.Terminated.Reason
			if terminatedReason != "" {
				return terminatedReason
			}
			// 如果没有特定原因，返回退出码
			return fmt.Sprintf("Terminated(code:%d)", containerStatus.State.Terminated.ExitCode)
		}
	}

	// 如果没有更具体的状态，返回 Pod Phase
	return string(pod.Status.Phase)
}

// createK8sClient 创建K8s客户端
func (s *K8SService) createK8sClient(cluster *model.K8SCluster) (*kubernetes.Clientset, error) {
	return core.NewK8sClientFromClusterConfig(
		cluster.ApiServer,
		cluster.CaCert,
		cluster.ClientCert,
		cluster.ClientKey,
		cluster.Token,
		cluster.SkipVerify,
	)
}

// GetClusterByID 根据ID获取集群
func (s *K8SService) GetClusterByID(clusterID uint) (*model.K8SCluster, error) {
	return s.clusterRepo.GetByID(clusterID)
}

// UpdateCluster 更新集群信息
func (s *K8SService) UpdateCluster(cluster *model.K8SCluster) error {
	return s.clusterRepo.Update(cluster)
}

// DeleteCluster 删除集群
func (s *K8SService) DeleteCluster(clusterID uint) error {
	return s.clusterRepo.Delete(clusterID)
}

func (s *K8SService) ListClustersWithConditions(conditions types.ClusterQueryConditions, page, size int) ([]model.K8SCluster, int64, error) {
	return s.clusterRepo.ListWithConditions(conditions, page, size)
}
