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
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/goharbor/harbor/src/jobservice/config"
	"github.com/goharbor/harbor/src/lib/log"
	libRedis "github.com/goharbor/harbor/src/lib/redis"
)

// RedisClient defines the job service operations related to redis
type RedisClient interface {
	AllJobTypes(ctx context.Context) ([]string, error)
	AllJobTypeStatus(ctx context.Context) (map[string]bool, error)
	PauseJob(ctx context.Context, jobName string) error
	UnpauseJob(ctx context.Context, jobName string) error
	StopPendingJobs(ctx context.Context, jobType string) (jobIDs []string, err error)
}

type redisClientImpl struct {
	redisPool *redis.Pool
	namespace string
}

// NewRedisClient create a redis client
func NewRedisClient(config *config.RedisPoolConfig) (RedisClient, error) {
	pool, err := redisPool(config)
	if err != nil {
		return nil, err
	}
	return &redisClientImpl{pool, config.Namespace}, nil
}

func redisPool(config *config.RedisPoolConfig) (*redis.Pool, error) {
	return libRedis.GetRedisPool("JobService", config.RedisURL, &libRedis.PoolParam{
		PoolMaxIdle:     6,
		PoolIdleTimeout: time.Duration(config.IdleTimeoutSecond) * time.Second,
	})
}

func (r *redisClientImpl) StopPendingJobs(ctx context.Context, jobType string) (jobIDs []string, err error) {
	jobIDs = []string{}
	log.Infof("job queue cleaned up %s", jobType)
	redisKeyJobQueue := fmt.Sprintf("{%s}:jobs:%v", r.namespace, jobType)
	conn := r.redisPool.Get()
	defer conn.Close()
	jobIDs, err = redis.Strings(conn.Do("LRANGE", redisKeyJobQueue, 0, -1))
	if err != nil {
		return jobIDs, err
	}
	log.Infof("updated %d tasks in pending status to stop", len(jobIDs))
	ret, err := redis.Int64(conn.Do("DEL", redisKeyJobQueue))
	if err != nil {
		return jobIDs, err
	}
	log.Infof("deleted %d keys in waiting queue for %s", ret, jobType)
	return jobIDs, nil
}

func (r *redisClientImpl) AllJobTypes(ctx context.Context) ([]string, error) {
	conn := r.redisPool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("SMEMBERS", fmt.Sprintf("{%s}:known_jobs", r.namespace)))
}

func (r *redisClientImpl) AllJobTypeStatus(ctx context.Context) (map[string]bool, error) {
	result := map[string]bool{}
	conn := r.redisPool.Get()
	defer conn.Close()
	redisKeyJobPaused := fmt.Sprintf("{%s}:jobs:*:paused", r.namespace)
	keys, err := redis.Strings(conn.Do("KEYS", redisKeyJobPaused))
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		jobType := key[len(r.namespace)+8 : len(key)-7]
		result[jobType] = true
	}
	return result, nil
}

func (r *redisClientImpl) PauseJob(ctx context.Context, jobName string) error {
	log.Infof("pause job type:%s", jobName)
	redisKeyJobPaused := fmt.Sprintf("{%s}:jobs:%s:paused", r.namespace, jobName)
	conn := r.redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", redisKeyJobPaused, "1")
	return err
}

func (r *redisClientImpl) UnpauseJob(ctx context.Context, jobName string) error {
	log.Infof("unpause job %s", jobName)
	redisKeyJobPaused := fmt.Sprintf("{%s}:jobs:%s:paused", r.namespace, jobName)
	conn := r.redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("DEL", redisKeyJobPaused)
	return err
}
