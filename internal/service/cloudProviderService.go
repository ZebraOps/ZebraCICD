package service

import (
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
)

type CloudProviderService struct {
	providerRepo *handler.CloudProviderRepository
}

func NewCloudProviderService(providerRepo *handler.CloudProviderRepository) *CloudProviderService {
	return &CloudProviderService{
		providerRepo: providerRepo,
	}
}

// CreateCloudProvider 创建云厂商
func (s *CloudProviderService) CreateCloudProvider(provider *model.CloudProvider) error {
	return s.providerRepo.Create(provider)
}

// GetCloudProviderByID 根据ID获取云厂商
func (s *CloudProviderService) GetCloudProviderByID(id uint) (*model.CloudProvider, error) {
	return s.providerRepo.GetByID(id)
}

// GetCloudProviderByProvider 根据提供商标识获取云厂商
func (s *CloudProviderService) GetCloudProviderByProvider(provider string) (*model.CloudProvider, error) {
	return s.providerRepo.GetByProvider(provider)
}

// ListCloudProvidersWithConditions 根据条件分页获取云厂商列表
func (s *CloudProviderService) ListCloudProvidersWithConditions(conditions types.CloudProviderQueryConditions, page, size int) ([]model.CloudProvider, int64, error) {
	return s.providerRepo.ListWithConditions(conditions, page, size)
}

// UpdateCloudProvider 更新云厂商
func (s *CloudProviderService) UpdateCloudProvider(provider *model.CloudProvider) error {
	return s.providerRepo.Update(provider)
}

// DeleteCloudProvider 删除云厂商
func (s *CloudProviderService) DeleteCloudProvider(id uint) error {
	return s.providerRepo.Delete(id)
}
