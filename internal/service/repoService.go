package service

import (
	"fmt"

	"github.com/ZebraOps/ZebraCICD/internal/core"
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
)

type RepoService struct {
	repoRepo      *handler.RepoRepository
	gitlabClient  *core.GitLabClient
	gitlabBaseURL string
}

func NewRepoService(
	repoRepo *handler.RepoRepository,
	gitlabClient *core.GitLabClient,
	gitlabBaseURL string) *RepoService {
	return &RepoService{
		repoRepo:      repoRepo,
		gitlabClient:  gitlabClient,
		gitlabBaseURL: gitlabBaseURL,
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
