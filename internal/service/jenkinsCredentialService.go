package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/ZebraOps/ZebraCICD/internal/core"
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"github.com/ZebraOps/ZebraCICD/pkg/timeutil"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// SyncResult 同步结果统计
type SyncResult struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
}

type JenkinsCredentialService struct {
	repo        *handler.JenkinsCredentialRepository
	platformSvc *JenkinsPlatformService
}

func NewJenkinsCredentialService(repo *handler.JenkinsCredentialRepository, platformSvc *JenkinsPlatformService) *JenkinsCredentialService {
	return &JenkinsCredentialService{repo: repo, platformSvc: platformSvc}
}

func (s *JenkinsCredentialService) CreateCredential(cred *model.JenkinsCredential) error {
	return s.repo.Create(cred)
}

func (s *JenkinsCredentialService) GetCredentialByID(id uint) (*model.JenkinsCredential, error) {
	return s.repo.GetByID(id)
}

func (s *JenkinsCredentialService) ListCredentials(conditions types.JenkinsCredentialQueryConditions, page, size int) ([]model.JenkinsCredential, int64, error) {
	return s.repo.ListWithConditions(conditions, page, size)
}

func (s *JenkinsCredentialService) UpdateCredential(cred *model.JenkinsCredential) error {
	return s.repo.Update(cred)
}

func (s *JenkinsCredentialService) DeleteCredential(id uint) error {
	return s.repo.Delete(id)
}

// SyncCredentials 与指定 Jenkins 平台同步凭据
// 同步策略（Upsert + 软标记）：
//   - Jenkins 有、本地无 → INSERT (added+1)
//   - Jenkins 有、本地有 → UPDATE 元数据 (updated+1)
//   - Jenkins 无、本地有 → status=synced_deleted (deleted+1)
func (s *JenkinsCredentialService) SyncCredentials(platformID uint) (*SyncResult, error) {
	// 1. 获取 Jenkins 平台配置
	platform, err := s.platformSvc.GetJenkinsPlatformByID(platformID)
	if err != nil {
		return nil, fmt.Errorf("jenkins platform not found: %w", err)
	}

	// 2. 从 Jenkins 获取凭据列表
	client := core.NewJenkinsClient(platform.URL, platform.Username, platform.Password)
	remoteItems, err := client.ListCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch credentials from jenkins: %w", err)
	}

	result := &SyncResult{}
	now := timeutil.JSONTime(time.Now())

	// 3. 构建 remote credential_id 集合（用于后续检测已删除项）
	remoteIDs := lo.Map(remoteItems, func(item core.JenkinsCredentialItem, _ int) string {
		return item.ID
	})
	remoteIDSet := lo.SliceToMap(remoteIDs, func(id string) (string, struct{}) {
		return id, struct{}{}
	})

	// 4. Upsert：遍历 Jenkins 上的凭据
	for _, item := range remoteItems {
		existing, findErr := s.repo.FindByPlatformAndCredID(platformID, item.ID)
		if findErr != nil {
			if !errors.Is(findErr, gorm.ErrRecordNotFound) {
				// 真实 DB 错误，终止同步
				return nil, fmt.Errorf("failed to find credential %s: %w", item.ID, findErr)
			}
			// 本地不存在 → INSERT
			newCred := &model.JenkinsCredential{
				JenkinsPlatformID: platformID,
				CredentialID:      item.ID,
				DisplayName:       item.DisplayName,
				Description:       item.Description,
				CredentialType:    item.TypeName,
				Username:          item.Username,
				Scope:             item.Scope,
				Status:            "active",
				SyncedAt:          now,
			}
			if newCred.Scope == "" {
				newCred.Scope = "GLOBAL"
			}
			if createErr := s.repo.Create(newCred); createErr != nil {
				return nil, fmt.Errorf("failed to create credential %s: %w", item.ID, createErr)
			}
			result.Added++
		} else {
			// 本地已存在 → UPDATE 元数据
			existing.DisplayName = item.DisplayName
			existing.Description = item.Description
			existing.CredentialType = item.TypeName
			existing.Username = item.Username
			if item.Scope != "" {
				existing.Scope = item.Scope
			}
			// 恢复之前被标记为 synced_deleted 的记录
			existing.Status = "active"
			existing.SyncedAt = now
			if updateErr := s.repo.Update(existing); updateErr != nil {
				return nil, fmt.Errorf("failed to update credential %s: %w", item.ID, updateErr)
			}
			result.Updated++
		}
	}

	// 5. 检测本地有但 Jenkins 上已不存在的凭据 → 标记 synced_deleted
	localIDs, err := s.repo.ListIDsByPlatform(platformID)
	if err != nil {
		return nil, fmt.Errorf("failed to list local credentials: %w", err)
	}

	var toDelete []string
	for _, localID := range localIDs {
		if _, exists := remoteIDSet[localID]; !exists {
			toDelete = append(toDelete, localID)
		}
	}

	if len(toDelete) > 0 {
		deleted, deleteErr := s.repo.MarkDeletedByPlatformAndCredIDs(platformID, toDelete)
		if deleteErr != nil {
			return nil, fmt.Errorf("failed to mark deleted credentials: %w", deleteErr)
		}
		result.Deleted = int(deleted)
	}

	return result, nil
}
