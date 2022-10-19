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
	"github.com/goharbor/harbor/src/lib/log"
	"github.com/goharbor/harbor/src/lib/q"
	libRedis "github.com/goharbor/harbor/src/lib/redis"
	"github.com/goharbor/harbor/src/pkg/task"
	"github.com/gomodule/redigo/redis"
	"time"
)

// PoolManager the interface to retrieve job service monitor metrics
type PoolManager interface {
	List(ctx context.Context, config *config.RedisPoolConfig) ([]*WorkerPool, error)
}

type poolManager struct {
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

func redisPool(config *config.RedisPoolConfig) (*redis.Pool, error) {
	return libRedis.GetRedisPool("JobService", config.RedisURL, &libRedis.PoolParam{
		PoolMaxIdle:     6,
		PoolIdleTimeout: time.Duration(config.IdleTimeoutSecond) * time.Second,
	})
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

// JobServiceClient ...
type JobServiceClient interface {
	ListWorkers(ctx context.Context, config *config.RedisPoolConfig, poolID string) ([]*Worker, error)
	ListQueues(ctx context.Context, config *config.RedisPoolConfig) ([]*Queue, error)
	StopPendingJobs(ctx context.Context, config *config.RedisPoolConfig, jobType string) error
	PauseJob(ctx context.Context, config *config.RedisPoolConfig, jobName string) error
	UnpauseJob(ctx context.Context, config *config.RedisPoolConfig, jobName string) error
	AllJobTypes(ctx context.Context, config *config.RedisPoolConfig) ([]string, error)
	AllJobTypeStatus(ctx context.Context, config *config.RedisPoolConfig) (map[string]bool, error)
}

type RedisClient interface {
	AllJobTypes(ctx context.Context, config *config.RedisPoolConfig) ([]string, error)
	AllJobTypeStatus(ctx context.Context, config *config.RedisPoolConfig) (map[string]bool, error)
	PauseJob(ctx context.Context, config *config.RedisPoolConfig, jobName string) error
	UnpauseJob(ctx context.Context, config *config.RedisPoolConfig, jobName string) error
}

type RedisClientImpl struct {
	redisConfig *config.RedisPoolConfig
}

func (r RedisClientImpl) AllJobTypes(ctx context.Context, config *config.RedisPoolConfig) ([]string, error) {
	pool, err := redisPool(config)
	if err != nil {
		return nil, err
	}
	conn := pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("SMEMBERS", fmt.Sprintf("{%s}:known_jobs", config.Namespace)))
}

func (r RedisClientImpl) AllJobTypeStatus(ctx context.Context, config *config.RedisPoolConfig) (map[string]bool, error) {
	result := map[string]bool{}
	pool, err := redisPool(config)
	if err != nil {
		return nil, err
	}
	conn := pool.Get()
	defer conn.Close()
	redisKeyJobPaused := fmt.Sprintf("{%s}:jobs:*:paused", config.Namespace)
	keys, err := redis.Strings(conn.Do("KEYS", redisKeyJobPaused))
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		jobType := key[len(config.Namespace)+8 : len(key)-7]
		result[jobType] = true
	}
	return result, nil
}

func (r RedisClientImpl) PauseJob(ctx context.Context, config *config.RedisPoolConfig, jobName string) error {
	log.Infof("pause job type:%s", jobName)
	pool, err := redisPool(config)
	if err != nil {
		return err
	}
	redisKeyJobPaused := fmt.Sprintf("{%s}:jobs:%s:paused", config.Namespace, jobName)
	conn := pool.Get()
	defer conn.Close()
	_, err = conn.Do("SET", redisKeyJobPaused, "1")
	return err
}

func (r RedisClientImpl) UnpauseJob(ctx context.Context, config *config.RedisPoolConfig, jobName string) error {
	log.Infof("unpause job %s", jobName)
	pool, err := redisPool(config)
	if err != nil {
		return err
	}
	redisKeyJobPaused := fmt.Sprintf("{%s}:jobs:%s:paused", config.Namespace, jobName)
	conn := pool.Get()
	defer conn.Close()
	_, err = conn.Do("DEL", redisKeyJobPaused)
	return err
}

type jobServiceClientImpl struct {
	taskMgr task.Manager
}

func (w *jobServiceClientImpl) AllJobTypeStatus(ctx context.Context, config *config.RedisPoolConfig) (map[string]bool, error) {
	result := map[string]bool{}
	pool, err := redisPool(config)
	if err != nil {
		return nil, err
	}
	conn := pool.Get()
	defer conn.Close()
	redisKeyJobPaused := fmt.Sprintf("{%s}:jobs:*:paused", config.Namespace)
	keys, err := redis.Strings(conn.Do("KEYS", redisKeyJobPaused))
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		jobType := key[len(config.Namespace)+8 : len(key)-7]
		result[jobType] = true
	}
	return result, nil
}

func (w *jobServiceClientImpl) PauseJob(ctx context.Context, config *config.RedisPoolConfig, jobName string) error {
	log.Infof("pause job type:%s", jobName)
	pool, err := redisPool(config)
	if err != nil {
		return err
	}
	redisKeyJobPaused := fmt.Sprintf("{%s}:jobs:%s:paused", config.Namespace, jobName)
	conn := pool.Get()
	defer conn.Close()
	_, err = conn.Do("SET", redisKeyJobPaused, "1")
	return err
}

func (w *jobServiceClientImpl) UnpauseJob(ctx context.Context, config *config.RedisPoolConfig, jobName string) error {
	log.Infof("unpause job %s", jobName)
	pool, err := redisPool(config)
	if err != nil {
		return err
	}
	redisKeyJobPaused := fmt.Sprintf("{%s}:jobs:%s:paused", config.Namespace, jobName)
	conn := pool.Get()
	defer conn.Close()
	_, err = conn.Do("DEL", redisKeyJobPaused)
	return err
}

func (w *jobServiceClientImpl) StopPendingJobs(ctx context.Context, config *config.RedisPoolConfig, jobType string) error {
	log.Infof("job queue cleaned up %s", jobType)
	redisKeyJobQueue := fmt.Sprintf("{%s}:jobs:%v", config.Namespace, jobType)
	pool, err := redisPool(config)
	if err != nil {
		return err
	}
	conn := pool.Get()
	defer conn.Close()
	jobIDs, err := redis.Strings(conn.Do("LRANGE", redisKeyJobQueue, 0, -1))
	if err != nil {
		return err
	}
	if err := w.UpdateJobStatusInTask(ctx, jobIDs); err != nil {
		return err
	}
	log.Infof("updated %d tasks in pending status to stop", len(jobIDs))
	ret, err := redis.Int64(conn.Do("DEL", redisKeyJobQueue))
	if err != nil {
		return err
	}
	log.Infof("deleted %d keys in waiting queue for %s", ret, jobType)
	return nil
}

func (w *jobServiceClientImpl) AllJobTypes(ctx context.Context, config *config.RedisPoolConfig) ([]string, error) {
	pool, err := redisPool(config)
	if err != nil {
		return nil, err
	}
	conn := pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("SMEMBERS", fmt.Sprintf("{%s}:known_jobs", config.Namespace)))
}

func (w *jobServiceClientImpl) UpdateJobStatusInTask(ctx context.Context, jobIDs []string) error {
	for _, jobID := range jobIDs {
		ts, err := w.taskMgr.List(ctx, q.New(q.KeyWords{"job_id": jobID}))
		if err != nil {
			return err
		}
		if len(ts) == 0 {
			continue
		}
		ts[0].Status = "Stopped"
		if err := w.taskMgr.Update(ctx, ts[0], "Status"); err != nil {
			return err
		}
	}
	return nil
}

func (w *jobServiceClientImpl) ListQueues(ctx context.Context, config *config.RedisPoolConfig) ([]*Queue, error) {
	resultQueues := make([]*Queue, 0)
	client, err := jsClient(config)
	if err != nil {
		return nil, err
	}
	queues, err := client.Queues()
	if err != nil {
		return nil, err
	}
	for _, q := range queues {
		resultQueues = append(resultQueues, &Queue{
			JobType: q.JobName,
			Count:   q.Count,
			Latency: q.Latency,
		})
	}
	return resultQueues, nil
}

func (w *jobServiceClientImpl) ListWorkers(ctx context.Context, config *config.RedisPoolConfig, poolID string) ([]*Worker, error) {
	client, err := jsClient(config)
	if err != nil {
		return nil, err
	}

	wphs, err := client.WorkerPoolHeartbeats()
	if err != nil {
		return nil, err
	}
	workerPoolMap := make(map[string]string)
	for _, wph := range wphs {
		for _, id := range wph.WorkerIDs {
			workerPoolMap[id] = wph.WorkerPoolID
		}
	}

	workers, err := client.WorkerObservations()
	if err != nil {
		return nil, err
	}
	if poolID == "all" {
		return convertToWorker(workers, workerPoolMap), nil
	}
	// filter workers by pool id
	filteredWorkers := make([]*work.WorkerObservation, 0)
	for _, w := range workers {
		if workerPoolMap[w.WorkerID] == poolID {
			filteredWorkers = append(filteredWorkers, w)
		}
	}
	return convertToWorker(filteredWorkers, workerPoolMap), nil
}

func convertToWorker(workers []*work.WorkerObservation, workerPoolMap map[string]string) []*Worker {
	wks := make([]*Worker, 0)
	for _, w := range workers {
		wks = append(wks, &Worker{
			WorkerID:  w.WorkerID,
			PoolID:    workerPoolMap[w.WorkerID],
			IsBusy:    w.IsBusy,
			JobName:   w.JobName,
			JobID:     w.JobID,
			StartedAt: w.StartedAt,
			CheckIn:   w.Checkin,
			CheckInAt: w.CheckinAt,
		})
	}
	return wks
}

// NewJobServiceClient ...
func NewJobServiceClient() *jobServiceClientImpl {
	return &jobServiceClientImpl{taskMgr: task.NewManager()}
}
