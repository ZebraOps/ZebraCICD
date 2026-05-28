package queue

import (
	"time"

	"github.com/hibiken/asynq"
)

func NewServer(addr, password string, db, concurrency int) *asynq.Server {
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
			// 指数退避：30s, 60s, 90s
			RetryDelayFunc: func(n int, e error, t *asynq.Task) time.Duration {
				return time.Duration(n) * 30 * time.Second
			},
		},
	)
}
