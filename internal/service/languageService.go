package service

import (
	"github.com/ZebraOps/ZebraCICD/internal/handler"
	"github.com/ZebraOps/ZebraCICD/internal/model"
	"github.com/ZebraOps/ZebraCICD/internal/types"
)

type LanguageService struct {
	langRepo *handler.LanguageRepository
}

func NewLanguageService(langRepo *handler.LanguageRepository) *LanguageService {
	return &LanguageService{
		langRepo: langRepo,
	}
}

// CreateLanguage 创建开发语言
func (s *LanguageService) CreateLanguage(lang *model.Language) error {
	return s.langRepo.Create(lang)
}

// GetLanguageByID 根据ID获取开发语言
func (s *LanguageService) GetLanguageByID(id uint) (*model.Language, error) {
	return s.langRepo.GetByID(id)
}

// ListLanguagesWithConditions 根据条件分页获取开发语言列表
func (s *LanguageService) ListLanguagesWithConditions(conditions types.LanguageQueryConditions, page, size int) ([]model.Language, int64, error) {
	return s.langRepo.ListWithConditions(conditions, page, size)
}

// UpdateLanguage 更新开发语言
func (s *LanguageService) UpdateLanguage(lang *model.Language) error {
	return s.langRepo.Update(lang)
}

// DeleteLanguage 删除开发语言
func (s *LanguageService) DeleteLanguage(id uint) error {
	return s.langRepo.Delete(id)
}
