package service

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ZebraOps/ZebraCICD/config"
	"github.com/ZebraOps/ZebraCICD/internal/core"
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/pkg/log"
	"github.com/ZebraOps/ZebraCICD/pkg/queue"
	sshclient "github.com/ZebraOps/ZebraCICD/pkg/ssh"
	"github.com/samber/lo"
	"golang.org/x/crypto/ssh"
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
	db               *gorm.DB
	cfg              *config.Config
	gitlab           *core.GitLabClient
	registry         core.RegistryClient // 动态镜像仓库客户端
	jenkins          *core.JenkinsClient
	k8s              *core.K8sClient
	queueClient      *queue.Client
	serverRepo       *handler.ServerRepository
	stageHistoryRepo *handler.StageHistoryRepository
}

type JenkinsBuildResult struct {
	JobName     string
	BuildNumber int
	QueueID     int
}

var ErrNoJenkinsBuildInfo = stderrors.New("task has no Jenkins build info")

// int32Ptr 返回指向int32值的指针
func int32Ptr(i int32) *int32 {
	return &i
}

// stripURLProtocol 剥离 URL 中的 http:// 或 https:// 前缀
// Docker 镜像引用格式不允许包含协议前缀
func stripURLProtocol(rawURL string) string {
	u := strings.TrimPrefix(rawURL, "https://")
	u = strings.TrimPrefix(u, "http://")
	return strings.TrimSuffix(u, "/")
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

func NewDeployService(db *gorm.DB, cfg *config.Config, queueClient *queue.Client, serverRepo *handler.ServerRepository, stageHistoryRepo *handler.StageHistoryRepository) *DeployService {
	// 使用配置值创建 GitLab 客户端
	gitlabTimeout := cfg.GitlabHTTPTimeout
	if gitlabTimeout == 0 {
		gitlabTimeout = 15 * time.Second
	}
	gc := core.NewGitLabClientWithTimeout(cfg.GitLabURL, cfg.GitLabToken, gitlabTimeout)

	rc := core.NewV2RegistryAdapter(cfg.RegistryURL, cfg.RegistryUser, cfg.RegistryPass)

	// 使用配置值创建 Jenkins 客户端
	jenkinsHTTPTimeout := cfg.JenkinsHTTPTimeout
	if jenkinsHTTPTimeout == 0 {
		jenkinsHTTPTimeout = 30 * time.Second
	}
	jenkinsBuildWaitTimeout := cfg.JenkinsBuildWaitTimeout
	if jenkinsBuildWaitTimeout == 0 {
		jenkinsBuildWaitTimeout = 10 * time.Minute
	}
	jenkinsBuildPollInterval := cfg.JenkinsBuildPollInterval
	if jenkinsBuildPollInterval == 0 {
		jenkinsBuildPollInterval = 10 * time.Second
	}
	jc := core.NewJenkinsClientWithTimeout(cfg.JenkinsURL, cfg.JenkinsUser, cfg.JenkinsPass, jenkinsHTTPTimeout, jenkinsBuildWaitTimeout, jenkinsBuildPollInterval)

	return &DeployService{
		db:               db,
		cfg:              cfg,
		gitlab:           gc,
		registry:         rc,
		jenkins:          jc,
		queueClient:      queueClient,
		serverRepo:       serverRepo,
		stageHistoryRepo: stageHistoryRepo,
	}
}

// startStage creates a StageHistory record with status=running and sets StartedAt.
func (s *DeployService) startStage(taskID uint, stage string, retryCount int) error {
	now := time.Now()
	stageHistory := &model.StageHistory{
		TaskID:     taskID,
		Stage:      stage,
		Status:     "running",
		RetryCount: retryCount,
		StartedAt:  &now,
	}
	return s.stageHistoryRepo.Create(stageHistory)
}

// finishStage updates the StageHistory record to status=success/failed and sets FinishedAt.
func (s *DeployService) finishStage(taskID uint, stage string, status string, errorMsg string) error {
	now := time.Now()
	stageHistory, err := s.stageHistoryRepo.GetByTaskIDAndStage(taskID, stage)
	if err != nil {
		return err
	}
	stageHistory.Status = status
	stageHistory.FinishedAt = &now
	if errorMsg != "" {
		stageHistory.ErrorMessage = errorMsg
	}
	return s.stageHistoryRepo.Update(stageHistory)
}

// updateStageLogSummary updates the LogSummary field for the current running stage.
func (s *DeployService) updateStageLogSummary(taskID uint, stage string, summary string) error {
	stageHistory, err := s.stageHistoryRepo.GetByTaskIDAndStage(taskID, stage)
	if err != nil {
		return err
	}
	stageHistory.LogSummary = summary
	return s.stageHistoryRepo.Update(stageHistory)
}

func (s *DeployService) CreateTask(t *model.DeployTask) error {
	// 设置默认执行模式
	if t.ExecutionMode == "" {
		t.ExecutionMode = "auto"
	}
	if t.ScheduleType == "" {
		t.ScheduleType = "immediate"
	}
	if t.BuildStatus == "" {
		t.BuildStatus = "pending"
	}
	if t.DeployStatus == "" {
		t.DeployStatus = "pending"
	}

	// 根据执行模式设置状态和镜像标签
	switch t.ExecutionMode {
	case "auto":
		switch t.ScheduleType {
		case "immediate":
			t.Status = "PENDING"
			t.ImageTag = time.Now().Format("20060102150405")
		case "scheduled":
			t.Status = "SCHEDULED"
			t.ImageTag = time.Now().Format("20060102150405") // 先生成，执行时更新
		}
	case "manual":
		t.Status = "CREATED"
		t.ImageTag = "" // 手动执行时不生成，触发构建时生成
		t.BuildStatus = "ready"   // 可执行构建
		t.DeployStatus = "pending" // 等待构建完成
	}

	// 自动生成部署名时仅使用应用英文名，不再拼接 ID。
	// 同时兼容旧前端传入的"英文名-ID"默认值并归一为"英文名"。
	var app model.Application
	if err := s.db.First(&app, t.ProjectID).Error; err == nil {
		legacyByID := fmt.Sprintf("app-%d", t.ProjectID)
		legacyByENameWithID := ""
		if app.EName != "" {
			legacyByENameWithID = fmt.Sprintf("%s-%d", app.EName, t.ProjectID)
		}

		if t.DeploymentName == "" || t.DeploymentName == legacyByID || (legacyByENameWithID != "" && t.DeploymentName == legacyByENameWithID) {
			if app.EName != "" {
				t.DeploymentName = app.EName
			} else if t.DeploymentName == "" || t.DeploymentName == legacyByID {
				t.DeploymentName = "app"
			}
		}
	} else if t.DeploymentName == "" {
		t.DeploymentName = "app"
	}

	// 同步 DeployType（兼容旧流程）
	if t.DeployType == "" && t.DeployTarget != "" {
		t.DeployType = t.DeployTarget
	}

	if err := s.db.Create(t).Error; err != nil {
		return err
	}

	// 根据执行模式入队
	switch t.ExecutionMode {
	case "auto":
		switch t.ScheduleType {
		case "immediate":
			return s.queueClient.EnqueueDeployTask(t.ID)
		case "scheduled":
			return s.queueClient.EnqueueDeployTaskAt(t.ID, t.ScheduledAt)
		}
	case "manual":
		// 手动执行模式不入队，等待用户触发
		log.S().Infof("Manual task %d created, waiting for trigger", t.ID)
	}

	return nil
}

// RetryTask 重试一个失败的部署任务：将状态重置为 PENDING，清空执行字段，递增 retry_count，
// 清理旧的阶段历史记录，重新入队。
func (s *DeployService) RetryTask(taskID uint) (*model.DeployTask, error) {
	var task model.DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, fmt.Errorf("任务不存在: %w", err)
	}

	if task.Status != "FAILED" {
		return nil, fmt.Errorf("只能重试失败的任务，当前状态: %s", task.Status)
	}

	// 清理旧的阶段历史记录，避免重试后仍显示上次失败的错误信息
	if err := s.db.Where("task_id = ?", taskID).Delete(&model.StageHistory{}).Error; err != nil {
		log.S().Warnf("清理任务 %d 旧的阶段历史记录失败: %v", taskID, err)
	}

	// 使用 Updates(map) 而非 Save，确保清空字段（nil、0、""）被正确写入
	result := s.db.Model(&model.DeployTask{}).Where("id = ? AND status = ?", taskID, "FAILED").Updates(map[string]interface{}{
		"status":               "PENDING",
		"retry_count":          gorm.Expr("retry_count + 1"),
		"started_at":           nil,
		"finished_at":          nil,
		"error_message":        "",
		"jenkins_build_number": 0,
	})
	if result.Error != nil {
		return nil, fmt.Errorf("更新任务失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("只能重试失败的任务")
	}

	// 入队失败时回滚状态，避免任务孤立在 PENDING 状态
	if err := s.queueClient.EnqueueDeployTaskRetry(taskID); err != nil {
		s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"status":      "FAILED",
			"retry_count": gorm.Expr("retry_count - 1"),
		})
		return nil, fmt.Errorf("重试入队失败: %w", err)
	}

	// 重新加载以获取更新后的值
	s.db.First(&task, taskID)
	return &task, nil
}

// ProcessDeploymentTask 由 Asynq worker 调用，执行完整的 Jenkins→Registry→K8s 流程。
func (s *DeployService) ProcessDeploymentTask(ctx context.Context, taskID uint) error {
	var task model.DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return fmt.Errorf("load task %d: %w", taskID, err)
	}

	// 幂等保护：已成功则跳过（Asynq 重试时不重复执行）
	if task.Status == "SUCCESS" {
		return nil
	}

	// 检查是否为回滚任务：跳过构建，直接部署
	if task.IsRollback {
		return s.ProcessRollbackTask(ctx, taskID)
	}

	// --- PENDING stage: record that the task has started processing ---
	s.startStage(taskID, "PENDING", task.RetryCount)
	s.finishStage(taskID, "PENDING", "success", "")

	// 1. 开始构建阶段
	s.startStage(taskID, "BUILDING", task.RetryCount)
	s.updateTaskStatus(taskID, "BUILDING", "开始Jenkins构建流程", "")

	// 2. 触发Jenkins构建
	buildResult, jenkinsClient, err := s.triggerJenkinsBuild(&task)
	if err != nil {
		s.finishStage(taskID, "BUILDING", "failed", err.Error())
		s.updateTaskStatus(taskID, "FAILED", fmt.Sprintf("Jenkins构建失败: %v", err), err.Error())
		return err
	}

	// 保存 Jenkins 构建编号
	s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Update("jenkins_build_number", buildResult.BuildNumber)

	// 3. 等待构建完成（使用触发构建时的同一 Jenkins 客户端）
	if !s.waitForJenkinsBuild(ctx, jenkinsClient, buildResult.JobName, buildResult.BuildNumber) {
		errMsg := "Jenkins构建失败或超时"
		s.finishStage(taskID, "BUILDING", "failed", errMsg)
		s.updateTaskStatus(taskID, "FAILED", errMsg, errMsg)
		return fmt.Errorf("jenkins build failed or timed out: job=%s build=%d", buildResult.JobName, buildResult.BuildNumber)
	}
	s.finishStage(taskID, "BUILDING", "success", "")

	// 4. 开始推送阶段
	s.startStage(taskID, "PUSHING", task.RetryCount)
	s.updateTaskStatus(taskID, "PUSHING", "开始推送镜像到仓库", "")
	log.S().Infof("pushing image: project=%s name=%s tag=%s", task.RegistryProject, task.ImageName, task.ImageTag)

	// 5. 验证镜像推送
	if !s.verifyImageInRegistryForTask(&task) {
		errMsg := "镜像验证失败"
		s.finishStage(taskID, "PUSHING", "failed", errMsg)
		s.updateTaskStatus(taskID, "FAILED", errMsg, errMsg)
		return fmt.Errorf("registry image not found: %s/%s:%s", task.RegistryProject, task.ImageName, task.ImageTag)
	}
	s.finishStage(taskID, "PUSHING", "success", "")

	// 6. 开始部署阶段 — 根据部署目标分支
	s.startStage(taskID, "DEPLOYING", task.RetryCount)
	deployTarget := task.DeployTarget
	if deployTarget == "" {
		deployTarget = task.DeployType // 兼容旧数据
	}
	switch deployTarget {
	case "docker":
		s.updateTaskStatus(taskID, "DEPLOYING", "开始部署到Linux主机(Docker)", "")
		if err := s.deployToDocker(&task); err != nil {
			s.finishStage(taskID, "DEPLOYING", "failed", err.Error())
			s.updateTaskStatus(taskID, "FAILED", fmt.Sprintf("Docker部署失败: %v", err), err.Error())
			return err
		}
	case "k8s":
		s.updateTaskStatus(taskID, "DEPLOYING", "开始部署到K8s集群", "")
		if err := s.deployToK8s(&task); err != nil {
			s.finishStage(taskID, "DEPLOYING", "failed", err.Error())
			s.updateTaskStatus(taskID, "FAILED", fmt.Sprintf("K8s部署失败: %v", err), err.Error())
			return err
		}
	case "linux":
		s.updateTaskStatus(taskID, "DEPLOYING", "开始部署到Linux主机(文件提取+Nginx)", "")
		if err := s.deployToLinux(&task); err != nil {
			s.finishStage(taskID, "DEPLOYING", "failed", err.Error())
			s.updateTaskStatus(taskID, "FAILED", fmt.Sprintf("Linux部署失败: %v", err), err.Error())
			return err
		}
	default:
		errMsg := fmt.Sprintf("未知的部署目标: %s", deployTarget)
		s.finishStage(taskID, "DEPLOYING", "failed", errMsg)
		s.updateTaskStatus(taskID, "FAILED", errMsg, errMsg)
		return fmt.Errorf("unknown deploy target: %s", deployTarget)
	}
	s.finishStage(taskID, "DEPLOYING", "success", "")

	// 8. 部署成功
	s.updateTaskStatus(taskID, "SUCCESS", "部署成功完成", "")
	return nil
}

// triggerJenkinsBuild 触发Jenkins构建（支持平台配置注入）
// 返回构建结果和使用的 Jenkins 客户端（供 waitForJenkinsBuild 使用）
func (s *DeployService) triggerJenkinsBuild(task *model.DeployTask) (*core.JenkinsBuildResult, *core.JenkinsClient, error) {
	// 1. 根据应用ID获取关联仓库
	var app model.Application
	if err := s.db.First(&app, task.ProjectID).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to get application %d: %v", task.ProjectID, err)
	}

	var repo model.Repo
	if err := s.db.First(&repo, app.RepoID).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to get repo %d: %v", app.RepoID, err)
	}

	// 2. 获取构建模板：必须使用任务指定的模板ID
	var buildTemplate *model.BuildTemplate
	if task.BuildTemplateID != nil && *task.BuildTemplateID > 0 {
		var bt model.BuildTemplate
		if err := s.db.First(&bt, *task.BuildTemplateID).Error; err != nil {
			return nil, nil, fmt.Errorf("failed to get build template %d: %v", *task.BuildTemplateID, err)
		}
		buildTemplate = &bt
	}

	if buildTemplate == nil {
		return nil, nil, fmt.Errorf("no build template specified for task %d", task.ID)
	}

	// 3. 查找 ApplicationDeployment 以获取平台关联配置
	var appDeploy model.ApplicationDeployment
	jenkinsClient := s.jenkins // 默认使用全局 Jenkins 客户端
	var registryCredsID string
	var gitCredsID string
	var registryURL string
	var registryProject string
	var imageName string
	var imageRepoForProjectCheck *model.ImageRepository // 用于确保仓库项目存在

	// 尝试查找部署配置来获取平台关联
	if err := s.db.Where("application_id = ? AND environment_id = ?", task.ProjectID, task.EnvID).First(&appDeploy).Error; err == nil {
		// 查到了部署配置，使用平台关联数据
		credentialMode := appDeploy.CredentialMode
		if credentialMode == "" {
			credentialMode = "auto_create"
		}

		// 3a. Jenkins 平台：如果关联了，创建专用客户端
		if appDeploy.JenkinsPlatformID != nil && *appDeploy.JenkinsPlatformID > 0 {
			var jenkinsPlatform model.JenkinsPlatform
			if err := s.db.First(&jenkinsPlatform, *appDeploy.JenkinsPlatformID).Error; err == nil {
				jenkinsClient = core.NewJenkinsClient(jenkinsPlatform.URL, jenkinsPlatform.Username, jenkinsPlatform.Password)
				log.S().Infof("Using Jenkins platform %s (ID=%d) for task %d", jenkinsPlatform.Name, jenkinsPlatform.ID, task.ID)
			} else {
				log.S().Warnf("Failed to load JenkinsPlatform %d, falling back to global config: %v", *appDeploy.JenkinsPlatformID, err)
			}
		}

		// 3b. 镜像仓库：注入仓库凭据
		if appDeploy.ImageRepoID != nil && *appDeploy.ImageRepoID > 0 {
			var imageRepo model.ImageRepository
			if err := s.db.First(&imageRepo, *appDeploy.ImageRepoID).Error; err == nil {
				imageRepoForProjectCheck = &imageRepo // 保存引用，用于后续项目检查
				registryCredsID = fmt.Sprintf("zebra-registry-%d", imageRepo.ID)
				registryURL = stripURLProtocol(imageRepo.URL)
				// 注入仓库凭据到 Jenkins
				if err := jenkinsClient.CreateOrUpdateUsernamePasswordCredential(
					registryCredsID,
					imageRepo.Username,
					imageRepo.Password,
					fmt.Sprintf("ZebraOps Registry Credential (%s)", imageRepo.Name),
				); err != nil {
					// 凭据注入失败，检查是否已存在；若不存在则阻断流程
					exists, checkErr := jenkinsClient.CredentialExists(registryCredsID)
					if checkErr != nil || !exists {
						return nil, nil, fmt.Errorf("failed to inject registry credential %s and it does not exist in Jenkins: %v", registryCredsID, err)
					}
					log.S().Warnf("Registry credential injection failed but %s already exists in Jenkins, continuing: %v", registryCredsID, err)
				} else {
					log.S().Infof("Registry credential %s injected into Jenkins", registryCredsID)
				}
			} else {
				log.S().Warnf("Failed to load ImageRepository %d: %v", *appDeploy.ImageRepoID, err)
			}
		}

		if credentialMode == "manual_select" {
			if appDeploy.JenkinsCredentialID == nil || *appDeploy.JenkinsCredentialID == 0 {
				return nil, nil, fmt.Errorf("deployment is in manual_select mode but jenkins_credential_id is empty")
			}

			var selectedCred model.JenkinsCredential
			if err := s.db.First(&selectedCred, *appDeploy.JenkinsCredentialID).Error; err != nil {
				return nil, nil, fmt.Errorf("failed to load selected Jenkins credential %d: %v", *appDeploy.JenkinsCredentialID, err)
			}
			if selectedCred.Status != "active" {
				return nil, nil, fmt.Errorf("selected Jenkins credential %s is not active", selectedCred.CredentialID)
			}
			if appDeploy.JenkinsPlatformID != nil && *appDeploy.JenkinsPlatformID > 0 && selectedCred.JenkinsPlatformID != *appDeploy.JenkinsPlatformID {
				return nil, nil, fmt.Errorf("selected Jenkins credential %s does not belong to Jenkins platform %d", selectedCred.CredentialID, *appDeploy.JenkinsPlatformID)
			}

			gitCredsID = selectedCred.CredentialID
			log.S().Infof("Using selected Jenkins credential %s for task %d in manual_select mode", gitCredsID, task.ID)
		} else if appDeploy.GitPlatformID != nil && *appDeploy.GitPlatformID > 0 {
			// 3c. Git 平台：自动注入 Git 凭据
			var gitPlatform model.GitPlatform
			if err := s.db.First(&gitPlatform, *appDeploy.GitPlatformID).Error; err == nil {
				gitCredsID = fmt.Sprintf("zebra-git-%d", gitPlatform.ID)
				// 根据 AuthType 创建不同类型的凭据
				switch gitPlatform.AuthType {
				case "token":
					// 解析 AuthConfig JSON 获取 token
					var authConfig struct {
						Token string `json:"token"`
					}
					if err := json.Unmarshal([]byte(gitPlatform.AuthConfig), &authConfig); err != nil {
						log.S().Warnf("Failed to parse AuthConfig JSON for GitPlatform %d (token type): %v", gitPlatform.ID, err)
					} else if authConfig.Token == "" {
						log.S().Warnf("AuthConfig token is empty for GitPlatform %d", gitPlatform.ID)
					} else {
						if err := jenkinsClient.CreateOrUpdateSecretTextCredential(
							gitCredsID,
							authConfig.Token,
							fmt.Sprintf("ZebraOps Git Token (%s)", gitPlatform.Name),
						); err != nil {
							// 凭据注入失败，检查是否已存在；若不存在则阻断流程
							exists, checkErr := jenkinsClient.CredentialExists(gitCredsID)
							if checkErr != nil || !exists {
								return nil, nil, fmt.Errorf("failed to inject Git token credential %s and it does not exist in Jenkins: %v", gitCredsID, err)
							}
							log.S().Warnf("Git token credential injection failed but %s already exists in Jenkins, continuing: %v", gitCredsID, err)
						} else {
							log.S().Infof("Git token credential %s injected into Jenkins", gitCredsID)
						}
					}
				case "password":
					// 解析 AuthConfig JSON 获取 username/password
					var authConfig struct {
						Username string `json:"username"`
						Password string `json:"password"`
					}
					if err := json.Unmarshal([]byte(gitPlatform.AuthConfig), &authConfig); err != nil {
						log.S().Warnf("Failed to parse AuthConfig JSON for GitPlatform %d (password type): %v", gitPlatform.ID, err)
					} else {
						if err := jenkinsClient.CreateOrUpdateUsernamePasswordCredential(
							gitCredsID,
							authConfig.Username,
							authConfig.Password,
							fmt.Sprintf("ZebraOps Git Credential (%s)", gitPlatform.Name),
						); err != nil {
							// 凭据注入失败，检查是否已存在；若不存在则阻断流程
							exists, checkErr := jenkinsClient.CredentialExists(gitCredsID)
							if checkErr != nil || !exists {
								return nil, nil, fmt.Errorf("failed to inject Git credential %s and it does not exist in Jenkins: %v", gitCredsID, err)
							}
							log.S().Warnf("Git credential injection failed but %s already exists in Jenkins, continuing: %v", gitCredsID, err)
						} else {
							log.S().Infof("Git credential %s injected into Jenkins", gitCredsID)
						}
					}
				default:
					log.S().Warnf("Unsupported Git auth type %s for platform %d", gitPlatform.AuthType, gitPlatform.ID)
				}
			} else {
				log.S().Warnf("Failed to load GitPlatform %d: %v", *appDeploy.GitPlatformID, err)
			}
		}
	} else {
		log.S().Infof("No ApplicationDeployment found for app=%d env=%d, using global config fallback", task.ProjectID, task.EnvID)
	}

	// 4. Fallback：如果平台关联未设置，使用全局配置
	if registryURL == "" {
		registryURL = stripURLProtocol(s.cfg.RegistryURL)
	}
	if registryProject == "" {
		registryProject = task.RegistryProject
	}
	if imageName == "" {
		imageName = task.ImageName
	}
	// 兜底：确保 IMAGE_NAME 参数不为空，防止 Jenkins post 块中
	// 因 Groovy 变量绑定缺失抛出 MissingPropertyException
	if imageName == "" {
		imageName = strings.ToLower(strings.ReplaceAll(app.EName, " ", "-"))
	}
	// Docker 镜像名必须全小写
	imageName = strings.ToLower(imageName)
	if registryCredsID == "" {
		// 未配置平台关联时，使用配置中的默认凭据 ID
		registryCredsID = s.cfg.JenkinsDefaultRegistryCred
		if registryCredsID == "" {
			registryCredsID = "registry-creds"
		}
	}
	if gitCredsID == "" {
		gitCredsID = s.cfg.JenkinsDefaultGitCred
		if gitCredsID == "" {
			gitCredsID = "gitlab_user_orange"
		}
	}

	// 4b. 确保镜像仓库项目/命名空间存在（Harbor/ACR 需要预先创建项目）
	if imageRepoForProjectCheck != nil {
		registryClient := core.NewRegistryClientFromRepo(imageRepoForProjectCheck)
		if err := registryClient.EnsureProjectExists(registryProject); err != nil {
			log.S().Warnf("Failed to ensure registry project %s exists: %v, continuing (project may already exist)", registryProject, err)
			// 不阻断流程——如果项目创建失败但项目实际上已存在，Jenkins push 仍能成功
		}
	} else {
		// 全局配置 fallback：使用 V2 适配器（V2 不需要项目创建）
		registryClient := core.NewV2RegistryAdapter(s.cfg.RegistryURL, s.cfg.RegistryUser, s.cfg.RegistryPass)
		if err := registryClient.EnsureProjectExists(registryProject); err != nil {
			log.S().Warnf("Failed to ensure registry project %s exists (global config): %v", registryProject, err)
		}
	}

	// 5. 检查 Jenkins Job 是否存在，不存在则创建；已存在则更新配置
	jobExists, err := jenkinsClient.CheckJobExists(task.JenkinsJobName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check job existence: %v", err)
	}

	jobConfig := s.generateJobConfig(buildTemplate, task.GitRef, repo.RepoURL, task.ImageTag)
	if !jobExists {
		fmt.Fprintf(os.Stdout, "Jenkins Job %s does not exist, creating...\n", task.JenkinsJobName)
		if err := jenkinsClient.CreateJob(task.JenkinsJobName, jobConfig); err != nil {
			return nil, nil, fmt.Errorf("failed to create job: %v", err)
		}
	} else {
		// Job 已存在，更新其配置以确保参数列表和 pipeline 内容与模板一致
		if err := jenkinsClient.UpdateJob(task.JenkinsJobName, jobConfig); err != nil {
			log.S().Warnf("Failed to update Jenkins job %s config: %v, continuing with existing config", task.JenkinsJobName, err)
		} else {
			log.S().Infof("Jenkins job %s config updated to match latest template", task.JenkinsJobName)
		}
	}

	// 6. 鉴权
	if err := jenkinsClient.Authenticate(); err != nil {
		return nil, nil, fmt.Errorf("Jenkins authentication failed: %v", err)
	}
	fmt.Println("开始触发Jenkins构建")

	// 7. 构建参数——包含平台注入的非敏感数据和凭据 ID
	params := map[string]string{
		"TARGET_BRANCH":     task.GitRef,
		"Repo_URL":          repo.RepoURL,
		"Tag":               task.ImageTag,
		"REGISTRY_URL":      registryURL,
		"REGISTRY_PROJECT":  registryProject,
		"IMAGE_NAME":        imageName,
		"REGISTRY_CREDS_ID":   registryCredsID,
		"GIT_CREDS_ID":        gitCredsID,
		"DEPLOY_TARGET":       task.DeployTarget,
		"DOCKERFILE_CONTENT":  buildTemplate.Dockerfile,
	}

	result, err := jenkinsClient.BuildJob(task.JenkinsJobName, params)
	if err != nil {
		return nil, nil, err
	}
	return result, jenkinsClient, nil
}

// waitForJenkinsBuild 等待Jenkins构建完成（使用触发构建时的同一 Jenkins 客户端）
func (s *DeployService) waitForJenkinsBuild(ctx context.Context, jenkinsClient *core.JenkinsClient, jobName string, buildNumber int) bool {
	// 使用配置值
	buildWaitTimeout := s.cfg.JenkinsBuildWaitTimeout
	if buildWaitTimeout == 0 {
		buildWaitTimeout = 10 * time.Minute
	}
	pollInterval := s.cfg.JenkinsBuildPollInterval
	if pollInterval == 0 {
		pollInterval = 10 * time.Second
	}

	timeout := time.After(buildWaitTimeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-timeout:
			return false
		case <-ticker.C:
			status, err := jenkinsClient.GetBuildStatus(jobName, buildNumber)
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

// verifyImageInRegistry 验证镜像仓库中的镜像是否存在
// 使用部署配置关联的仓库客户端，如无关联则使用全局默认
func (s *DeployService) verifyImageInRegistry(project, imageName, tag string) bool {
	return s.registry.VerifyImageExists(project, imageName, tag)
}

// getRegistryClientForTask 根据部署任务关联的应用配置获取动态镜像仓库客户端
func (s *DeployService) getRegistryClientForTask(task *model.DeployTask) core.RegistryClient {
	var appDeploy model.ApplicationDeployment
	if err := s.db.Where("application_id = ? AND environment_id = ?", task.ProjectID, task.EnvID).First(&appDeploy).Error; err == nil {
		if appDeploy.ImageRepoID != nil && *appDeploy.ImageRepoID > 0 {
			var imageRepo model.ImageRepository
			if err := s.db.First(&imageRepo, *appDeploy.ImageRepoID).Error; err == nil {
				return core.NewRegistryClient(imageRepo.Type, imageRepo.URL, imageRepo.Username, imageRepo.Password)
			}
			log.S().Warnf("Failed to load ImageRepository %d for task %d: %v", *appDeploy.ImageRepoID, task.ID, err)
		}
	}
	// Fallback: 全局配置
	return s.registry
}

// verifyImageInRegistryForTask 使用动态客户端验证镜像（按部署配置关联的仓库验证）
func (s *DeployService) verifyImageInRegistryForTask(task *model.DeployTask) bool {
	client := s.getRegistryClientForTask(task)
	return client.VerifyImageExists(task.RegistryProject, task.ImageName, task.ImageTag)
}

// deployToDocker 通过SSH部署docker-compose到Linux主机
func (s *DeployService) deployToDocker(task *model.DeployTask) error {
	// 1. 获取目标服务器信息
	server, err := s.serverRepo.GetByID(task.ServerID)
	if err != nil {
		return fmt.Errorf("获取服务器 %d 失败: %v", task.ServerID, err)
	}

	// 2. 获取部署模板（必须指定模板ID且类型为docker）
	if task.DeploymentTemplateID == nil || *task.DeploymentTemplateID == 0 {
		return fmt.Errorf("Docker部署任务 %d 未指定部署模板", task.ID)
	}
	var deploymentTemplate model.DeploymentTemplate
	if err := s.db.First(&deploymentTemplate, *task.DeploymentTemplateID).Error; err != nil {
		return fmt.Errorf("获取部署模板 %d 失败: %v", *task.DeploymentTemplateID, err)
	}
	if deploymentTemplate.TemplateType != "docker" {
		return fmt.Errorf("模板 %d 类型为 '%s'，期望 'docker'", deploymentTemplate.ID, deploymentTemplate.TemplateType)
	}

	// 3. 渲染模板内容
	// 3. 渲染模板内容（包含自定义变量）
	customVars := map[string]interface{}{}
	if deploymentTemplate.Variables != "" {
		if err := json.Unmarshal([]byte(deploymentTemplate.Variables), &customVars); err != nil {
			log.S().Warnf("Failed to parse deployment template variables for task %d: %v", task.ID, err)
		}
	}
	renderedYAML := s.renderDockerTemplate(deploymentTemplate.Content, task, customVars)

	// 4. 创建SSH客户端
	sshClient, err := s.createSSHClientFromServer(server)
	if err != nil {
		return fmt.Errorf("SSH连接 %s 失败: %v", server.Host, err)
	}
	defer sshClient.Close()

	// 5. 创建远端部署目录
	deployBase := s.cfg.DeployBasePath
	if deployBase == "" {
		deployBase = "/opt/zebra-deploy"
	}
	composeDir := fmt.Sprintf("%s/%s", deployBase, task.DeploymentName)
	_, _, exitCode, _ := sshClient.RunCommandOutput(fmt.Sprintf("mkdir -p %s", composeDir))
	if exitCode != 0 {
		return fmt.Errorf("创建部署目录 %s 失败 (exit=%d)", composeDir, exitCode)
	}

	// 6. 上传渲染后的docker-compose.yml
	composePath := fmt.Sprintf("%s/docker-compose.yml", composeDir)
	if err := sshClient.UploadFile(composePath, []byte(renderedYAML)); err != nil {
		return fmt.Errorf("上传 docker-compose.yml 失败: %v", err)
	}
	log.S().Infof("uploaded docker-compose.yml to %s:%s", server.Host, composePath)

	// 7. 拉取镜像
	_, stderr, exitCode, _ := sshClient.RunCommandOutput(fmt.Sprintf("cd %s && docker-compose pull 2>&1", composeDir))
	if exitCode != 0 {
		return fmt.Errorf("docker-compose pull 失败 (exit=%d): %s", exitCode, stderr)
	}
	log.S().Infof("docker-compose pull succeeded on %s", server.Host)

	// 8. 启动服务
	_, stderr, exitCode, _ = sshClient.RunCommandOutput(fmt.Sprintf("cd %s && docker-compose up -d 2>&1", composeDir))
	if exitCode != 0 {
		return fmt.Errorf("docker-compose up -d 失败 (exit=%d): %s", exitCode, stderr)
	}
	log.S().Infof("docker-compose up -d succeeded on %s", server.Host)

	// 9. 保存compose路径到任务记录
	s.db.Model(&model.DeployTask{}).Where("id = ?", task.ID).Update("docker_compose_path", composePath)

	return nil
}

// renderDockerTemplate 渲染Docker部署模板（docker-compose YAML）
func (s *DeployService) renderDockerTemplate(templateContent string, task *model.DeployTask, customVars map[string]interface{}) string {
	var projectName string
	var app model.Application
	if err := s.db.First(&app, task.ProjectID).Error; err == nil {
		projectName = app.EName
	} else {
		projectName = fmt.Sprintf("project-%d", task.ProjectID)
	}

	rendered := templateContent
	rendered = strings.ReplaceAll(rendered, "\\n", "\n")
	rendered = strings.ReplaceAll(rendered, "\r\n", "\n")
	rendered = strings.ReplaceAll(rendered, "\r", "\n")
	rendered = strings.ReplaceAll(rendered, "{{IMAGE_TAG}}", task.ImageTag)
	rendered = strings.ReplaceAll(rendered, "{{PROJECT_NAME}}", projectName)
	rendered = strings.ReplaceAll(rendered, "{{ENV_NAME}}", fmt.Sprintf("env-%d", task.EnvID))
	rendered = strings.ReplaceAll(rendered, "{{DEPLOYMENT_NAME}}", task.DeploymentName)

	// 替换自定义变量
	for key, value := range customVars {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", fmt.Sprintf("%v", value))
	}

	return rendered
}

// deployToLinux 通过SSH部署前端静态文件到Linux主机(Nginx代理)
// 使用 Docker 镜像提取方式：pull → create → cp → rm → nginx config → reload
func (s *DeployService) deployToLinux(task *model.DeployTask) error {
	// 1. 获取目标服务器信息
	server, err := s.serverRepo.GetByID(task.ServerID)
	if err != nil {
		return fmt.Errorf("获取服务器 %d 失败: %v", task.ServerID, err)
	}

	// 2. 获取部署模板（类型必须为 linux）
	if task.DeploymentTemplateID == nil || *task.DeploymentTemplateID == 0 {
		return fmt.Errorf("Linux部署任务 %d 未指定部署模板(Nginx配置)", task.ID)
	}
	var deploymentTemplate model.DeploymentTemplate
	if err := s.db.First(&deploymentTemplate, *task.DeploymentTemplateID).Error; err != nil {
		return fmt.Errorf("获取部署模板 %d 失败: %v", *task.DeploymentTemplateID, err)
	}
	if deploymentTemplate.TemplateType != "linux" {
		return fmt.Errorf("模板 %d 类型为 '%s'，期望 'linux'", deploymentTemplate.ID, deploymentTemplate.TemplateType)
	}

	// 3. 创建SSH客户端
	sshClient, err := s.createSSHClientFromServer(server)
	if err != nil {
		return fmt.Errorf("SSH连接 %s 失败: %v", server.Host, err)
	}
	defer sshClient.Close()

	// 4. Docker 镜像提取: pull → create → cp → rm
	fullImageRef := fmt.Sprintf("%s/%s:%s", task.RegistryProject, task.ImageName, task.ImageTag)

	// 4a. Pull image
	log.S().Infof("docker pull %s on %s", fullImageRef, server.Host)
	_, stderr, exitCode, _ := sshClient.RunCommandOutput(fmt.Sprintf("docker pull %s 2>&1", fullImageRef))
	if exitCode != 0 {
		return fmt.Errorf("docker pull 失败 (exit=%d): %s", exitCode, stderr)
	}

	// 4b. Create temporary container (do not start)
	containerName := fmt.Sprintf("zebra-extract-%d", task.ID)
	log.S().Infof("docker create --name %s %s", containerName, fullImageRef)
	_, stderr, exitCode, _ = sshClient.RunCommandOutput(fmt.Sprintf("docker create --name %s %s 2>&1", containerName, fullImageRef))
	if exitCode != 0 {
		return fmt.Errorf("docker create 失败 (exit=%d): %s", exitCode, stderr)
	}

	// 4c. 从模板 variables 中提取容器内源路径，默认 /app/dist
	containerSourcePath := "/app/dist"
	if deploymentTemplate.Variables != "" {
		var vars map[string]interface{}
		if err := yaml.Unmarshal([]byte(deploymentTemplate.Variables), &vars); err == nil {
			if sp, ok := vars["source_path"].(string); ok && sp != "" {
				containerSourcePath = sp
			}
		}
	}

	// 4d. 确保目标目录存在
	deployPath := task.DeployPath
	if deployPath == "" {
		deployBase := s.cfg.DeployBasePath
		if deployBase == "" {
			deployBase = "/opt/zebra-deploy"
		}
		deployPath = fmt.Sprintf("%s/%s", deployBase, task.DeploymentName)
	}
	log.S().Infof("mkdir -p %s", deployPath)
	_, _, exitCode, _ = sshClient.RunCommandOutput(fmt.Sprintf("mkdir -p %s 2>&1", deployPath))

	// 4e. Copy files from container to deploy path
	log.S().Infof("docker cp %s:%s/. %s/", containerName, containerSourcePath, deployPath)
	_, stderr, exitCode, _ = sshClient.RunCommandOutput(
		fmt.Sprintf("docker cp %s:%s/. %s/ 2>&1", containerName, containerSourcePath, deployPath))
	if exitCode != 0 {
		// Cleanup container before returning error
		sshClient.RunCommandOutput(fmt.Sprintf("docker rm %s 2>&1", containerName))
		return fmt.Errorf("docker cp 失败 (exit=%d): %s", exitCode, stderr)
	}

	// 4f. Remove temporary container
	log.S().Infof("docker rm %s", containerName)
	_, stderr, exitCode, _ = sshClient.RunCommandOutput(fmt.Sprintf("docker rm %s 2>&1", containerName))
	if exitCode != 0 {
		log.S().Warnf("docker rm %s 失败 (exit=%d): %s (非致命错误)", containerName, exitCode, stderr)
	}

	// 5. 渲染 Nginx 配置模板并上传
	// 5. 渲染 Nginx 配置模板并上传（包含自定义变量）
	linuxCustomVars := map[string]interface{}{}
	if deploymentTemplate.Variables != "" {
		if err := json.Unmarshal([]byte(deploymentTemplate.Variables), &linuxCustomVars); err != nil {
			log.S().Warnf("Failed to parse deployment template variables for task %d: %v", task.ID, err)
		}
	}
	renderedNginxConfig := s.renderLinuxTemplate(deploymentTemplate.Content, task, linuxCustomVars)
	nginxConfDir := s.cfg.NginxConfPath
	if nginxConfDir == "" {
		nginxConfDir = "/etc/nginx/conf.d"
	}
	nginxConfigPath := fmt.Sprintf("%s/%s.conf", nginxConfDir, task.DeploymentName)
	log.S().Infof("upload nginx config to %s:%s", server.Host, nginxConfigPath)
	if err := sshClient.UploadFile(nginxConfigPath, []byte(renderedNginxConfig)); err != nil {
		return fmt.Errorf("上传 Nginx 配置失败: %v", err)
	}

	// 6. 测试并重载 Nginx
	log.S().Infof("nginx -t on %s", server.Host)
	_, stderr, exitCode, _ = sshClient.RunCommandOutput("nginx -t 2>&1")
	if exitCode != 0 {
		return fmt.Errorf("nginx -t 测试失败 (exit=%d): %s", exitCode, stderr)
	}
	log.S().Infof("nginx -s reload on %s", server.Host)
	_, stderr, exitCode, _ = sshClient.RunCommandOutput("nginx -s reload 2>&1")
	if exitCode != 0 {
		return fmt.Errorf("nginx -s reload 失败 (exit=%d): %s", exitCode, stderr)
	}

	return nil
}

// renderLinuxTemplate 渲染Linux/Nginx部署模板
func (s *DeployService) renderLinuxTemplate(templateContent string, task *model.DeployTask, customVars map[string]interface{}) string {
	var projectName string
	var app model.Application
	if err := s.db.First(&app, task.ProjectID).Error; err == nil {
		projectName = app.EName
	} else {
		projectName = fmt.Sprintf("project-%d", task.ProjectID)
	}

	rendered := templateContent
	rendered = strings.ReplaceAll(rendered, "\\n", "\n")
	rendered = strings.ReplaceAll(rendered, "\r\n", "\n")
	rendered = strings.ReplaceAll(rendered, "\r", "\n")
	rendered = strings.ReplaceAll(rendered, "{{IMAGE_TAG}}", task.ImageTag)
	rendered = strings.ReplaceAll(rendered, "{{PROJECT_NAME}}", projectName)
	rendered = strings.ReplaceAll(rendered, "{{ENV_NAME}}", fmt.Sprintf("env-%d", task.EnvID))
	rendered = strings.ReplaceAll(rendered, "{{DEPLOYMENT_NAME}}", task.DeploymentName)
	rendered = strings.ReplaceAll(rendered, "{{DEPLOY_PATH}}", task.DeployPath)

	// 替换自定义变量
	for key, value := range customVars {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", fmt.Sprintf("%v", value))
	}

	return rendered
}

// createSSHClientFromServer 从Server模型创建SSHClient（支持密码和密钥认证）
func (s *DeployService) createSSHClientFromServer(server *model.Server) (*sshclient.SSHClient, error) {
	var authMethods []ssh.AuthMethod

	if server.AuthType == "key" {
		signer, err := ssh.ParsePrivateKey([]byte(server.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		authMethods = append(authMethods, ssh.Password(server.Password))
	}

	return sshclient.NewSSHClientWithAuth(server.Host, server.Port, server.Username, authMethods)
}
func (s *DeployService) deployToK8s(task *model.DeployTask) error {
	// 2. 根据环境配置获取集群信息
	var cluster model.K8SCluster
	if err := s.db.First(&cluster, task.K8sClusterID).Error; err != nil {
		return fmt.Errorf("failed to get k8s cluster: %v", err)
	}

	// 3. 创建K8s客户端
	clientset, err := s.getK8sClientByCluster(cluster)
	if err != nil {
		return err
	}

	// 4. 根据应用ID获取关联仓库和部署模板
	var app model.Application
	if err := s.db.First(&app, task.ProjectID).Error; err != nil {
		return fmt.Errorf("failed to get application %d: %v", task.ProjectID, err)
	}

	var repo model.Repo
	if err := s.db.First(&repo, app.RepoID).Error; err != nil {
		return fmt.Errorf("failed to get repo %d: %v", app.RepoID, err)
	}

	// 5. 获取部署模板：必须使用任务指定的模板ID
	var deploymentTemplate *model.DeploymentTemplate
	if task.DeploymentTemplateID != nil && *task.DeploymentTemplateID > 0 {
		var dt model.DeploymentTemplate
		if err := s.db.First(&dt, *task.DeploymentTemplateID).Error; err != nil {
			return fmt.Errorf("failed to get deployment template %d: %v", *task.DeploymentTemplateID, err)
		}
		deploymentTemplate = &dt
	}

	if deploymentTemplate == nil {
		return fmt.Errorf("no deployment template specified for task %d", task.ID)
	}

	// 6. 解析模板内容并进行参数替换（包含自定义变量）
	customVars := map[string]interface{}{}
	if deploymentTemplate.Variables != "" {
		if err := json.Unmarshal([]byte(deploymentTemplate.Variables), &customVars); err != nil {
			log.S().Warnf("Failed to parse deployment template variables for task %d: %v", task.ID, err)
		}
	}
	renderedYAML := s.renderTemplate(deploymentTemplate.Content, task, customVars)

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
// 替换内置占位符（IMAGE_TAG/NAMESPACE/PROJECT_NAME/ENV_NAME）以及
// DeploymentTemplate.Variables 中定义的自定义占位符
func (s *DeployService) renderTemplate(templateContent string, task *model.DeployTask, customVars map[string]interface{}) string {
	// 获取项目名称（应用的英文名）
	var projectName string
	var app model.Application
	if err := s.db.First(&app, task.ProjectID).Error; err == nil {
		projectName = app.EName
	} else {
		projectName = fmt.Sprintf("calc-api-project-%d", task.ProjectID)
	}

	// 先处理换行符，再替换占位符
	rendered := templateContent

	// 多步骤处理换行符
	rendered = strings.ReplaceAll(rendered, "\\n", "\n")
	rendered = strings.ReplaceAll(rendered, "\r\n", "\n")
	rendered = strings.ReplaceAll(rendered, "\r", "\n")

	// 替换内置模板占位符，保持YAML格式
	rendered = strings.ReplaceAll(rendered, "{{IMAGE_TAG}}", task.ImageTag)
	rendered = strings.ReplaceAll(rendered, "{{IMAGE_NAME}}", task.ImageName)
	rendered = strings.ReplaceAll(rendered, "{{NAMESPACE}}", task.K8sNamespace)
	rendered = strings.ReplaceAll(rendered, "{{PROJECT_NAME}}", projectName)
	rendered = strings.ReplaceAll(rendered, "{{ENV_NAME}}", fmt.Sprintf("env-%d", task.EnvID))

	// 替换自定义变量（从 DeploymentTemplate.Variables JSON 解析）
	for key, value := range customVars {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", fmt.Sprintf("%v", value))
	}

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

		// 关键：在这里统一转换所有 key
		rawObj = s.convertMapToStringKey(rawObj).(map[string]interface{})

		kind := s.safeExtractValue(rawObj, "kind")
		if kind == "" {
			return fmt.Errorf("资源缺少必要字段 (Kind: %s)", kind)
		}

		obj := &unstructured.Unstructured{Object: rawObj}

		// 统一规范化 K8s 资源名称：K8s 要求小写 RFC 1123 子域名
		s.sanitizeK8sResourceName(rawObj)

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

// sanitizeK8sResourceName 将 K8s 资源中所有名称字段转换为合规的小写形式。
// K8s 要求资源名称符合 RFC 1123 子域名/标签规范：小写字母、数字、'-' 或 '.'，首尾为字母数字。
// 这包括 metadata.name、namespace、container name、configMapRef、secretRef 等嵌套引用。
func (s *DeployService) sanitizeK8sResourceName(rawObj map[string]interface{}) {
	// 1. metadata.name / metadata.namespace
	if metadata, ok := rawObj["metadata"].(map[string]interface{}); ok {
		s.lowercaseMapKey(metadata, "name")
		s.lowercaseMapKey(metadata, "namespace")
	}

	// 2. spec 下的嵌套引用
	spec, _ := rawObj["spec"].(map[string]interface{})
	if spec == nil {
		return
	}

	// spec.serviceAccountName
	s.lowercaseMapKey(spec, "serviceAccountName")

	// spec.template.spec.containers[]
	template, _ := spec["template"].(map[string]interface{})
	if template == nil {
		return
	}
	templateSpec, _ := template["spec"].(map[string]interface{})
	if templateSpec == nil {
		return
	}

	containers, _ := templateSpec["containers"].([]interface{})
	for _, c := range containers {
		container, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		// container name
		s.lowercaseMapKey(container, "name")

		// envFrom[].configMapRef.name / secretRef.name
		s.sanitizeEnvFrom(container)

		// env[].valueFrom.configMapKeyRef.name / secretKeyRef.name
		s.sanitizeEnv(container)
	}

	// spec.template.spec.volumes[].configMap.name / secret.secretName
	volumes, _ := templateSpec["volumes"].([]interface{})
	for _, v := range volumes {
		vol, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if cm, ok := vol["configMap"].(map[string]interface{}); ok {
			s.lowercaseMapKey(cm, "name")
		}
		if secret, ok := vol["secret"].(map[string]interface{}); ok {
			s.lowercaseMapKey(secret, "secretName")
		}
	}
}

// sanitizeEnvFrom 规范化 envFrom 中的 configMapRef/secretRef 名称
func (s *DeployService) sanitizeEnvFrom(container map[string]interface{}) {
	envFrom, _ := container["envFrom"].([]interface{})
	for _, ef := range envFrom {
		item, ok := ef.(map[string]interface{})
		if !ok {
			continue
		}
		if ref, ok := item["configMapRef"].(map[string]interface{}); ok {
			s.lowercaseMapKey(ref, "name")
		}
		if ref, ok := item["secretRef"].(map[string]interface{}); ok {
			s.lowercaseMapKey(ref, "name")
		}
	}
}

// sanitizeEnv 规范化 env 中 valueFrom.configMapKeyRef/secretKeyRef 名称
func (s *DeployService) sanitizeEnv(container map[string]interface{}) {
	envVars, _ := container["env"].([]interface{})
	for _, ev := range envVars {
		envVar, ok := ev.(map[string]interface{})
		if !ok {
			continue
		}
		vf, ok := envVar["valueFrom"].(map[string]interface{})
		if !ok {
			continue
		}
		if ref, ok := vf["configMapKeyRef"].(map[string]interface{}); ok {
			s.lowercaseMapKey(ref, "name")
		}
		if ref, ok := vf["secretKeyRef"].(map[string]interface{}); ok {
			s.lowercaseMapKey(ref, "name")
		}
	}
}

// lowercaseMapKey 将 map 中指定 key 的字符串值转为小写
func (s *DeployService) lowercaseMapKey(m map[string]interface{}, key string) {
	if val, ok := m[key].(string); ok && val != "" {
		sanitized := strings.ToLower(val)
		if sanitized != val {
			m[key] = sanitized
			log.S().Infof("K8s 名称已规范化: %s → %s", val, sanitized)
		}
	}
}

// convertMapToStringKey 递归将所有 map 的 interface{} key 转为 string
func (s *DeployService) convertMapToStringKey(input interface{}) interface{} {
	switch v := input.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(v))
		for key, val := range v {
			m[fmt.Sprintf("%v", key)] = s.convertMapToStringKey(val)
		}
		return m
	case map[string]interface{}:
		m := make(map[string]interface{}, len(v))
		for key, val := range v {
			m[key] = s.convertMapToStringKey(val)
		}
		return m
	case []interface{}:
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
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// extractValueFromMetadata 从 metadata 中提取指定键的值
func (s *DeployService) extractValueFromMetadata(rawObj map[string]interface{}, key string) string {
	if metadata, exists := rawObj["metadata"]; exists && metadata != nil {
		fmt.Println("metadata:", metadata)
		switch v := metadata.(type) {
		case map[string]interface{}:
			return s.safeExtractValue(v, key)
		case map[interface{}]interface{}:
			for metaKey, value := range v {
				if keyStr, ok := metaKey.(string); ok && keyStr == key {
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
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if _, exists := itemMap[key]; exists {
						return s.safeExtractValue(itemMap, key)
					}
				}
			}
			return ""
		case string:
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
	_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), nsName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
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
		return fmt.Errorf("failed to get namespace %s: %v", nsName, err)
	}

	log.S().Infof("Namespace %s already exists, skipping creation", nsName)
	return nil
}

// applyConfigMap 使用 Server-Side Apply 创建或更新 ConfigMap
func (s *DeployService) applyConfigMap(name string, ns string, clientset *kubernetes.Clientset, obj *unstructured.Unstructured) error {
	configMap := &corev1.ConfigMap{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), configMap); err != nil {
		log.S().Errorf("Failed to convert unstructured object to ConfigMap: %v", err)
		return fmt.Errorf("failed to convert unstructured object to ConfigMap: %v", err)
	}

	if configMap.Namespace == "" {
		configMap.Namespace = ns
	}
	if configMap.Name == "" {
		configMap.Name = name
	}

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

	applyConfig := corev1apply.Service(service.Name, service.Namespace).
		WithSpec(corev1apply.ServiceSpec().
			WithPorts(lo.Map(service.Spec.Ports, func(p corev1.ServicePort, _ int) *corev1apply.ServicePortApplyConfiguration {
				return corev1apply.ServicePort().WithPort(p.Port).WithTargetPort(p.TargetPort)
			})...).
			WithSelector(service.Spec.Selector))

	_, err := clientset.CoreV1().Services(service.Namespace).Apply(context.TODO(), applyConfig, metav1.ApplyOptions{
		FieldManager: "zebra-cicd-controller",
		Force:        true,
	})
	return err
}

// applyDeployment 使用 Server-Side Apply 部署 Deployment
func (s *DeployService) applyDeployment(clientset *kubernetes.Clientset, obj *unstructured.Unstructured, task *model.DeployTask) error {
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
		replicas = 1
	}

	matchExpressions := lo.Map(deployment.Spec.Selector.MatchExpressions,
		func(expr metav1.LabelSelectorRequirement, index int) *metav1apply.LabelSelectorRequirementApplyConfiguration {
			return metav1apply.LabelSelectorRequirement().
				WithKey(expr.Key).
				WithOperator(expr.Operator).
				WithValues(expr.Values...)
		})

	selector := metav1apply.LabelSelector().
		WithMatchLabels(deployment.Spec.Selector.MatchLabels).
		WithMatchExpressions(matchExpressions...)

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
					WithContainers(containers...),
				),
			),
		)

	_, err := clientset.AppsV1().Deployments(deployment.Namespace).Apply(context.TODO(), applyConfig, metav1.ApplyOptions{
		FieldManager: "zebra-cicd-controller",
		Force:        true,
	})
	if err != nil {
		return fmt.Errorf("failed to apply Deployment %s in namespace %s: %v", deployment.Name, deployment.Namespace, err)
	}

	log.S().Infof("Applied Deployment: %s in namespace: %s", deployment.Name, deployment.Namespace)
	return nil
}

// updateTaskStatus 更新任务状态
func (s *DeployService) updateTaskStatus(taskID uint, status, message, errorMsg string) {
	now := time.Now()
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": now,
	}

	if status == "BUILDING" {
		updates["started_at"] = now
	}

	if status == "SUCCESS" || status == "FAILED" {
		updates["finished_at"] = now
	}
	if errorMsg != "" {
		updates["error_message"] = errorMsg
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

// GetTaskStages returns the StageHistory records for the latest execution round of a deploy task.
// It uses retry_count to filter, but falls back to returning the latest batch by creation time
// if retry_count is not yet populated (backward compatibility).
func (s *DeployService) GetTaskStages(taskID uint) ([]model.StageHistory, error) {
	// Try to get records by latest retry_count
	maxRetry, err := s.stageHistoryRepo.GetLatestRetryCount(taskID)
	if err != nil {
		return nil, err
	}

	// If we have retry_count data, use it
	if maxRetry > 0 {
		return s.stageHistoryRepo.GetByTaskIDAndRetryCount(taskID, maxRetry)
	}

	// Fallback: all records have retry_count=0 (pre-migration data).
	// In this case, we can't distinguish rounds by retry_count.
	// Return the latest batch by only including records created after the most recent PENDING stage.
	// This assumes each execution round starts with a PENDING stage.
	var allStages []model.StageHistory
	allStages, err = s.stageHistoryRepo.GetByTaskID(taskID)
	if err != nil {
		return nil, err
	}

	if len(allStages) == 0 {
		return allStages, nil
	}

	// Find the last PENDING stage — it marks the start of the latest execution round
	lastPendingIdx := -1
	for i, st := range allStages {
		if st.Stage == "PENDING" {
			lastPendingIdx = i
		}
	}

	if lastPendingIdx >= 0 {
		// Return only records from the last execution round onwards
		return allStages[lastPendingIdx:], nil
	}

	// No PENDING stage found — return all records
	return allStages, nil
}

// ListTasks 分页查询部署任务列表
func (s *DeployService) ListTasks(status string, projectID uint, envID uint, department string, page, size int) ([]model.DeployTask, int64, error) {
	db := s.db.Model(&model.DeployTask{})

	if status != "" {
		db = db.Where("deploy_tasks.status = ?", status)
	}
	if projectID > 0 {
		db = db.Where("deploy_tasks.project_id = ?", projectID)
	}
	if envID > 0 {
		db = db.Where("deploy_tasks.env_id = ?", envID)
	}

	// 按部门过滤需要JOIN Application和Repo表
	if department != "" {
		db = db.Joins("JOIN applications ON applications.id = deploy_tasks.project_id").
			Joins("JOIN repos ON repos.id = applications.repo_id").
			Where("repos.repo_department = ?", department)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []model.DeployTask
	offset := (page - 1) * size
	if err := db.Order("deploy_tasks.id DESC").Offset(offset).Limit(size).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// DeleteTask 删除部署任务，同时尝试删除关联的Jenkins Job
func (s *DeployService) DeleteTask(id uint) error {
	var task model.DeployTask
	if err := s.db.First(&task, id).Error; err != nil {
		return err
	}

	// 如果有关联 Jenkins Job，尝试删除
	if task.JenkinsJobName != "" && s.jenkins != nil {
		if err := s.jenkins.DeleteJob(task.JenkinsJobName); err != nil {
			log.S().Warnf("failed to delete Jenkins job '%s': %v", task.JenkinsJobName, err)
		}
	}

	return s.db.Delete(&model.DeployTask{}, id).Error
}

// BatchDeleteTasks 批量删除部署任务，同时尝试删除关联的Jenkins Job
func (s *DeployService) BatchDeleteTasks(ids []uint) error {
	var tasks []model.DeployTask
	if err := s.db.Where("id IN ?", ids).Find(&tasks).Error; err != nil {
		return err
	}

	for _, task := range tasks {
		if task.JenkinsJobName != "" && s.jenkins != nil {
			if err := s.jenkins.DeleteJob(task.JenkinsJobName); err != nil {
				log.S().Warnf("failed to delete Jenkins job '%s': %v", task.JenkinsJobName, err)
			}
		}
	}

	return s.db.Where("id IN ?", ids).Delete(&model.DeployTask{}).Error
}

// TemplatesForTask 创建任务时可选的模板信息
type TemplatesForTask struct {
	BuildTemplates      []model.BuildTemplate      `json:"build_templates"`
	DeploymentTemplates []model.DeploymentTemplate `json:"deployment_templates"`
}

// GetAvailableTemplatesForTask 根据应用ID和环境ID获取可用的构建/部署模板
// 模板与应用关联，优先返回关联模板，若无关联则返回所有模板
func (s *DeployService) GetAvailableTemplatesForTask(appID, envID uint) (*TemplatesForTask, error) {
	result := &TemplatesForTask{
		BuildTemplates:      make([]model.BuildTemplate, 0),
		DeploymentTemplates: make([]model.DeploymentTemplate, 0),
	}

	// 优先通过应用关联获取模板
	var app model.Application
	if err := s.db.Preload("BuildTemplates").Preload("DeploymentTemplates").First(&app, appID).Error; err != nil {
		return nil, fmt.Errorf("application %d not found: %v", appID, err)
	}

	if len(app.BuildTemplates) > 0 {
		result.BuildTemplates = app.BuildTemplates
	} else {
		// 无关联则返回所有构建模板
		var allBuilds []model.BuildTemplate
		if err := s.db.Order("id ASC").Find(&allBuilds).Error; err != nil {
			return nil, fmt.Errorf("failed to list build templates: %v", err)
		}
		result.BuildTemplates = allBuilds
	}

	if len(app.DeploymentTemplates) > 0 {
		result.DeploymentTemplates = app.DeploymentTemplates
	} else {
		// 无关联则返回所有激活的部署模板
		var allDeploys []model.DeploymentTemplate
		if err := s.db.Where("status = ?", "active").Order("id ASC").Find(&allDeploys).Error; err != nil {
			return nil, fmt.Errorf("failed to list deployment templates: %v", err)
		}
		result.DeploymentTemplates = allDeploys
	}

	return result, nil
}

// GetTaskConsole 获取任务的 Jenkins 控制台输出
func (s *DeployService) GetTaskConsole(taskID uint) (string, error) {
	var task model.DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return "", fmt.Errorf("task %d not found: %v", taskID, err)
	}

	if task.JenkinsJobName == "" || task.JenkinsBuildNumber <= 0 {
		return "", fmt.Errorf("task %d: %w", taskID, ErrNoJenkinsBuildInfo)
	}

	return s.jenkins.GetConsoleOutput(task.JenkinsJobName, task.JenkinsBuildNumber)
}

func (s *DeployService) generateJobConfig(template *model.BuildTemplate, targetBranch, repoURL, tag string) string {
	pipelineContent := strings.ReplaceAll(template.Pipeline, "\\n", "\n")
	pipelineContent = strings.ReplaceAll(pipelineContent, "\r\n", "\n")
	// 清理 JSON 转义残留的引号（\" → "），但保留 shell 行续符（\\）
	pipelineContent = strings.ReplaceAll(pipelineContent, "\\\"", "\"")
	pipelineContent = strings.ReplaceAll(pipelineContent, "\r", "\n")

	// 兼容旧 pipeline 脚本中的参数名迁移：HARBOR_* → REGISTRY_*
	// 数据库中的旧 BuildTemplate 可能仍在引用 HARBOR_REGISTRY/HARBOR_PROJECT/HARBOR_CREDS_ID
	// 替换后确保 pipeline 与新参数定义一致
	pipelineContent = strings.ReplaceAll(pipelineContent, "HARBOR_REGISTRY", "REGISTRY_URL")
	pipelineContent = strings.ReplaceAll(pipelineContent, "HARBOR_PROJECT", "REGISTRY_PROJECT")
	pipelineContent = strings.ReplaceAll(pipelineContent, "HARBOR_CREDS_ID", "REGISTRY_CREDS_ID")

	// 安全保护：将 pipeline 中裸 ${VAR} 替换为 ${params.VAR}
	// Declarative Pipeline 中 environment {} 变量与同名 Job 参数冲突时，
	// Groovy sandbox 绑定不稳定（post 块和早期阶段均可能触发），
	// 抛出 MissingPropertyException。params. 前缀在任何阶段都可靠。
	// 仅匹配纯 ${VAR_NAME}，不影响已使用 ${params.VAR_NAME} / ${env.VAR_NAME} 的脚本。
	// 覆盖全部 10 个 Job 参数名
	bareParamNames := []string{
		"IMAGE_NAME", "REGISTRY_URL", "REGISTRY_PROJECT", "Tag",
		"TARGET_BRANCH", "Repo_URL", "REGISTRY_CREDS_ID", "GIT_CREDS_ID",
		"DEPLOY_TARGET", "DOCKERFILE_CONTENT",
	}
	for _, name := range bareParamNames {
		re := regexp.MustCompile(`\$\{` + regexp.QuoteMeta(name) + `\}`)
		pipelineContent = re.ReplaceAllString(pipelineContent, `${params.` + name + `}`)
	}

	escapedPipeline := escapeXMLContent(pipelineContent)

	config := fmt.Sprintf(`<?xml version='1.1' encoding='UTF-8'?>
<flow-definition plugin="workflow-job@2.40">
	<description>Generated by Zebra CI/CD for %s</description>
	<keepDependencies>false</keepDependencies>
	<properties>
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
				<hudson.model.StringParameterDefinition>
					<name>REGISTRY_URL</name>
					<description>Image registry URL</description>
					<defaultValue></defaultValue>
					<trim>false</trim>
				</hudson.model.StringParameterDefinition>
				<hudson.model.StringParameterDefinition>
					<name>REGISTRY_PROJECT</name>
					<description>Image registry project</description>
					<defaultValue></defaultValue>
					<trim>false</trim>
				</hudson.model.StringParameterDefinition>
				<hudson.model.StringParameterDefinition>
					<name>IMAGE_NAME</name>
					<description>Image name</description>
					<defaultValue></defaultValue>
					<trim>false</trim>
				</hudson.model.StringParameterDefinition>
				<hudson.model.StringParameterDefinition>
					<name>REGISTRY_CREDS_ID</name>
					<description>Registry credential ID in Jenkins</description>
					<defaultValue></defaultValue>
					<trim>false</trim>
				</hudson.model.StringParameterDefinition>
				<hudson.model.StringParameterDefinition>
					<name>GIT_CREDS_ID</name>
					<description>Git credential ID in Jenkins</description>
					<defaultValue></defaultValue>
					<trim>false</trim>
				</hudson.model.StringParameterDefinition>
				<hudson.model.StringParameterDefinition>
					<name>DEPLOY_TARGET</name>
					<description>Deployment target (k8s/docker/linux)</description>
					<defaultValue>k8s</defaultValue>
					<trim>false</trim>
				</hudson.model.StringParameterDefinition>
				<hudson.model.StringParameterDefinition>
					<name>DOCKERFILE_CONTENT</name>
					<description>Dockerfile content from build template</description>
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

// escapeXMLContent XML 转义函数
func escapeXMLContent(content string) string {
	replacer := strings.NewReplacer(
		"]]>", "]]]]><![CDATA[>",
	)
	return replacer.Replace(content)
}

// GetRollbackHistory 获取可回滚的历史部署任务
// 查找同一 deployment_name + env_id + deploy_target 的成功历史任务
func (s *DeployService) GetRollbackHistory(taskID uint, page, size int) ([]model.DeployTask, int64, error) {
	// 1. 获取当前任务信息
	var currentTask model.DeployTask
	if err := s.db.First(&currentTask, taskID).Error; err != nil {
		return nil, 0, fmt.Errorf("任务不存在: %v", err)
	}

	// 2. 构建查询条件：同 deployment_name + env_id + deploy_target，状态为 SUCCESS，排除当前任务
	query := s.db.Model(&model.DeployTask{}).
		Where("deployment_name = ?", currentTask.DeploymentName).
		Where("env_id = ?", currentTask.EnvID).
		Where("deploy_target = ?", currentTask.DeployTarget).
		Where("status = ?", "SUCCESS").
		Where("id != ?", taskID).
		Where("is_rollback = ?", false) // 排除回滚任务，只显示正常部署的历史

	// 3. 分页查询
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计历史任务失败: %v", err)
	}

	var tasks []model.DeployTask
	offset := (page - 1) * size
	if err := query.Order("id DESC").Offset(offset).Limit(size).Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("查询历史任务失败: %v", err)
	}

	return tasks, total, nil
}

// RollbackDeployment 执行部署回滚
// 基于历史任务创建新的部署任务，使用相同的 ImageTag，跳过构建阶段
func (s *DeployService) RollbackDeployment(currentTaskID, historyTaskID uint) (*model.DeployTask, error) {
	// 1. 获取当前任务和历史任务
	var currentTask, historyTask model.DeployTask
	if err := s.db.First(&currentTask, currentTaskID).Error; err != nil {
		return nil, fmt.Errorf("当前任务不存在: %v", err)
	}
	if err := s.db.First(&historyTask, historyTaskID).Error; err != nil {
		return nil, fmt.Errorf("历史任务不存在: %v", err)
	}

	// 2. 验证历史任务
	if historyTask.Status != "SUCCESS" {
		return nil, fmt.Errorf("只能回滚到成功的部署任务，历史任务状态: %s", historyTask.Status)
	}
	if historyTask.DeploymentName != currentTask.DeploymentName {
		return nil, fmt.Errorf("部署名称不匹配: 当前=%s, 历史=%s", currentTask.DeploymentName, historyTask.DeploymentName)
	}
	if historyTask.EnvID != currentTask.EnvID {
		return nil, fmt.Errorf("环境ID不匹配: 当前=%d, 历史=%d", currentTask.EnvID, historyTask.EnvID)
	}
	if historyTask.DeployTarget != currentTask.DeployTarget {
		return nil, fmt.Errorf("部署目标不匹配: 当前=%s, 历史=%s", currentTask.DeployTarget, historyTask.DeployTarget)
	}

	// 3. 验证镜像在仓库中仍然存在
	client := s.getRegistryClientForTask(&historyTask)
	if !client.VerifyImageExists(historyTask.RegistryProject, historyTask.ImageName, historyTask.ImageTag) {
		return nil, fmt.Errorf("历史镜像不存在: %s/%s:%s", historyTask.RegistryProject, historyTask.ImageName, historyTask.ImageTag)
	}

	// 4. 创建新的回滚任务
	now := time.Now()
	newTask := &model.DeployTask{
		ProjectID:            historyTask.ProjectID,
		EnvID:                historyTask.EnvID,
		GitRef:               historyTask.GitRef,
		ImageTag:             historyTask.ImageTag, // 关键：使用历史镜像版本
		Status:               "PENDING",
		K8sClusterID:         historyTask.K8sClusterID,
		K8sNamespace:         historyTask.K8sNamespace,
		JenkinsJobName:       historyTask.JenkinsJobName,
		RegistryProject:      historyTask.RegistryProject,
		ImageName:            historyTask.ImageName,
		DeploymentName:       historyTask.DeploymentName,
		BuildTemplateID:      historyTask.BuildTemplateID,
		DeploymentTemplateID: historyTask.DeploymentTemplateID,
		DeployType:           historyTask.DeployType,
		DeployTarget:         historyTask.DeployTarget,
		ServerID:             historyTask.ServerID,
		DeployPath:           historyTask.DeployPath,
		IsRollback:           true,
		RollbackFrom:         historyTaskID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.db.Create(newTask).Error; err != nil {
		return nil, fmt.Errorf("创建回滚任务失败: %v", err)
	}

	log.S().Infof("Created rollback task %d from history task %d (image_tag=%s)", newTask.ID, historyTaskID, newTask.ImageTag)

	// 5. 入队执行（会调用 ProcessDeploymentTask）
	if err := s.queueClient.EnqueueDeployTask(newTask.ID); err != nil {
		// 回滚任务创建失败，删除任务记录
		s.db.Delete(&model.DeployTask{}, newTask.ID)
		return nil, fmt.Errorf("回滚任务入队失败: %v", err)
	}

	return newTask, nil
}

// ProcessRollbackTask 处理回滚任务的部署流程（跳过 BUILDING 和 PUSHING 阶段）
func (s *DeployService) ProcessRollbackTask(ctx context.Context, taskID uint) error {
	var task model.DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return fmt.Errorf("load task %d: %w", taskID, err)
	}

	// 幂等保护
	if task.Status == "SUCCESS" {
		return nil
	}

	// 记录 PENDING 阶段
	s.startStage(taskID, "PENDING", 0)
	s.finishStage(taskID, "PENDING", "success", "")

	// 跳过 BUILDING，直接标记为成功
	s.startStage(taskID, "BUILDING", 0)
	s.updateTaskStatus(taskID, "BUILDING", "回滚任务跳过构建", "")
	s.finishStage(taskID, "BUILDING", "success", "")

	// 跳过 PUSHING，直接标记为成功
	s.startStage(taskID, "PUSHING", 0)
	s.updateTaskStatus(taskID, "PUSHING", "回滚任务跳过推送验证", "")
	s.finishStage(taskID, "PUSHING", "success", "")

	// 直接进入 DEPLOYING 阶段
	s.startStage(taskID, "DEPLOYING", 0)
	deployTarget := task.DeployTarget
	if deployTarget == "" {
		deployTarget = task.DeployType // 兼容旧数据
	}

	var deployErr error
	switch deployTarget {
	case "docker":
		s.updateTaskStatus(taskID, "DEPLOYING", "开始回滚部署到Linux主机(Docker)", "")
		deployErr = s.deployToDocker(&task)
	case "k8s":
		s.updateTaskStatus(taskID, "DEPLOYING", "开始回滚部署到K8s集群", "")
		deployErr = s.deployToK8s(&task)
	case "linux":
		s.updateTaskStatus(taskID, "DEPLOYING", "开始回滚部署到Linux主机(文件提取+Nginx)", "")
		deployErr = s.deployToLinux(&task)
	default:
		deployErr = fmt.Errorf("未知的部署目标: %s", deployTarget)
	}

	if deployErr != nil {
		s.finishStage(taskID, "DEPLOYING", "failed", deployErr.Error())
		s.updateTaskStatus(taskID, "FAILED", fmt.Sprintf("回滚部署失败: %v", deployErr), deployErr.Error())
		return deployErr
	}

	s.finishStage(taskID, "DEPLOYING", "success", "")
	s.updateTaskStatus(taskID, "SUCCESS", "回滚部署成功完成", "")
	log.S().Infof("Rollback task %d completed successfully (image_tag=%s)", taskID, task.ImageTag)
	return nil
}

// TriggerBuild 手动触发构建阶段
func (s *DeployService) TriggerBuild(taskID uint) (*model.DeployTask, error) {
	var task model.DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, fmt.Errorf("任务不存在: %v", err)
	}

	// 验证执行模式和状态
	if task.ExecutionMode != "manual" {
		return nil, fmt.Errorf("只有手动执行模式的任务才能触发构建")
	}
	if task.BuildStatus != "ready" {
		return nil, fmt.Errorf("当前构建状态为 %s，无法执行构建", task.BuildStatus)
	}

	// 生成镜像标签
	now := time.Now()
	imageTag := now.Format("20060102150405")

	// 更新状态
	updates := map[string]interface{}{
		"status":        "BUILDING",
		"build_status":  "executing",
		"image_tag":     imageTag,
		"started_at":    now,
		"updated_at":    now,
	}
	if err := s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新任务状态失败: %v", err)
	}

	// 重新加载任务
	s.db.First(&task, taskID)

	// 记录阶段历史
	s.startStage(taskID, "BUILDING", 0)
	log.S().Infof("Manual build triggered for task %d, image_tag=%s", taskID, imageTag)

	// 入队构建任务
	if err := s.queueClient.EnqueueBuildTask(taskID); err != nil {
		// 回滚状态
		s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"status":       "CREATED",
			"build_status": "ready",
		})
		return nil, fmt.Errorf("构建任务入队失败: %v", err)
	}

	return &task, nil
}

// TriggerDeploy 手动触发部署阶段（仅执行部署，构建已完成）
func (s *DeployService) TriggerDeploy(taskID uint) (*model.DeployTask, error) {
	var task model.DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, fmt.Errorf("任务不存在: %v", err)
	}

	// 验证执行模式和状态
	if task.ExecutionMode != "manual" {
		return nil, fmt.Errorf("只有手动执行模式的任务才能触发部署")
	}
	if task.BuildStatus != "done" {
		return nil, fmt.Errorf("构建尚未完成，当前构建状态: %s", task.BuildStatus)
	}
	if task.DeployStatus != "ready" {
		return nil, fmt.Errorf("当前部署状态为 %s，无法执行部署", task.DeployStatus)
	}

	// 更新状态
	now := time.Now()
	updates := map[string]interface{}{
		"status":        "DEPLOYING",
		"deploy_status": "executing",
		"updated_at":    now,
	}
	if err := s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新任务状态失败: %v", err)
	}

	// 重新加载任务
	s.db.First(&task, taskID)

	// 记录阶段历史
	s.startStage(taskID, "DEPLOYING", 0)
	log.S().Infof("Manual deploy triggered for task %d, image_tag=%s", taskID, task.ImageTag)

	// 入队部署任务
	if err := s.queueClient.EnqueueDeployOnlyTask(taskID); err != nil {
		// 回滚状态
		s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"status":       "BUILDING",
			"deploy_status": "ready",
		})
		return nil, fmt.Errorf("部署任务入队失败: %v", err)
	}

	return &task, nil
}

// TriggerAll 一键执行构建和部署
func (s *DeployService) TriggerAll(taskID uint) (*model.DeployTask, error) {
	var task model.DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, fmt.Errorf("任务不存在: %v", err)
	}

	// 验证执行模式
	if task.ExecutionMode != "manual" {
		return nil, fmt.Errorf("只有手动执行模式的任务才能触发执行")
	}

	// 如果构建未完成，先触发构建
	if task.BuildStatus == "ready" {
		if _, err := s.TriggerBuild(taskID); err != nil {
			return nil, err
		}
		// 构建完成后会自动触发部署（在 ProcessBuildTask 中）
		s.db.First(&task, taskID)
		return &task, nil
	}

	// 如果构建已完成，直接触发部署
	if task.BuildStatus == "done" && task.DeployStatus == "ready" {
		return s.TriggerDeploy(taskID)
	}

	return nil, fmt.Errorf("当前状态不支持一键执行: build_status=%s, deploy_status=%s", task.BuildStatus, task.DeployStatus)
}

// ProcessBuildTask 处理仅构建任务（手动执行模式的构建阶段）
func (s *DeployService) ProcessBuildTask(ctx context.Context, taskID uint) error {
	var task model.DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return fmt.Errorf("load task %d: %w", taskID, err)
	}

	// 幂等保护
	if task.BuildStatus == "done" {
		return nil
	}

	log.S().Infof("Processing build task %d (manual mode)", taskID)

	// 触发 Jenkins 构建
	buildResult, jenkinsClient, err := s.triggerJenkinsBuild(&task)
	if err != nil {
		s.finishStage(taskID, "BUILDING", "failed", err.Error())
		s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"status":        "FAILED",
			"build_status":  "failed",
			"error_message": err.Error(),
		})
		return err
	}

	// 保存 Jenkins 构建编号
	s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Update("jenkins_build_number", buildResult.BuildNumber)

	// 等待构建完成
	if !s.waitForJenkinsBuild(ctx, jenkinsClient, buildResult.JobName, buildResult.BuildNumber) {
		errMsg := "Jenkins构建失败或超时"
		s.finishStage(taskID, "BUILDING", "failed", errMsg)
		s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"status":        "FAILED",
			"build_status":  "failed",
			"error_message": errMsg,
		})
		return fmt.Errorf("jenkins build failed: %s", errMsg)
	}

	// 验证镜像推送
	if !s.verifyImageInRegistryForTask(&task) {
		errMsg := "镜像验证失败"
		s.finishStage(taskID, "PUSHING", "failed", errMsg)
		s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"status":        "FAILED",
			"build_status":  "failed",
			"error_message": errMsg,
		})
		return fmt.Errorf("image verification failed")
	}

	// 构建完成
	now := time.Now()
	s.finishStage(taskID, "BUILDING", "success", "")
	s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":          "BUILDING", // 保持构建状态，等待部署触发
		"build_status":    "done",
		"deploy_status":   "ready",
		"build_finished_at": now,
	})

	log.S().Infof("Build task %d completed successfully", taskID)
	return nil
}

// ProcessDeployOnlyTask 处理仅部署任务（手动执行模式的部署阶段）
func (s *DeployService) ProcessDeployOnlyTask(ctx context.Context, taskID uint) error {
	var task model.DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return fmt.Errorf("load task %d: %w", taskID, err)
	}

	// 幂等保护
	if task.DeployStatus == "done" {
		return nil
	}

	log.S().Infof("Processing deploy-only task %d (manual mode)", taskID)

	// 执行部署
	deployTarget := task.DeployTarget
	if deployTarget == "" {
		deployTarget = task.DeployType
	}

	var deployErr error
	switch deployTarget {
	case "docker":
		deployErr = s.deployToDocker(&task)
	case "k8s":
		deployErr = s.deployToK8s(&task)
	case "linux":
		deployErr = s.deployToLinux(&task)
	default:
		deployErr = fmt.Errorf("未知的部署目标: %s", deployTarget)
	}

	now := time.Now()
	if deployErr != nil {
		s.finishStage(taskID, "DEPLOYING", "failed", deployErr.Error())
		s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
			"status":        "FAILED",
			"deploy_status": "failed",
			"error_message": deployErr.Error(),
			"finished_at":   now,
		})
		return deployErr
	}

	// 部署完成
	s.finishStage(taskID, "DEPLOYING", "success", "")
	s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":        "SUCCESS",
		"deploy_status": "done",
		"finished_at":   now,
	})

	log.S().Infof("Deploy-only task %d completed successfully", taskID)
	return nil
}

// CancelScheduledTask 取消定时任务
func (s *DeployService) CancelScheduledTask(taskID uint) error {
	var task model.DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return fmt.Errorf("任务不存在: %v", err)
	}

	if task.Status != "SCHEDULED" {
		return fmt.Errorf("只有 SCHEDULED 状态的任务才能取消")
	}

	// 更新状态为取消
	now := time.Now()
	if err := s.db.Model(&model.DeployTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":      "CANCELLED",
		"finished_at": now,
	}).Error; err != nil {
		return fmt.Errorf("更新状态失败: %v", err)
	}

	log.S().Infof("Scheduled task %d cancelled", taskID)
	return nil
}

// ListScheduledTasks 获取待执行的定时任务列表
func (s *DeployService) ListScheduledTasks(page, size int) ([]model.DeployTask, int64, error) {
	query := s.db.Model(&model.DeployTask{}).Where("status = ?", "SCHEDULED")

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []model.DeployTask
	offset := (page - 1) * size
	if err := query.Order("scheduled_at ASC").Offset(offset).Limit(size).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}
