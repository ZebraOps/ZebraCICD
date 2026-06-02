package handler

import (
	"github.com/ZebraOps/ZebraCICD/internal/model"

	"gorm.io/gorm"
)

type StageHistoryRepository struct {
	db *gorm.DB
}

func NewStageHistoryRepository(db *gorm.DB) *StageHistoryRepository {
	return &StageHistoryRepository{db: db}
}

// Create inserts a new StageHistory record.
func (r *StageHistoryRepository) Create(stage *model.StageHistory) error {
	return r.db.Create(stage).Error
}

// Update modifies an existing StageHistory record.
func (r *StageHistoryRepository) Update(stage *model.StageHistory) error {
	return r.db.Save(stage).Error
}

// GetByTaskID returns all StageHistory records for a given task, ordered by ID.
func (r *StageHistoryRepository) GetByTaskID(taskID uint) ([]model.StageHistory, error) {
	var stages []model.StageHistory
	err := r.db.Where("task_id = ?", taskID).Order("id ASC").Find(&stages).Error
	return stages, err
}

// GetByTaskIDAndStage returns the StageHistory for a specific task+stage combination.
func (r *StageHistoryRepository) GetByTaskIDAndStage(taskID uint, stage string) (*model.StageHistory, error) {
	var stageHistory model.StageHistory
	err := r.db.Where("task_id = ? AND stage = ?", taskID, stage).First(&stageHistory).Error
	if err != nil {
		return nil, err
	}
	return &stageHistory, nil
}