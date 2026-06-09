package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ZebraOps/ZebraCICD/internal/service"
	"github.com/ZebraOps/ZebraCICD/pkg/queue"
	"github.com/hibiken/asynq"
)

type DeployWorker struct {
	deploySvc *service.DeployService
}

func NewDeployWorker(deploySvc *service.DeployService) *DeployWorker {
	return &DeployWorker{deploySvc: deploySvc}
}

// HandleDeployTask 是 Asynq worker 的入口，由 asynq.Server 调用。
// 处理全流程部署任务（构建 + 部署）
func (w *DeployWorker) HandleDeployTask(ctx context.Context, t *asynq.Task) error {
	var payload queue.DeployTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		// payload 格式错误不应重试，直接丢弃
		return fmt.Errorf("invalid deploy task payload: %w", asynq.SkipRetry)
	}
	return w.deploySvc.ProcessDeploymentTask(ctx, payload.TaskID)
}

// HandleBuildTask 处理仅构建任务（手动执行模式）
func (w *DeployWorker) HandleBuildTask(ctx context.Context, t *asynq.Task) error {
	var payload queue.DeployTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("invalid build task payload: %w", asynq.SkipRetry)
	}
	return w.deploySvc.ProcessBuildTask(ctx, payload.TaskID)
}

// HandleDeployOnlyTask 处理仅部署任务（手动执行模式）
func (w *DeployWorker) HandleDeployOnlyTask(ctx context.Context, t *asynq.Task) error {
	var payload queue.DeployTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("invalid deploy-only task payload: %w", asynq.SkipRetry)
	}
	return w.deploySvc.ProcessDeployOnlyTask(ctx, payload.TaskID)
}