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
func (w *DeployWorker) HandleDeployTask(ctx context.Context, t *asynq.Task) error {
	var payload queue.DeployTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		// payload 格式错误不应重试，直接丢弃
		return fmt.Errorf("invalid deploy task payload: %w", asynq.SkipRetry)
	}
	return w.deploySvc.ProcessDeploymentTask(ctx, payload.TaskID)
}
