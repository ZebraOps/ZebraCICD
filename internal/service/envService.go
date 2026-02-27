package service

import (
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
)

type EnvService struct {
	envRepo *handler.EnvRepository
}

func NewEnvService(envRepo *handler.EnvRepository) *EnvService {
	return &EnvService{
		envRepo: envRepo,
	}
}

// CreateEnv 创建环境
func (s *EnvService) CreateEnv(env *model.Environment) error {
	return s.envRepo.Create(env)
}

// GetEnvByID 根据ID获取环境
func (s *EnvService) GetEnvByID(id uint) (*model.Environment, error) {
	return s.envRepo.GetByID(id)
}

// ListEnvsWithConditions 根据条件分页获取环境列表
func (s *EnvService) ListEnvsWithConditions(conditions types.EnvQueryConditions, page, size int) ([]model.Environment, int64, error) {
	return s.envRepo.ListWithConditions(conditions, page, size)
}

// UpdateEnv 更新环境
func (s *EnvService) UpdateEnv(env *model.Environment) error {
	return s.envRepo.Update(env)
}

// DeleteEnv 删除环境
func (s *EnvService) DeleteEnv(id uint) error {
	return s.envRepo.Delete(id)
}
