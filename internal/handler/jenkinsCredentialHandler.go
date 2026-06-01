package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type JenkinsCredentialRepository struct {
	db *gorm.DB
}

func NewJenkinsCredentialRepository(db *gorm.DB) *JenkinsCredentialRepository {
	return &JenkinsCredentialRepository{db: db}
}

func (r *JenkinsCredentialRepository) Create(cred *model.JenkinsCredential) error {
	return r.db.Create(cred).Error
}

func (r *JenkinsCredentialRepository) GetByID(id uint) (*model.JenkinsCredential, error) {
	var cred model.JenkinsCredential
	if err := r.db.First(&cred, id).Error; err != nil {
		return nil, err
	}
	return &cred, nil
}

// FindByPlatformAndCredID 根据平台ID和凭据ID查找，用于 Upsert 判断
// 使用 Find+RowsAffected 而非 First，避免 GORM 将「未找到」以 ERROR 级别写入日志
func (r *JenkinsCredentialRepository) FindByPlatformAndCredID(platformID uint, credentialID string) (*model.JenkinsCredential, error) {
	var cred model.JenkinsCredential
	result := r.db.Session(&gorm.Session{Logger: r.db.Logger.LogMode(logger.Warn)}).
		Where("jenkins_platform_id = ? AND credential_id = ?", platformID, credentialID).
		Limit(1).Find(&cred)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &cred, nil
}

// ListIDsByPlatform 返回某平台下所有凭据的 credential_id 集合（用于同步时检测已删除项）
func (r *JenkinsCredentialRepository) ListIDsByPlatform(platformID uint) ([]string, error) {
	var ids []string
	err := r.db.Model(&model.JenkinsCredential{}).
		Where("jenkins_platform_id = ? AND status != ?", platformID, "synced_deleted").
		Pluck("credential_id", &ids).Error
	return ids, err
}

func (r *JenkinsCredentialRepository) ListWithConditions(conditions types.JenkinsCredentialQueryConditions, page, size int) ([]model.JenkinsCredential, int64, error) {
	var creds []model.JenkinsCredential
	var total int64

	offset := (page - 1) * size
	db := r.db.Model(&model.JenkinsCredential{})

	if conditions.JenkinsPlatformID > 0 {
		db = db.Where("jenkins_platform_id = ?", conditions.JenkinsPlatformID)
	}
	if conditions.Name != "" {
		db = db.Where("credential_id LIKE ? OR display_name LIKE ?", "%"+conditions.Name+"%", "%"+conditions.Name+"%")
	}
	if conditions.CredentialType != "" {
		db = db.Where("credential_type = ?", conditions.CredentialType)
	}
	if conditions.Status != "" {
		db = db.Where("status = ?", conditions.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Offset(offset).Limit(size).Order("id DESC").Find(&creds).Error; err != nil {
		return nil, 0, err
	}
	return creds, total, nil
}

func (r *JenkinsCredentialRepository) Update(cred *model.JenkinsCredential) error {
	return r.db.Save(cred).Error
}

func (r *JenkinsCredentialRepository) Delete(id uint) error {
	return r.db.Delete(&model.JenkinsCredential{}, id).Error
}

// MarkDeletedByPlatformAndCredIDs 将指定平台下指定凭据ID列表的状态置为 synced_deleted
func (r *JenkinsCredentialRepository) MarkDeletedByPlatformAndCredIDs(platformID uint, credIDs []string) (int64, error) {
	result := r.db.Model(&model.JenkinsCredential{}).
		Where("jenkins_platform_id = ? AND credential_id IN ?", platformID, credIDs).
		Update("status", "synced_deleted")
	return result.RowsAffected, result.Error
}
