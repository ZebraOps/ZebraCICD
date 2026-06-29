package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ZebraOps/ZebraCICD/internal/core"
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/ZebraOps/ZebraCICD/pkg/log"
	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
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

		// 计算 restart count（所有容器的重启次数之和）
		var restartCount int
		for _, cs := range pod.Status.ContainerStatuses {
			restartCount += int(cs.RestartCount)
		}

		// 计算 ready 状态 "n/m"
		readyContainers := 0
		totalContainers := len(pod.Status.ContainerStatuses)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyContainers++
			}
		}
		ready := fmt.Sprintf("%d/%d", readyContainers, totalContainers)

		// 提取 Pod 中的容器详情（名称、就绪、重启次数、镜像、状态）
		var containers []types.ContainerInfo
		for _, cs := range pod.Status.ContainerStatuses {
			state := "running"
			if cs.State.Waiting != nil {
				state = "waiting"
			} else if cs.State.Terminated != nil {
				state = "terminated"
			}
			containers = append(containers, types.ContainerInfo{
				Name:         cs.Name,
				Ready:        cs.Ready,
				RestartCount: cs.RestartCount,
				Image:        cs.Image,
				State:        state,
			})
		}

		pods = append(pods, types.PodInfo{
			Name:         pod.Name,
			Status:       podStatus,
			NodeName:     pod.Spec.NodeName,
			Namespace:    pod.Namespace,
			PodIP:        pod.Status.PodIP,
			StartTime:    startTime,
			Labels:       pod.Labels,
			RestartCount: restartCount,
			Ready:        ready,
			Containers:   containers,
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

// ListNamespaces 根据集群ID动态获取命名空间列表
func (s *K8SService) ListNamespaces(clusterID uint) ([]string, error) {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("集群 %d 不存在: %v", clusterID, err)
	}

	clientset, err := s.createK8sClient(cluster)
	if err != nil {
		return nil, fmt.Errorf("创建K8s客户端失败: %v", err)
	}

	nsList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取命名空间列表失败: %v", err)
	}

	names := make([]string, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		names = append(names, ns.Name)
	}
	return names, nil
}

// ListDeploymentPods 根据 Deployment 名称获取关联的 Pod 列表
// 先查询 K8s Deployment 对象获取其 spec.selector.matchLabels，
// 然后用这些 labels 作为 labelSelector 查询 Pods
func (s *K8SService) ListDeploymentPods(clusterID uint, namespace, deploymentName string) ([]types.PodInfo, error) {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return nil, err
	}

	clientset, err := s.createK8sClient(cluster)
	if err != nil {
		return nil, err
	}

	// 1. 获取 Deployment 对象以提取其 selector
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Deployment %s 失败: %v", deploymentName, err)
	}

	// 2. 从 Deployment 的 selector 构建 labelSelector 字符串
	labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)
	if labelSelector == "" {
		// 如果没有 selector，尝试用 Deployment 名称的 app label 作为 fallback
		labelSelector = fmt.Sprintf("app=%s", deploymentName)
	}

	// 3. 用 labelSelector 查询 Pods
	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("查询 Pods 失败: %v", err)
	}

	var pods []types.PodInfo
	for _, pod := range podList.Items {
		podStatus := getPodDetailedStatus(&pod)

		var startTime *time.Time
		if pod.Status.StartTime != nil {
			startTime = &pod.Status.StartTime.Time
		}

		var restartCount int
		for _, cs := range pod.Status.ContainerStatuses {
			restartCount += int(cs.RestartCount)
		}

		readyContainers := 0
		totalContainers := len(pod.Status.ContainerStatuses)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyContainers++
			}
		}
		ready := fmt.Sprintf("%d/%d", readyContainers, totalContainers)

		// 提取容器详情
		var containers []types.ContainerInfo
		for _, cs := range pod.Status.ContainerStatuses {
			state := "running"
			if cs.State.Waiting != nil {
				state = "waiting"
			} else if cs.State.Terminated != nil {
				state = "terminated"
			}
			containers = append(containers, types.ContainerInfo{
				Name:         cs.Name,
				Ready:        cs.Ready,
				RestartCount: cs.RestartCount,
				Image:        cs.Image,
				State:        state,
			})
		}

		pods = append(pods, types.PodInfo{
			Name:         pod.Name,
			Status:       podStatus,
			NodeName:     pod.Spec.NodeName,
			Namespace:    pod.Namespace,
			PodIP:        pod.Status.PodIP,
			StartTime:    startTime,
			Labels:       pod.Labels,
			RestartCount: restartCount,
			Ready:        ready,
			Containers:   containers,
		})
	}

	return pods, nil
}

// ListDeploymentPodsByTask 根据 DeployTask 获取关联的 Pod 列表
// 这是一个便捷方法，根据任务的 deploy_target 分发到不同的查询逻辑
func (s *K8SService) ListDeploymentPodsByTask(clusterID uint, namespace, deploymentName string) ([]types.PodInfo, error) {
	return s.ListDeploymentPods(clusterID, namespace, deploymentName)
}

// GetPodLogs 获取 Pod 日志（类似 kubectl logs）
func (s *K8SService) GetPodLogs(clusterID uint, namespace, podName string, tailLines int64, container string) (*types.PodLogResponse, error) {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return nil, err
	}

	clientset, err := s.createK8sClient(cluster)
	if err != nil {
		return nil, err
	}

	logOpts := &corev1.PodLogOptions{
		TailLines: &tailLines,
	}
	if container != "" {
		logOpts.Container = container
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, logOpts)
	stream, err := req.Stream(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("获取 Pod %s 日志失败: %v", podName, err)
	}
	defer stream.Close()

	logData, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("读取 Pod %s 日志失败: %v", podName, err)
	}

	return &types.PodLogResponse{
		Output:    string(logData),
		PodName:   podName,
		Namespace: namespace,
		Container: container,
	}, nil
}

// ExecPod 通过 WebSocket 桥接 K8s Pod exec（类似 kubectl exec -it）。
// 在 Pod 中启动 /bin/sh，将 WebSocket 的读写流与 exec 的 stdin/stdout 对接。
func (s *K8SService) ExecPod(clusterID uint, namespace, podName, container string, wsConn *websocket.Conn) error {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return fmt.Errorf("集群 %d 不存在: %v", clusterID, err)
	}

	restConfig := core.NewK8sRestConfig(
		cluster.ApiServer,
		cluster.CaCert,
		cluster.ClientCert,
		cluster.ClientKey,
		cluster.Token,
		cluster.SkipVerify,
	)

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("创建K8s客户端失败: %v", err)
	}

	// WebSocket → exec stream 桥接
	streamHandler := newWSStreamHandler(wsConn)
	defer streamHandler.Stop() // 确保 Stream 返回后 reader goroutine 退出

	// 尝试多个常见 shell（部分容器只含 /bin/bash 或无 /bin/sh）
	shells := [][]string{{"/bin/sh"}, {"/bin/bash"}}
	var lastErr error
	for _, cmd := range shells {
		req := clientset.CoreV1().RESTClient().Post().
			Resource("pods").
			Name(podName).
			Namespace(namespace).
			SubResource("exec").
			VersionedParams(&corev1.PodExecOptions{
				Container: container,
				Command:   cmd,
				Stdin:     true,
				Stdout:    true,
				Stderr:    true,
				TTY:       true,
			}, scheme.ParameterCodec)

		executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
		if err != nil {
			lastErr = fmt.Errorf("创建 exec 执行器失败: %v", err)
			continue
		}

		err = executor.StreamWithContext(context.Background(), remotecommand.StreamOptions{
			Stdin:  streamHandler,
			Stdout: streamHandler,
			Stderr: streamHandler,
			Tty:    true,
		})
		if err != nil {
			log.S().Warnf("Pod exec stream (%v) 结束: %v", cmd, err)
			lastErr = err
			continue
		}
		// 成功，清空错误
		lastErr = nil
		break
	}
	if lastErr != nil {
		errMsg := fmt.Sprintf("\r\n\x1b[31mPod exec 失败: %v\x1b[0m\r\n", lastErr)
		wsConn.WriteMessage(websocket.BinaryMessage, []byte(errMsg))
	}

	return nil
}

// wsReadMsg 封装 WebSocket 读取结果，通过 channel 传递。
type wsReadMsg struct {
	data []byte
	err  error
}

// wsStreamHandler 实现 remotecommand 所需的 io.Reader/io.Writer 接口，
// 将 WebSocket 与 K8s SPDY exec stream 桥接。
//
// 关键设计：Read 不直接调用 conn.ReadMessage()，而是通过独立的 reader
// goroutine 将消息写入 channel。这样当 Stream() 返回时，Stop() 可以
// 触发 reader goroutine 退出，避免 Read 永久阻塞导致的 goroutine 泄漏。
type wsStreamHandler struct {
	conn      *websocket.Conn
	readCh    chan wsReadMsg
	done      chan struct{}
	buf       []byte // 缓存未读完的数据
	bufOff    int
	closeOnce sync.Once
}

// newWSStreamHandler 创建 handler 并启动后台 reader goroutine。
func newWSStreamHandler(conn *websocket.Conn) *wsStreamHandler {
	h := &wsStreamHandler{
		conn:   conn,
		readCh: make(chan wsReadMsg, 1),
		done:   make(chan struct{}),
	}
	go h.readLoop()
	return h
}

// readLoop 在后台持续从 WebSocket 读取消息，通过 channel 交给 Read()。
// 当 Stop() 关闭 done channel 时退出。
func (w *wsStreamHandler) readLoop() {
	for {
		select {
		case <-w.done:
			return
		default:
		}
		_, msg, err := w.conn.ReadMessage()
		// 非阻塞发送：如果 done 已关闭则丢弃消息
		select {
		case w.readCh <- wsReadMsg{msg, err}:
		case <-w.done:
			return
		}
		if err != nil {
			return // 连接关闭或出错，退出循环
		}
	}
}

// Read 实现 io.Reader。优先消费内部缓冲，然后从 channel 读取新消息。
func (w *wsStreamHandler) Read(p []byte) (int, error) {
	// 优先消费内部缓冲中未读完的数据
	if w.bufOff < len(w.buf) {
		n := copy(p, w.buf[w.bufOff:])
		w.bufOff += n
		return n, nil
	}

	// 从 channel 读取（阻塞直到有消息或 done 关闭）
	select {
	case msg := <-w.readCh:
		if msg.err != nil {
			return 0, msg.err
		}
		w.buf = msg.data
		w.bufOff = 0
		n := copy(p, msg.data)
		w.bufOff = n
		return n, nil
	case <-w.done:
		return 0, io.EOF
	}
}

// Write 实现 io.Writer。使用 BinaryMessage 传输终端数据。
func (w *wsStreamHandler) Write(p []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Stop 通知 reader goroutine 退出。幂等（多次调用安全）。
func (w *wsStreamHandler) Stop() {
	w.closeOnce.Do(func() {
		close(w.done)
	})
}

// DeletePod 删除指定的 K8s Pod。
func (s *K8SService) DeletePod(clusterID uint, namespace, podName string) error {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return err
	}

	clientset, err := s.createK8sClient(cluster)
	if err != nil {
		return err
	}

	return clientset.CoreV1().Pods(namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
}

// GetPodMetrics 获取指定命名空间下所有 Pod 的 CPU/内存使用情况。
// 通过 K8s Metrics API (/apis/metrics.k8s.io/v1beta1) 获取。
// 如果集群未安装 metrics-server，返回空 map（不报错）。
func (s *K8SService) GetPodMetrics(clusterID uint, namespace string) (map[string]types.PodMetric, error) {
	cluster, err := s.clusterRepo.GetByID(clusterID)
	if err != nil {
		return nil, err
	}

	restConfig := core.NewK8sRestConfig(
		cluster.ApiServer,
		cluster.CaCert,
		cluster.ClientCert,
		cluster.ClientKey,
		cluster.Token,
		cluster.SkipVerify,
	)

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("创建K8s客户端失败: %v", err)
	}

	// 通过 REST 客户端直接请求 Metrics API
	req := clientset.CoreV1().RESTClient().Get().
		AbsPath("/apis/metrics.k8s.io/v1beta1").
		Namespace(namespace).
		Resource("pods").
		SetHeader("Accept", "application/json")

	raw, err := req.DoRaw(context.TODO())
	if err != nil {
		// metrics-server 未安装或不可用 → 返回空 map
		log.S().Debugf("获取 Pod Metrics 失败（可能未安装 metrics-server）: %v", err)
		return map[string]types.PodMetric{}, nil
	}

	// 手动解析 JSON，避免引入额外的 metrics 客户端包
	var result struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Containers []struct {
				Usage struct {
					CPU    string `json:"cpu"`
					Memory string `json:"memory"`
				} `json:"usage"`
			} `json:"containers"`
		} `json:"items"`
	}

	if err := json.Unmarshal(raw, &result); err != nil {
		log.S().Warnf("解析 Pod Metrics 失败: %v", err)
		return map[string]types.PodMetric{}, nil
	}

	metrics := make(map[string]types.PodMetric)
	for _, item := range result.Items {
		var totalCPU, totalMemory int64
		for _, c := range item.Containers {
			cpuMilli, _ := parseCPUToMilli(c.Usage.CPU)
			memBytes, _ := parseMemToBytes(c.Usage.Memory)
			totalCPU += cpuMilli
			totalMemory += memBytes
		}
		metrics[item.Metadata.Name] = types.PodMetric{
			CPU:    formatMilliCPU(totalCPU),
			Memory: formatBytes(totalMemory),
		}
	}

	return metrics, nil
}

// parseCPUToMilli 将 K8s CPU 值（"100m" / "1" / "500u"）转换为毫核。
func parseCPUToMilli(cpu string) (int64, error) {
	cpu = strings.TrimSpace(cpu)
	if cpu == "" {
		return 0, nil
	}
	// "100m" → 100
	if strings.HasSuffix(cpu, "m") {
		return strconv.ParseInt(strings.TrimSuffix(cpu, "m"), 10, 64)
	}
	// "1" → 1000 (cores to millicores)
	// Nano cores "5000000000n" → 5
	if strings.HasSuffix(cpu, "n") {
		v, err := strconv.ParseInt(strings.TrimSuffix(cpu, "n"), 10, 64)
		return v / 1000000, err
	}
	v, err := strconv.ParseFloat(cpu, 64)
	return int64(v * 1000), err
}

// parseMemToBytes 将 K8s 内存值（"128Mi" / "1Gi" / "128974848"）转换为字节。
func parseMemToBytes(mem string) (int64, error) {
	mem = strings.TrimSpace(mem)
	if mem == "" {
		return 0, nil
	}
	if strings.HasSuffix(mem, "Ki") {
		v, err := strconv.ParseInt(strings.TrimSuffix(mem, "Ki"), 10, 64)
		return v * 1024, err
	}
	if strings.HasSuffix(mem, "Mi") {
		v, err := strconv.ParseInt(strings.TrimSuffix(mem, "Mi"), 10, 64)
		return v * 1024 * 1024, err
	}
	if strings.HasSuffix(mem, "Gi") {
		v, err := strconv.ParseInt(strings.TrimSuffix(mem, "Gi"), 10, 64)
		return v * 1024 * 1024 * 1024, err
	}
	// Plain bytes number
	return strconv.ParseInt(mem, 10, 64)
}

// formatMilliCPU 将毫核值格式化为人类可读的 CPU 字符串。
func formatMilliCPU(milli int64) string {
	if milli >= 1000 {
		return fmt.Sprintf("%.2f", float64(milli)/1000)
	}
	return fmt.Sprintf("%dm", milli)
}

// formatBytes 将字节数格式化为人类可读的内存字符串。
func formatBytes(bytes int64) string {
	if bytes >= 1024*1024*1024 {
		return fmt.Sprintf("%.2fGi", float64(bytes)/(1024*1024*1024))
	}
	if bytes >= 1024*1024 {
		return fmt.Sprintf("%.2fMi", float64(bytes)/(1024*1024))
	}
	if bytes >= 1024 {
		return fmt.Sprintf("%.2fKi", float64(bytes)/1024)
	}
	return fmt.Sprintf("%dB", bytes)
}

