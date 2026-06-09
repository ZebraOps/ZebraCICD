package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// 任务类型常量
const (
	TypeDeployTask     = "deploy:process"     // 全流程执行（构建+部署）
	TypeBuildTask      = "deploy:build"       // 仅执行构建
	TypeDeployOnlyTask = "deploy:deploy_only" // 仅执行部署
)

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

// EnqueueDeployTaskRetry 将重试的部署任务入队。
// 使用带时间戳的 TaskID 避免与原始 deploy:{id} 任务冲突。
func (c *Client) EnqueueDeployTaskRetry(taskID uint) error {
	payload, err := json.Marshal(DeployTaskPayload{TaskID: taskID})
	if err != nil {
		return fmt.Errorf("marshal deploy payload: %w", err)
	}
	task := asynq.NewTask(
		TypeDeployTask,
		payload,
		asynq.TaskID(fmt.Sprintf("deploy:%d:retry-%d", taskID, time.Now().Unix())),
		asynq.MaxRetry(3),
		asynq.Timeout(35*time.Minute),
		asynq.Queue("deploy"),
	)
	_, err = c.client.Enqueue(task)
	return err
}

// EnqueueDeployTaskAt 在指定时间执行部署任务（定时执行）。
func (c *Client) EnqueueDeployTaskAt(taskID uint, scheduledAt *time.Time) error {
	payload, err := json.Marshal(DeployTaskPayload{TaskID: taskID})
	if err != nil {
		return fmt.Errorf("marshal deploy payload: %w", err)
	}

	opts := []asynq.Option{
		asynq.TaskID(fmt.Sprintf("deploy:%d:scheduled", taskID)),
		asynq.MaxRetry(3),
		asynq.Timeout(35 * time.Minute),
		asynq.Queue("deploy"),
	}

	if scheduledAt != nil {
		opts = append(opts, asynq.ProcessAt(*scheduledAt))
	}

	task := asynq.NewTask(TypeDeployTask, payload, opts...)
	_, err = c.client.Enqueue(task)
	return err
}

// EnqueueBuildTask 仅执行构建阶段。
func (c *Client) EnqueueBuildTask(taskID uint) error {
	payload, err := json.Marshal(DeployTaskPayload{TaskID: taskID})
	if err != nil {
		return fmt.Errorf("marshal build payload: %w", err)
	}
	task := asynq.NewTask(
		TypeBuildTask,
		payload,
		asynq.TaskID(fmt.Sprintf("build:%d:%d", taskID, time.Now().Unix())),
		asynq.MaxRetry(3),
		asynq.Timeout(20*time.Minute),
		asynq.Queue("deploy"),
	)
	_, err = c.client.Enqueue(task)
	return err
}

// EnqueueDeployOnlyTask 仅执行部署阶段（构建已完成）。
func (c *Client) EnqueueDeployOnlyTask(taskID uint) error {
	payload, err := json.Marshal(DeployTaskPayload{TaskID: taskID})
	if err != nil {
		return fmt.Errorf("marshal deploy-only payload: %w", err)
	}
	task := asynq.NewTask(
		TypeDeployOnlyTask,
		payload,
		asynq.TaskID(fmt.Sprintf("deploy-only:%d:%d", taskID, time.Now().Unix())),
		asynq.MaxRetry(3),
		asynq.Timeout(15*time.Minute),
		asynq.Queue("deploy"),
	)
	_, err = c.client.Enqueue(task)
	return err
}
