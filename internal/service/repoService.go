package service

import (
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
	return s.repoRepo.Create(repo)
}

func (s *RepoService) GetRepoByID(id uint) (*model.Repo, error) {
	return s.repoRepo.GetByID(id)
}

func (s *RepoService) ListRepos() ([]model.Repo, error) {
	return s.repoRepo.List()
}

func (s *RepoService) UpdateRepo(repo *model.Repo) error {
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

// ListRepoBranchesByApp 通过应用ID获取关联仓库的分支列表
func (s *RepoService) ListRepoBranchesByApp(appID uint) ([]string, error) {
	var app model.Application
	if err := s.db.First(&app, appID).Error; err != nil {
		return nil, fmt.Errorf("应用 %d 不存在: %v", appID, err)
	}
	var repo model.Repo
	if err := s.db.First(&repo, app.RepoID).Error; err != nil {
		return nil, fmt.Errorf("仓库 %d 不存在: %v", app.RepoID, err)
	}
	projectPath := repo.RepoNumber
	if projectPath == "" {
		return nil, fmt.Errorf("仓库 %d 缺少 GitLab 项目编号", repo.ID)
	}
	branches, err := s.gitlabClient.GetBranches(projectPath)
	if err != nil {
		return nil, fmt.Errorf("获取分支列表失败: %v", err)
	}
	names := make([]string, 0, len(branches))
	for _, b := range branches {
		names = append(names, b.Name)
	}
	return names, nil
}

// ListRepoTagsByApp 通过应用ID获取关联仓库的标签列表
func (s *RepoService) ListRepoTagsByApp(appID uint) ([]string, error) {
	var app model.Application
	if err := s.db.First(&app, appID).Error; err != nil {
		return nil, fmt.Errorf("应用 %d 不存在: %v", appID, err)
	}
	var repo model.Repo
	if err := s.db.First(&repo, app.RepoID).Error; err != nil {
		return nil, fmt.Errorf("仓库 %d 不存在: %v", app.RepoID, err)
	}
	projectPath := repo.RepoNumber
	if projectPath == "" {
		return nil, fmt.Errorf("仓库 %d 缺少 GitLab 项目编号", repo.ID)
	}
	tags, err := s.gitlabClient.GetTags(projectPath)
	if err != nil {
		return nil, fmt.Errorf("获取标签列表失败: %v", err)
	}
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}
	return names, nil
}
