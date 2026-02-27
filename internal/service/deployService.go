package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ZebraOps/ZebraCICD/config"
	"github.com/ZebraOps/ZebraCICD/internal/core"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/pkg/log"
	"github.com/samber/lo"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type DeployService struct {
	db         *gorm.DB
	cfg        *config.Config
	gitlab     *core.GitLabClient
	harbor     *core.HarborClient
	jenkins    *core.JenkinsClient // 新增Jenkins客户端
	k8s        *core.K8sClient     // 新增K8s客户端
	workerStop chan struct{}
}
type JenkinsBuildResult struct {
	JobName     string
	BuildNumber int
	QueueID     int
}

// int32Ptr 返回指向int32值的指针
func int32Ptr(i int32) *int32 {
	return &i
}

// getK8sClient 根据集群ID获取K8s客户端
func (s *DeployService) getK8sClient(clusterID uint) (*kubernetes.Clientset, error) {
	// 从数据库获取K8s集群配置
	var cluster model.K8SCluster
	if err := s.db.First(&cluster, clusterID).Error; err != nil {
		return nil, err
	}

	// 使用core包中的方法创建客户端
	return core.NewK8sClientFromClusterConfig(
		cluster.ApiServer,
		cluster.CaCert,
		cluster.ClientCert,
		cluster.ClientKey,
		cluster.Token,
		cluster.SkipVerify,
	)
}

func NewDeployService(db *gorm.DB, cfg *config.Config) *DeployService {
	gc := core.NewGitLabClient(cfg.GitLabURL, cfg.GitLabToken)
	hc := core.NewHarborClient(cfg.HarborURL)
	jc := core.NewJenkinsClient(cfg.JenkinsURL, cfg.JenkinsUser, cfg.JenkinsPass) // 使用公共构造函数

	return &DeployService{
		db:         db,
		cfg:        cfg,
		gitlab:     gc,
		harbor:     hc,
		jenkins:    jc,
		workerStop: make(chan struct{}),
	}
}

func (s *DeployService) CreateTask(t *model.DeployTask) error {
	t.Status = "PENDING"
	timestamp := time.Now().Format("20060102150405")
	t.ImageTag = fmt.Sprintf("%s", timestamp)

	if err := s.db.Create(t).Error; err != nil {
		return err
	}

	// 启动部署流程
	go s.processDeploymentTask(t.ID)
	return nil
}

// processDeploymentTask 处理部署任务的主要流程
func (s *DeployService) processDeploymentTask(taskID uint) {
	var task model.DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		log.S().Infof("processDeploymentTask: failed to load task %d: %v", taskID, err)
		return
	}

	// 1. 开始构建阶段
	s.updateTaskStatus(taskID, "BUILDING", "开始Jenkins构建流程")

	// 2. 触发Jenkins构建
	buildResult, err := s.triggerJenkinsBuild(&task)
	if err != nil {
		s.updateTaskStatus(taskID, "FAILED", fmt.Sprintf("Jenkins构建失败: %v", err))
		return
	}

	// 3. 等待构建完成
	if !s.waitForJenkinsBuild(buildResult.JobName, buildResult.BuildNumber) {
		s.updateTaskStatus(taskID, "FAILED", "Jenkins构建失败或超时")
		return
	}

	// 4. 开始推送阶段
	s.updateTaskStatus(taskID, "PUSHING", "开始推送镜像到Harbor")
	fmt.Println(task.HarborProject, task.ImageName, task.ImageTag)

	// 5. 验证镜像推送
	if !s.verifyImageInHarbor(task.HarborProject, task.ImageName, task.ImageTag) {
		s.updateTaskStatus(taskID, "FAILED", "Harbor镜像验证失败")
		return
	}

	// 6. 开始部署阶段
	s.updateTaskStatus(taskID, "DEPLOYING", "开始部署到K8s集群")

	// 7. 部署到K8s
	if err := s.deployToK8s(&task); err != nil {
		s.updateTaskStatus(taskID, "FAILED", fmt.Sprintf("K8s部署失败: %v", err))
		return
	}

	// 8. 部署成功
	s.updateTaskStatus(taskID, "SUCCESS", "部署成功完成")
}

// triggerJenkinsBuild 触发Jenkins构建
func (s *DeployService) triggerJenkinsBuild(task *model.DeployTask) (*core.JenkinsBuildResult, error) {
	// 1. 根据仓库ID获取构建模板
	var repo model.Repo
	if err := s.db.Preload("Templates").First(&repo, task.ProjectID).Error; err != nil {
		return nil, fmt.Errorf("failed to get repo: %v", err)
	}

	if len(repo.Templates) == 0 {
		return nil, fmt.Errorf("no build template found for repo %d", task.ProjectID)
	}

	// 2. 获取第一个构建模板（实际业务中可能需要更复杂的逻辑）
	buildTemplate := repo.Templates[0]

	// 3. 检查 Jenkins Job 是否存在，不存在则创建
	jobExists, err := s.jenkins.CheckJobExists(buildTemplate.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check job existence: %v", err)
	}

	if !jobExists {
		fmt.Fprintf(os.Stdout, "Jenkins Job %s does not exist, creating...\n", buildTemplate.Name)
		// 创建新的 Jenkins Job
		jobConfig := s.generateJobConfig(buildTemplate, task.GitRef, repo.RepoURL, task.ImageTag)
		if err := s.jenkins.CreateJob(buildTemplate.Name, jobConfig); err != nil {
			return nil, fmt.Errorf("failed to create job: %v", err)
		}
	}

	params := map[string]string{
		"TARGET_BRANCH": task.GitRef,
		"Repo_URL":      repo.RepoURL,
		"Tag":           task.ImageTag,
	}

	if err := s.jenkins.Authenticate(); err != nil {
		return nil, fmt.Errorf("Jenkins authentication failed: %v", err)
	}
	fmt.Println("开始触发Jenkins构建")

	return s.jenkins.BuildJob(buildTemplate.Name, params)
}

// waitForJenkinsBuild 等待Jenkins构建完成
func (s *DeployService) waitForJenkinsBuild(jobName string, buildNumber int) bool {
	timeout := time.After(10 * time.Minute) // 设置超时时间
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return false
		case <-ticker.C:
			status, err := s.jenkins.GetBuildStatus(jobName, buildNumber)
			if err != nil {
				log.S().Infof("Error getting build status: %v", err)
				continue
			}

			if status.IsComplete() {
				return status.IsSuccess()
			}
		}
	}
}

// verifyImageInHarbor 验证Harbor中的镜像
func (s *DeployService) verifyImageInHarbor(project, imageName, tag string) bool {
	// 查询Harbor确认镜像已推送
	tags, err := s.harbor.GetImageTags(project, imageName)
	if err != nil {
		log.S().Infof("Error getting image tags from Harbor: %v", err)
		return false
	}

	for _, harborTag := range tags {
		if harborTag.Name == tag {
			return true
		}
	}
	return false
}

// deployToK8s 部署到K8s集群
func (s *DeployService) deployToK8s(task *model.DeployTask) error {

	// 2. 根据环境配置获取集群信息（假设环境表中有集群ID字段）
	// 如果环境表没有直接关联集群，需要通过其他方式获取
	var cluster model.K8SCluster
	if err := s.db.First(&cluster, task.K8sClusterID).Error; err != nil {
		return fmt.Errorf("failed to get k8s cluster: %v", err)
	}

	// 3. 创建K8s客户端
	clientset, err := s.getK8sClientByCluster(cluster)
	if err != nil {
		return err
	}

	// 4. 根据仓库ID获取部署模板（通过仓库与部署模板的关联关系）
	var repo model.Repo
	if err := s.db.Preload("DeploymentTemplates").First(&repo, task.ProjectID).Error; err != nil {
		return fmt.Errorf("failed to get repo: %v", err)
	}

	if len(repo.DeploymentTemplates) == 0 {
		return fmt.Errorf("no deployment template found for repo %d", task.ProjectID)
	}

	// 5. 使用第一个部署模板（实际业务中可能需要更复杂的逻辑）
	deploymentTemplate := repo.DeploymentTemplates[0]

	// 6. 解析模板内容并进行参数替换
	renderedYAML := s.renderTemplate(deploymentTemplate.Content, task)

	// 7. 解析YAML并创建K8s资源
	return s.applyYAMLResources(clientset, renderedYAML, task)
}

// getK8sClientByCluster 根据集群信息创建K8s客户端
func (s *DeployService) getK8sClientByCluster(cluster model.K8SCluster) (*kubernetes.Clientset, error) {
	// 使用core包中的方法创建客户端
	return core.NewK8sClientFromClusterConfig(
		cluster.ApiServer,
		cluster.CaCert,
		cluster.ClientCert,
		cluster.ClientKey,
		cluster.Token,
		cluster.SkipVerify,
	)
}

// renderTemplate 渲染部署模板
func (s *DeployService) renderTemplate(templateContent string, task *model.DeployTask) string {
	// 获取项目相关信息
	var projectName string
	var repo model.Repo
	if err := s.db.Select("e_name").First(&repo, task.ProjectID).Error; err == nil {
		projectName = repo.EName
	} else {
		projectName = fmt.Sprintf("calc-api-project-%d", task.ProjectID)
	}

	// 先处理换行符，再替换占位符
	rendered := templateContent

	// 多步骤处理换行符
	rendered = strings.ReplaceAll(rendered, "\\n", "\n")
	rendered = strings.ReplaceAll(rendered, "\r\n", "\n")
	rendered = strings.ReplaceAll(rendered, "\r", "\n")

	// 替换模板中的占位符，保持YAML格式
	rendered = strings.ReplaceAll(rendered, "{{IMAGE_TAG}}", task.ImageTag)
	rendered = strings.ReplaceAll(rendered, "{{NAMESPACE}}", task.K8sNamespace)
	rendered = strings.ReplaceAll(rendered, "{{PROJECT_NAME}}", projectName)
	rendered = strings.ReplaceAll(rendered, "{{ENV_NAME}}", fmt.Sprintf("env-%d", task.EnvID))

	return rendered
}

// applyYAMLResources 应用YAML资源到K8s集群
func (s *DeployService) applyYAMLResources(clientset *kubernetes.Clientset, yamlContent string, task *model.DeployTask) error {
	processedContent := strings.ReplaceAll(yamlContent, "\\n", "\n")
	processedContent = strings.ReplaceAll(processedContent, "\r\n", "\n")
	processedContent = strings.ReplaceAll(processedContent, "\r", "\n")

	documents := strings.Split(processedContent, "---")

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var rawObj map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &rawObj); err != nil {
			return fmt.Errorf("YAML 格式错误: %v", err)
		}

		// ✅ 关键：在这里统一转换所有 key
		rawObj = s.convertMapToStringKey(rawObj).(map[string]interface{})

		kind := s.safeExtractValue(rawObj, "kind")
		if kind == "" {
			return fmt.Errorf("资源缺少必要字段 (Kind: %s)", kind)
		}

		obj := &unstructured.Unstructured{Object: rawObj}

		switch kind {
		case "Namespace":
			name := s.extractValueFromMetadata(rawObj, "name")
			if err := s.applyNamespace(clientset, name); err != nil {
				return err
			}
			log.S().Infof("Applied Namespace: %s", name)
		case "ConfigMap":
			name := s.extractValueFromMetadata(rawObj, "name")
			ns := s.extractValueFromMetadata(rawObj, "namespace")
			if err := s.applyConfigMap(name, ns, clientset, obj); err != nil {
				return err
			}
			log.S().Infof("Applied ConfigMap: %s/%s", ns, name)
		case "Deployment":
			if err := s.applyDeployment(clientset, obj, task); err != nil {
				return err
			}
			log.S().Infof("Applied Deployment")
		case "Service":
			if err := s.applyService(clientset, obj); err != nil {
				return err
			}
			log.S().Infof("Applied Service")
		default:
			log.S().Warnf("Unsupported resource type: %s", kind)
		}
	}

	return nil
}

// convertMapToStringKey 递归将所有 map 的 interface{} key 转为 string
func (s *DeployService) convertMapToStringKey(input interface{}) interface{} {
	switch v := input.(type) {
	case map[interface{}]interface{}:
		// 关键：处理 interface{} key 的 map
		m := make(map[string]interface{}, len(v))
		for key, val := range v {
			// 强制转换 key 为 string
			m[fmt.Sprintf("%v", key)] = s.convertMapToStringKey(val)
		}
		return m
	case map[string]interface{}:
		// 已经是 string key，继续递归处理值
		m := make(map[string]interface{}, len(v))
		for key, val := range v {
			m[key] = s.convertMapToStringKey(val)
		}
		return m
	case []interface{}:
		// 处理数组中的元素
		for i, item := range v {
			v[i] = s.convertMapToStringKey(item)
		}
		return v
	default:
		return v
	}
}

// safeExtractValue 安全地从 map 中提取值，处理各种可能的数据类型
func (s *DeployService) safeExtractValue(obj map[string]interface{}, key string) string {
	if val, exists := obj[key]; exists && val != nil {
		switch v := val.(type) {
		case string:
			return strings.TrimSpace(v)
		case int:
			return fmt.Sprintf("%d", v)
		case int64:
			return fmt.Sprintf("%d", v)
		case float64:
			// 检查是否为整数
			if v == float64(int64(v)) {
				return fmt.Sprintf("%.0f", v)
			}
			return fmt.Sprintf("%g", v)
		case float32:
			if float64(v) == float64(int64(v)) {
				return fmt.Sprintf("%.0f", float64(v))
			}
			return fmt.Sprintf("%g", float64(v))
		case bool:
			return fmt.Sprintf("%t", v)
		case nil:
			return ""
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// extractValueFromMetadata 从 metadata 中提取指定键的值，处理各种可能的格式
func (s *DeployService) extractValueFromMetadata(rawObj map[string]interface{}, key string) string {
	if metadata, exists := rawObj["metadata"]; exists && metadata != nil {
		fmt.Println("metadata:", metadata)
		switch v := metadata.(type) {
		case map[string]interface{}:
			// 正常情况：metadata 是一个 map
			return s.safeExtractValue(v, key)
		case map[interface{}]interface{}:
			// 特殊情况：metadata 是 map[interface{}]interface{} 类型
			for metaKey, value := range v {
				if keyStr, ok := metaKey.(string); ok && keyStr == key {
					// 找到了指定的键，将其值转换为字符串
					switch val := value.(type) {
					case string:
						return val
					case int:
						return fmt.Sprintf("%d", val)
					case float64:
						if val == float64(int64(val)) {
							return fmt.Sprintf("%.0f", val)
						}
						return fmt.Sprintf("%g", val)
					default:
						return fmt.Sprintf("%v", val)
					}
				}
			}
			return ""
		case []interface{}:
			// 异常情况：metadata 是一个数组，尝试从中找到指定键
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if _, exists := itemMap[key]; exists {
						return s.safeExtractValue(itemMap, key)
					}
				}
			}
			return ""
		case string:
			// 如果 metadata 被错误地解析为字符串，尝试再次解析
			var parsedMetadata map[string]interface{}
			if err := yaml.Unmarshal([]byte(v), &parsedMetadata); err == nil {
				return s.safeExtractValue(parsedMetadata, key)
			}
			return ""
		default:
			log.S().Infof("metadata 不是期望的类型: %T", v)
			return ""
		}
	}
	return ""
}

// applyNamespace 创建或更新Namespace
func (s *DeployService) applyNamespace(clientset *kubernetes.Clientset, nsName string) error {
	// 检查Namespace是否存在
	_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), nsName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// 创建Namespace
			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create namespace %s: %v", nsName, err)
			}
			log.S().Infof("Created namespace: %s", nsName)
			return nil
		}
		// 其他错误，如权限不足等
		return fmt.Errorf("failed to get namespace %s: %v", nsName, err)
	}

	// Namespace 已存在，记录日志并跳过
	log.S().Infof("Namespace %s already exists, skipping creation", nsName)
	return nil
}

// applyConfigMap 使用 Server-Side Apply 创建或更新 ConfigMap
func (s *DeployService) applyConfigMap(name string, ns string, clientset *kubernetes.Clientset, obj *unstructured.Unstructured) error {

	// ✅ 步骤1：安全转换
	configMap := &corev1.ConfigMap{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), configMap); err != nil {
		log.S().Errorf("Failed to convert unstructured object to ConfigMap: %v", err)
		return fmt.Errorf("failed to convert unstructured object to ConfigMap: %v", err)
	}

	// ✅ 步骤2：确保 namespace 和 name 正确
	if configMap.Namespace == "" {
		configMap.Namespace = ns
	}
	if configMap.Name == "" {
		configMap.Name = name
	}

	// ✅ 步骤4：应用资源
	applyConfig := corev1apply.ConfigMap(configMap.Name, configMap.Namespace).
		WithData(configMap.Data).
		WithBinaryData(configMap.BinaryData)

	_, err := clientset.CoreV1().ConfigMaps(configMap.Namespace).Apply(context.TODO(), applyConfig, metav1.ApplyOptions{
		FieldManager: "zebra-cicd-controller",
		Force:        true,
	})
	if err != nil {
		return fmt.Errorf("failed to apply ConfigMap %s in namespace %s: %v", configMap.Name, configMap.Namespace, err)
	}

	log.S().Infof("Applied ConfigMap: %s in namespace: %s", configMap.Name, configMap.Namespace)
	return nil
}

// applyService 创建或更新Service
func (s *DeployService) applyService(clientset *kubernetes.Clientset, obj *unstructured.Unstructured) error {
	service := &corev1.Service{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), service); err != nil {
		return err
	}

	// 使用 Server-Side Apply
	applyConfig := corev1apply.Service(service.Name, service.Namespace).
		WithSpec(corev1apply.ServiceSpec().
			WithPorts(lo.Map(service.Spec.Ports, func(p corev1.ServicePort, _ int) *corev1apply.ServicePortApplyConfiguration {
				return corev1apply.ServicePort().WithPort(p.Port).WithTargetPort(p.TargetPort)
			})...). // 注意这里的 ...
			WithSelector(service.Spec.Selector))

	_, err := clientset.CoreV1().Services(service.Namespace).Apply(context.TODO(), applyConfig, metav1.ApplyOptions{
		FieldManager: "zebra-cicd-controller",
		Force:        true,
	})
	return err
}

// applyDeployment 使用 Server-Side Apply 部署 Deployment
func (s *DeployService) applyDeployment(clientset *kubernetes.Clientset, obj *unstructured.Unstructured, task *model.DeployTask) error {
	// 转换为 Deployment 对象
	deployment := &appsv1.Deployment{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), deployment); err != nil {
		return fmt.Errorf("failed to convert unstructured object to Deployment: %v", err)
	}

	// 更新镜像标签
	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]
		if strings.Contains(container.Image, ":") {
			imageParts := strings.Split(container.Image, ":")
			container.Image = fmt.Sprintf("%s:%s", imageParts[0], task.ImageTag)
		} else {
			container.Image = fmt.Sprintf("%s:%s", container.Image, task.ImageTag)
		}
	}

	var replicas int32
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	} else {
		replicas = 1 // 默认副本数为 1
	}

	// 使用 lo.Map 并将结果转换为指针类型切片
	matchExpressions := lo.Map(deployment.Spec.Selector.MatchExpressions,
		func(expr metav1.LabelSelectorRequirement, index int) *metav1apply.LabelSelectorRequirementApplyConfiguration {
			return metav1apply.LabelSelectorRequirement().
				WithKey(expr.Key).
				WithOperator(expr.Operator).
				WithValues(expr.Values...)
		})

	selector := metav1apply.LabelSelector().
		WithMatchLabels(deployment.Spec.Selector.MatchLabels).
		WithMatchExpressions(matchExpressions...) // 直接传递指针切片

	containers := lo.Map(deployment.Spec.Template.Spec.Containers,
		func(c corev1.Container, index int) *corev1apply.ContainerApplyConfiguration {
			ports := lo.Map(c.Ports, func(p corev1.ContainerPort, index int) *corev1apply.ContainerPortApplyConfiguration {
				return corev1apply.ContainerPort().WithContainerPort(p.ContainerPort)
			})

			envFrom := lo.Map(c.EnvFrom, func(e corev1.EnvFromSource, index int) *corev1apply.EnvFromSourceApplyConfiguration {
				optional := false
				if e.ConfigMapRef.Optional != nil {
					optional = *e.ConfigMapRef.Optional
				}
				configMapRef := corev1apply.ConfigMapEnvSource().
					WithName(e.ConfigMapRef.Name).
					WithOptional(optional)
				return corev1apply.EnvFromSource().WithConfigMapRef(configMapRef)
			})

			return corev1apply.Container().
				WithName(c.Name).
				WithImage(c.Image).
				WithPorts(ports...).
				WithEnvFrom(envFrom...)
		})

	applyConfig := appsv1apply.Deployment(deployment.Name, deployment.Namespace).
		WithSpec(appsv1apply.DeploymentSpec().
			WithReplicas(replicas).
			WithSelector(selector).
			WithTemplate(corev1apply.PodTemplateSpec().
				WithLabels(deployment.Spec.Template.Labels).
				WithSpec(corev1apply.PodSpec().
					WithContainers(containers...), // 直接传递指针切片
				),
			),
		)

	// 使用 Server-Side Apply 提交资源
	_, err := clientset.AppsV1().Deployments(deployment.Namespace).Apply(context.TODO(), applyConfig, metav1.ApplyOptions{
		FieldManager: "zebra-cicd-controller", // 设置字段管理者
		Force:        true,                    // 强制覆盖冲突字段
	})
	if err != nil {
		return fmt.Errorf("failed to apply Deployment %s in namespace %s: %v", deployment.Name, deployment.Namespace, err)
	}

	log.S().Infof("Applied Deployment: %s in namespace: %s", deployment.Name, deployment.Namespace)
	return nil
}

// updateTaskStatus 更新任务状态
func (s *DeployService) updateTaskStatus(taskID uint, status, message string) {
	now := time.Now()
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": now,
	}

	if status == "SUCCESS" || status == "FAILED" {
		updates["finished_at"] = now
	}

	s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(updates)
	log.S().Infof("Task %d: %s - %s", taskID, status, message)
}

// GetTask 根据ID获取部署任务
func (s *DeployService) GetTask(id uint) (*model.DeployTask, error) {
	var t model.DeployTask
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// StartWorker 启动后台工作进程
func (s *DeployService) StartWorker() {
	// Start a goroutine to poll for pending tasks and process them
	ticker := time.NewTicker(s.cfg.WorkerPeriod)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.pollAndProcess()
			case <-s.workerStop:
				ticker.Stop()
				return
			}
		}
	}()
}

// pollAndProcess 轮询并处理待处理的任务
func (s *DeployService) pollAndProcess() {
	var tasks []model.DeployTask
	if err := s.db.Where("status = ?", "PENDING").Find(&tasks).Error; err != nil {
		log.S().Infof("error querying pending tasks: %v", err)
		return
	}
	for _, t := range tasks {
		// process in background goroutine per task
		go s.processTask(t.ID)
	}
}

// processTask 处理单个任务
func (s *DeployService) processTask(id uint) {
	// load fresh record
	var t model.DeployTask
	if err := s.db.First(&t, id).Error; err != nil {
		log.S().Infof("processTask: failed to load task %d: %v", id, err)
		return
	}

	// move to BUILDING
	s.db.Model(&t).Updates(map[string]interface{}{
		"status":     "BUILDING",
		"started_at": time.Now(),
	})

	// prepare log file
	logDir := os.TempDir()
	logFile := filepath.Join(logDir, fmt.Sprintf("deploy_task_%d.log", t.ID))
	f, _ := os.Create(logFile)
	defer f.Close()

	// Simulate build step
	fmt.Fprintf(f, "[%s] Task %d: starting build for ref %s\n", time.Now().Format(time.RFC3339), t.ID, t.GitRef)
	time.Sleep(3 * time.Second) // simulate
	// update image_tag if empty (simulate)
	if t.ImageTag == "" {
		t.ImageTag = fmt.Sprintf("%s-%d", t.GitRef, time.Now().Unix())
	}
	fmt.Fprintf(f, "[%s] Task %d: build finished, image=%s\n", time.Now().Format(time.RFC3339), t.ID, t.ImageTag)

	// Move to DEPLOYING
	s.db.Model(&t).Updates(map[string]interface{}{
		"status":    "DEPLOYING",
		"image_tag": t.ImageTag,
		"log_path":  logFile,
	})

	// Simulate deploy step
	fmt.Fprintf(f, "[%s] Task %d: deploying to env %d\n", time.Now().Format(time.RFC3339), t.ID, t.EnvID)
	time.Sleep(3 * time.Second) // simulate

	// mark success
	now := time.Now()
	updateErr := s.db.Model(&t).Updates(map[string]interface{}{
		"status":      "SUCCESS",
		"finished_at": now,
		"updated_at":  now,
	}).Error
	if updateErr != nil {
		fmt.Fprintf(f, "[%s] Task %d: failed updating status: %v\n", time.Now().Format(time.RFC3339), t.ID, updateErr)
	} else {
		fmt.Fprintf(f, "[%s] Task %d: deploy SUCCESS\n", time.Now().Format(time.RFC3339), t.ID)
	}
}

// StopWorker 停止后台工作进程
func (s *DeployService) StopWorker() {
	close(s.workerStop)
}

func (s *DeployService) generateJobConfig(template *model.BuildTemplate, targetBranch, repoURL, tag string) string {
	// 替换换行符并转义反斜杠
	pipelineContent := strings.ReplaceAll(template.Pipeline, "\\n", "\n")
	pipelineContent = strings.ReplaceAll(pipelineContent, "\r\n", "\n")
	pipelineContent = strings.ReplaceAll(pipelineContent, "\\", "")
	pipelineContent = strings.ReplaceAll(pipelineContent, "\r", "\n")

	// ✅ 对 Groovy 脚本进行 CDATA 转义
	escapedPipeline := escapeXMLContent(pipelineContent)

	// ✅ 核心修复：添加参数定义
	config := fmt.Sprintf(`<?xml version='1.1' encoding='UTF-8'?>
<flow-definition plugin="workflow-job@2.40">
	<description>Generated by Zebra CI/CD for %s</description>
	<keepDependencies>false</keepDependencies>
	<properties>
		<!-- ✅ 参数定义：定义 TARGET_BRANCH, Repo_URL, Tag 三个参数 -->
		<hudson.model.ParametersDefinitionProperty>
			<parameterDefinitions>
				<hudson.model.StringParameterDefinition>
					<name>TARGET_BRANCH</name>
					<description>Target git branch</description>
					<defaultValue>%s</defaultValue>
					<trim>false</trim>
				</hudson.model.StringParameterDefinition>
				<hudson.model.StringParameterDefinition>
					<name>Repo_URL</name>
					<description>Git repository URL</description>
					<defaultValue>%s</defaultValue>
					<trim>false</trim>
				</hudson.model.StringParameterDefinition>
				<hudson.model.StringParameterDefinition>
					<name>Tag</name>
					<description>Image tag/version</description>
					<defaultValue></defaultValue>
					<trim>false</trim>
				</hudson.model.StringParameterDefinition>
			</parameterDefinitions>
		</hudson.model.ParametersDefinitionProperty>
		<jenkins.model.BuildDiscarderProperty>
			<strategy class="hudson.tasks.LogRotator">
				<daysToKeep>-1</daysToKeep>
				<numToKeep>10</numToKeep>
				<artifactDaysToKeep>-1</artifactDaysToKeep>
				<artifactNumToKeep>-1</artifactNumToKeep>
			</strategy>
		</jenkins.model.BuildDiscarderProperty>
		<org.jenkinsci.plugins.workflow.job.properties.PipelineTriggersJobProperty>
			<triggers/>
		</org.jenkinsci.plugins.workflow.job.properties.PipelineTriggersJobProperty>
	</properties>
	<definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition" plugin="workflow-cps@2.87">
		<script><![CDATA[%s]]></script>
		<sandbox>true</sandbox>
	</definition>
	<triggers/>
	<disabled>false</disabled>
</flow-definition>`, targetBranch, repoURL, tag, escapedPipeline)

	return config
}

// ✅ XML 转义函数
func escapeXMLContent(content string) string {
	replacer := strings.NewReplacer(
		"]]>", "]]]]><![CDATA[>", // 处理 CDATA 结束符
	)
	return replacer.Replace(content)
}
