package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

const TypeDeployTask = "deploy:process"

type DeployTaskPayload struct {
	TaskID uint `json:"task_id"`
}

type Client struct {
	client *asynq.Client
}

func NewClient(addr, password string, db int) *Client {
	return &Client{
		client: asynq.NewClient(asynq.RedisClientOpt{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func (c *Client) Close() error {
	return c.client.Close()
}

// EnqueueDeployTask 将部署任务入队。
// asynq.TaskID 保证同一 taskID 不会被重复入队（幂等），重复调用返回 asynq.ErrTaskIDConflict。
func (c *Client) EnqueueDeployTask(taskID uint) error {
	payload, err := json.Marshal(DeployTaskPayload{TaskID: taskID})
	if err != nil {
		return fmt.Errorf("marshal deploy payload: %w", err)
	}
	task := asynq.NewTask(
		TypeDeployTask,
		payload,
		asynq.TaskID(fmt.Sprintf("deploy:%d", taskID)),
		asynq.MaxRetry(3),
		asynq.Timeout(35*time.Minute),
		asynq.Queue("deploy"),
	)
	_, err = c.client.Enqueue(task)
	return err
}
