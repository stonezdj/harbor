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
	"github.com/goharbor/harbor/src/jobservice/config"
	"github.com/goharbor/harbor/src/lib/log"
	libRedis "github.com/goharbor/harbor/src/lib/redis"
	"github.com/gomodule/redigo/redis"
	"time"
)

// RedisClient defines the job service operations related to redis
type RedisClient interface {
	AllJobTypes(ctx context.Context, config *config.RedisPoolConfig) ([]string, error)
	AllJobTypeStatus(ctx context.Context, config *config.RedisPoolConfig) (map[string]bool, error)
	PauseJob(ctx context.Context, config *config.RedisPoolConfig, jobName string) error
	UnpauseJob(ctx context.Context, config *config.RedisPoolConfig, jobName string) error
	StopPendingJobs(ctx context.Context, config *config.RedisPoolConfig, jobType string) (jobIDs []string, err error)
}

type RedisClientImpl struct{}

func NewRedisClient() RedisClient {
	return &RedisClientImpl{}
}

func redisPool(config *config.RedisPoolConfig) (*redis.Pool, error) {
	return libRedis.GetRedisPool("JobService", config.RedisURL, &libRedis.PoolParam{
		PoolMaxIdle:     6,
		PoolIdleTimeout: time.Duration(config.IdleTimeoutSecond) * time.Second,
	})
}

func (r *RedisClientImpl) StopPendingJobs(ctx context.Context, config *config.RedisPoolConfig, jobType string) (jobIDs []string, err error) {
	jobIDs = []string{}
	log.Infof("job queue cleaned up %s", jobType)
	redisKeyJobQueue := fmt.Sprintf("{%s}:jobs:%v", config.Namespace, jobType)
	pool, err := redisPool(config)
	if err != nil {
		return jobIDs, err
	}
	conn := pool.Get()
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

func (r *RedisClientImpl) AllJobTypes(ctx context.Context, config *config.RedisPoolConfig) ([]string, error) {
	pool, err := redisPool(config)
	if err != nil {
		return nil, err
	}
	conn := pool.Get()
	defer conn.Close()
	return redis.Strings(conn.Do("SMEMBERS", fmt.Sprintf("{%s}:known_jobs", config.Namespace)))
}

func (r *RedisClientImpl) AllJobTypeStatus(ctx context.Context, config *config.RedisPoolConfig) (map[string]bool, error) {
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

func (r *RedisClientImpl) PauseJob(ctx context.Context, config *config.RedisPoolConfig, jobName string) error {
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

func (r *RedisClientImpl) UnpauseJob(ctx context.Context, config *config.RedisPoolConfig, jobName string) error {
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
