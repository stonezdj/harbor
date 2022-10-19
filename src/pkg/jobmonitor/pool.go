//  Copyright Project Harbor Authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package jobmonitor

import (
	"context"
	"fmt"
	"github.com/gocraft/work"
	"github.com/goharbor/harbor/src/jobservice/config"
	libRedis "github.com/goharbor/harbor/src/lib/redis"
	"time"
)

// PoolManager the interface to retrieve job service monitor metrics
type PoolManager interface {
	List(ctx context.Context, config *config.RedisPoolConfig) ([]*WorkerPool, error)
}

type poolManager struct {
}

func (p poolManager) List(ctx context.Context, config *config.RedisPoolConfig) ([]*WorkerPool, error) {
	client, err := jsClient(config)
	if err != nil {
		return nil, err
	}
	workerPool := make([]*WorkerPool, 0)
	wh, err := client.WorkerPoolHeartbeats()
	if err != nil {
		return workerPool, err
	}
	for _, w := range wh {
		wp := &WorkerPool{
			PoolID:      w.WorkerPoolID,
			PID:         w.Pid,
			StartAt:     w.StartedAt,
			Concurency:  int(w.Concurrency),
			Host:        w.Host,
			HeartbeatAt: w.HeartbeatAt,
		}
		workerPool = append(workerPool, wp)
	}
	return workerPool, nil
}

// NewPoolManager create a PoolManager with namespace and redis Pool
func NewPoolManager() PoolManager {
	return &poolManager{}
}

func jsClient(config *config.RedisPoolConfig) (*work.Client, error) {
	pool, err := libRedis.GetRedisPool("JobService", config.RedisURL, &libRedis.PoolParam{
		PoolMaxIdle:     6,
		PoolIdleTimeout: time.Duration(config.IdleTimeoutSecond) * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return work.NewClient(fmt.Sprintf("{%s}", config.Namespace), pool), nil
}
