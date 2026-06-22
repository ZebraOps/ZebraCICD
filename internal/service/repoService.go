package service

import (
	"encoding/json"
	"fmt"

	"github.com/ZebraOps/ZebraCICD/internal/core"
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
)

type RepoService struct {
	repoRepo      *handler.RepoRepository
	gitlabClient  *core.GitLabClient
	gitlabBaseURL string
	db            *gorm.DB
}

func NewRepoService(
	repoRepo *handler.RepoRepository,
	gitlabClient *core.GitLabClient,
	gitlabBaseURL string,
	db *gorm.DB) *RepoService {
	return &RepoService{
		repoRepo:      repoRepo,
		gitlabClient:  gitlabClient,
		gitlabBaseURL: gitlabBaseURL,
		db:            db,
	}
}

func (s *RepoService) CreateRepo(repo *model.Repo) error {
	// 同一平台下名称唯一性检查
	var existing model.Repo
	if err := s.db.Where(
		"(git_platform_id IS NOT DISTINCT FROM ?) AND (c_name = ? OR e_name = ?)",
		repo.GitPlatformID, repo.CName, repo.EName,
	).First(&existing).Error; err == nil {
		if existing.CName == repo.CName {
			return fmt.Errorf("该平台下中文名称 '%s' 已存在", repo.CName)
		}
		if existing.EName == repo.EName {
			return fmt.Errorf("该平台下英文名称 '%s' 已存在", repo.EName)
		}
	}
	return s.repoRepo.Create(repo)
}

func (s *RepoService) GetRepoByID(id uint) (*model.Repo, error) {
	return s.repoRepo.GetByID(id)
}

func (s *RepoService) ListRepos() ([]model.Repo, error) {
	return s.repoRepo.List()
}

func (s *RepoService) UpdateRepo(repo *model.Repo) error {
	// 同一平台下名称唯一性检查（排除自身）
	var existing model.Repo
	if err := s.db.Where(
		"id != ? AND (git_platform_id IS NOT DISTINCT FROM ?) AND (c_name = ? OR e_name = ?)",
		repo.ID, repo.GitPlatformID, repo.CName, repo.EName,
	).First(&existing).Error; err == nil {
		if existing.CName == repo.CName {
			return fmt.Errorf("该平台下中文名称 '%s' 已被其他仓库使用", repo.CName)
		}
		if existing.EName == repo.EName {
			return fmt.Errorf("该平台下英文名称 '%s' 已被其他仓库使用", repo.EName)
		}
	}
	return s.repoRepo.Update(repo)
}

func (s *RepoService) DeleteRepo(id uint) error {
	return s.repoRepo.Delete(id)
}

// GetRepoInfoFromGitLab 根据id从GitLab获取仓库地址
func (s *RepoService) GetRepoInfoFromGitLab(id string) (*types.Project, error) {
	// 在 service 层调用时
	project, err := s.gitlabClient.GetProject(id)
	if err != nil {
		return nil, fmt.Errorf("project not found: %v", err)
	}

	return project, nil
}

// ListReposWithConditions 根据条件分页获取仓库列表
func (s *RepoService) ListReposWithConditions(conditions types.RepoQueryConditions, page, size int) ([]model.RepoResp, int64, error) {
	return s.repoRepo.ListWithConditions(conditions, page, size)
}

// ListRepoBranchesByApp 通过应用ID获取关联仓库的分支列表（支持多平台）
func (s *RepoService) ListRepoBranchesByApp(appID uint) ([]string, error) {
	var app model.Application
	if err := s.db.First(&app, appID).Error; err != nil {
		return nil, fmt.Errorf("应用 %d 不存在: %v", appID, err)
	}
	var repo model.Repo
	if err := s.db.First(&repo, app.RepoID).Error; err != nil {
		return nil, fmt.Errorf("仓库 %d 不存在: %v", app.RepoID, err)
	}
	if repo.RepoNumber == "" {
		return nil, fmt.Errorf("仓库 %d 缺少项目编号 (repo_number)，请在仓库管理中填写", repo.ID)
	}

	// 从应用的部署配置获取 git_platform_id
	platformCfg := s.getGitPlatformByAppID(appID)

	branches, err := core.FetchBranches(platformCfg.URL, platformCfg.PlatformType, platformCfg.AuthType, platformCfg.AuthConfig, repo.RepoNumber)
	if err != nil {
		return nil, fmt.Errorf("获取分支列表失败: %v", err)
	}
	names := make([]string, 0, len(branches))
	for _, b := range branches {
		names = append(names, b.Name)
	}
	return names, nil
}

// ListRepoTagsByApp 通过应用ID获取关联仓库的标签列表（支持多平台）
func (s *RepoService) ListRepoTagsByApp(appID uint) ([]string, error) {
	var app model.Application
	if err := s.db.First(&app, appID).Error; err != nil {
		return nil, fmt.Errorf("应用 %d 不存在: %v", appID, err)
	}
	var repo model.Repo
	if err := s.db.First(&repo, app.RepoID).Error; err != nil {
		return nil, fmt.Errorf("仓库 %d 不存在: %v", app.RepoID, err)
	}
	if repo.RepoNumber == "" {
		return nil, fmt.Errorf("仓库 %d 缺少项目编号 (repo_number)，请在仓库管理中填写", repo.ID)
	}

	platformCfg := s.getGitPlatformByAppID(appID)

	tags, err := core.FetchTags(platformCfg.URL, platformCfg.PlatformType, platformCfg.AuthType, platformCfg.AuthConfig, repo.RepoNumber)
	if err != nil {
		return nil, fmt.Errorf("获取标签列表失败: %v", err)
	}
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}
	return names, nil
}

// gitPlatformInfo 提取后返回的 Git 平台配置简化结构
type gitPlatformInfo struct {
	URL          string
	PlatformType string
	AuthType     string
	AuthConfig   string
}

// getGitPlatformByAppID 从应用部署配置中查找关联的 Git 平台
func (s *RepoService) getGitPlatformByAppID(appID uint) gitPlatformInfo {
	// 默认回退：使用主 GitLab 客户端
	fallback := gitPlatformInfo{
		URL:          s.gitlabBaseURL,
		PlatformType: "gitlab",
		AuthType:     "token",
		AuthConfig:   "",
	}

	// 通过 ApplicationDeployment 获取 git_platform_id
	var deploy model.ApplicationDeployment
	if err := s.db.Where("application_id = ?", appID).First(&deploy).Error; err != nil {
		return fallback
	}
	if deploy.GitPlatformID == nil {
		return fallback
	}

	var platform model.GitPlatform
	if err := s.db.First(&platform, *deploy.GitPlatformID).Error; err != nil {
		return fallback
	}

	// 解析 auth_config JSON，提取 token 注入到 AuthConfig
	authConfig := platform.AuthConfig
	if authConfig == "" {
		authConfig = "{}"
	}
	// 对于 token 认证，确保 token 设置到请求头
	var ac map[string]string
	if json.Unmarshal([]byte(authConfig), &ac) == nil {
		if token, ok := ac["token"]; ok && token != "" {
			authConfig = fmt.Sprintf(`{"token":"%s"}`, token)
		}
	}

	return gitPlatformInfo{
		URL:          platform.URL,
		PlatformType: platform.PlatformType,
		AuthType:     platform.AuthType,
		AuthConfig:   authConfig,
	}
}
