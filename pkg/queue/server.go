package queue

import (
	"time"

	"github.com/hibiken/asynq"
)

func NewServer(addr, password string, db, concurrency int, retryDelayBase time.Duration) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     addr,
			Password: password,
			DB:       db,
		},
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				"deploy": 10,
			},
			// 指数退避，基数由配置决定
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				return time.Duration(n) * retryDelayBase
			},
		},
	)
}
