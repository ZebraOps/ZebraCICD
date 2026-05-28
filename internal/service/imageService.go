package service

import (
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
)

// ImageRepositoryService 提供通用的镜像仓库存储功能
type ImageRepositoryService struct {
	repo *handler.ImageRepositoryRepository
}

// NewImageRepositoryService 创建一个新的 ImageRepositoryService 实例
func NewImageRepositoryService(repo *handler.ImageRepositoryRepository) *ImageRepositoryService {
	return &ImageRepositoryService{repo: repo}
}

// CreateRepository 创建镜像仓库
func (s *ImageRepositoryService) CreateRepository(repository *model.ImageRepository) error {
	return s.repo.Create(repository)
}

// GetRepositoryByID 根据ID获取镜像仓库
func (s *ImageRepositoryService) GetRepositoryByID(id uint) (*model.ImageRepository, error) {
	return s.repo.GetByID(id)
}

// ListRepositories 根据条件分页获取镜像仓库列表
func (s *ImageRepositoryService) ListRepositories(conditions types.ImageRepositoryQueryConditions) ([]model.ImageRepository, int64, error) {
	return s.repo.ListWithConditions(conditions)
}

// UpdateRepository 更新镜像仓库
func (s *ImageRepositoryService) UpdateRepository(repository *model.ImageRepository) error {
	return s.repo.Update(repository)
}

// DeleteRepository 删除镜像仓库
func (s *ImageRepositoryService) DeleteRepository(id uint) error {
	return s.repo.Delete(id)
}
