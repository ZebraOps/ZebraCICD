package service

import (
	"fmt"

	"github.com/ZebraOps/ZebraCICD/internal/core"
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
)

type JenkinsPlatformService struct {
	repo *handler.JenkinsPlatformRepository
}

func NewJenkinsPlatformService(repo *handler.JenkinsPlatformRepository) *JenkinsPlatformService {
	return &JenkinsPlatformService{repo: repo}
}

func (s *JenkinsPlatformService) CreateJenkinsPlatform(platform *model.JenkinsPlatform) error {
	return s.repo.Create(platform)
}

func (s *JenkinsPlatformService) GetJenkinsPlatformByID(id uint) (*model.JenkinsPlatform, error) {
	return s.repo.GetByID(id)
}

func (s *JenkinsPlatformService) ListJenkinsPlatformsWithConditions(conditions types.JenkinsPlatformQueryConditions, page, size int) ([]model.JenkinsPlatform, int64, error) {
	return s.repo.ListWithConditions(conditions, page, size)
}

func (s *JenkinsPlatformService) UpdateJenkinsPlatform(platform *model.JenkinsPlatform) error {
	return s.repo.Update(platform)
}

func (s *JenkinsPlatformService) DeleteJenkinsPlatform(id uint) error {
	return s.repo.Delete(id)
}

// TestJenkinsPlatformConnection 测试Jenkins平台连通性
func (s *JenkinsPlatformService) TestJenkinsPlatformConnection(id uint) error {
	platform, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("jenkins platform not found")
	}
	client := core.NewJenkinsClient(platform.URL, platform.Username, platform.Password)
	return client.Authenticate()
}