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

package driver

import (
	"encoding/json"
	"fmt"
	"github.com/goharbor/harbor/src/common"
	"github.com/goharbor/harbor/src/lib/log"
	redislib "github.com/goharbor/harbor/src/lib/redis"
	"github.com/gomodule/redigo/redis"
)

const (
	harborCfg             = "harborCfg"
	DefaultCfgCacheIntSec = 20
)

// CachedDriver to cache configure properties,
// disable cache by setting cfg_cache_interval_seconds=0
type CachedDriver struct {
	// IntervalSec - the interval to refresh cache, no cache when Interval=0
	IntervalSec int64
	Driver      Driver
}

func (c *CachedDriver) getConfig() (map[string]interface{}, error) {
	conn := redislib.DefaultPool().Get()
	defer conn.Close()
	str, err := redis.String(conn.Do("GET", harborCfg))
	if err == redis.ErrNil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var cfgMap map[string]interface{}
	err = json.Unmarshal([]byte(str), &cfgMap)
	if err != nil {
		return nil, err
	}

	return cfgMap, nil
}

func (c *CachedDriver) setConfig(cfgMap map[string]interface{}) error {
	conn := redislib.DefaultPool().Get()
	defer conn.Close()
	val, err := json.Marshal(cfgMap)
	if err != nil {
		return err
	}
	reply, err := redis.String(conn.Do("SET", harborCfg, string(val), "EX", c.IntervalSec))
	if err != nil {
		return err
	}
	if reply != "OK" {
		return fmt.Errorf("bad reply value")
	}
	return nil
}
func (c *CachedDriver) expireCache() error {
	conn := redislib.DefaultPool().Get()
	defer conn.Close()
	_, err := redis.Int64(conn.Do("DEL", harborCfg))
	if err != nil {
		return err
	}
	return nil
}

func (c *CachedDriver) refresh() error {
	log.Debug("Refresh configure properties from store.")
	cfg, err := c.Driver.Load()
	if err != nil {
		log.Errorf("Failed to load config %+v", err)
		return err
	}
	c.updateInterval(cfg)
	if c.IntervalSec == 0 {
		return nil
	}
	if err = c.setConfig(cfg); err != nil {
		log.Errorf("Failed to save to cache %v", err)
	}
	return nil
}

func (c *CachedDriver) updateInterval(cfg map[string]interface{}) {
	i, exist := cfg[common.CfgCacheIntervalSeconds]
	if !exist {
		return
	}
	sec, ok := i.(int64)
	if !ok || sec < 0 {
		return
	}
	c.IntervalSec = sec
}

// Load - load config item
func (c *CachedDriver) Load() (map[string]interface{}, error) {
	if c.IntervalSec == 0 {
		cfg, err := c.Driver.Load()
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}
	cfg, err := c.getConfig()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		err := c.refresh()
		if err != nil {
			return nil, err
		}
	}
	if cfg != nil {
		return cfg, nil
	}
	return c.getConfig()
}

// Save - save config item into config driver
func (c *CachedDriver) Save(cfg map[string]interface{}) error {
	err := c.Driver.Save(cfg)
	if err != nil {
		return err
	}
	if c.IntervalSec == 0 {
		return nil
	}
	return c.expireCache()
}
