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
	"github.com/gocraft/work"
	"github.com/goharbor/harbor/src/jobservice/config"
)

// WorkerClient ...
type WorkerClient interface {
	ListWorkers(ctx context.Context, config *config.RedisPoolConfig, poolID string) ([]*Worker, error)
}

type workerClientImpl struct{}

// NewWorkerClient ...
func NewWorkerClient() *workerClientImpl {
	return &workerClientImpl{}
}

func (w *workerClientImpl) ListWorkers(ctx context.Context, config *config.RedisPoolConfig, poolID string) ([]*Worker, error) {
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
