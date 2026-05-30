package service

import (
	"fmt"

	"github.com/ZebraOps/ZebraCICD/internal/core"
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
)

type GitPlatformService struct {
	repo *handler.GitPlatformRepository
}

func NewGitPlatformService(repo *handler.GitPlatformRepository) *GitPlatformService {
	return &GitPlatformService{
		repo: repo,
	}
}

// CreateGitPlatform 创建Git平台配置
func (s *GitPlatformService) CreateGitPlatform(platform *model.GitPlatform) error {
	return s.repo.Create(platform)
}

// GetGitPlatformByID 根据ID获取Git平台配置
func (s *GitPlatformService) GetGitPlatformByID(id uint) (*model.GitPlatform, error) {
	return s.repo.GetByID(id)
}

// ListGitPlatformsWithConditions 根据条件分页获取Git平台配置列表
func (s *GitPlatformService) ListGitPlatformsWithConditions(conditions types.GitPlatformQueryConditions, page, size int) ([]model.GitPlatform, int64, error) {
	return s.repo.ListWithConditions(conditions, page, size)
}

// UpdateGitPlatform 更新Git平台配置
func (s *GitPlatformService) UpdateGitPlatform(platform *model.GitPlatform) error {
	return s.repo.Update(platform)
}

// DeleteGitPlatform 删除Git平台配置
func (s *GitPlatformService) DeleteGitPlatform(id uint) error {
	return s.repo.Delete(id)
}

// TestGitPlatformConnection 测试Git平台连通性
func (s *GitPlatformService) TestGitPlatformConnection(id uint) error {
	platform, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("git platform not found")
	}
	return core.TestGitPlatformConnectivity(platform.URL, platform.PlatformType, platform.AuthType, platform.AuthConfig)
}

// ListPlatformProjects 获取指定Git平台的仓库/项目列表
func (s *GitPlatformService) ListPlatformProjects(id uint, search string, page, size int) ([]types.Project, error) {
	platform, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("git platform not found")
	}
	return core.FetchPlatformProjects(platform.URL, platform.PlatformType, platform.AuthType, platform.AuthConfig, search, page, size)
}