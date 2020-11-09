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
	"github.com/goharbor/harbor/src/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

var TestConfig = map[string]interface{}{
	common.ExtEndpoint:        "host01.com",
	common.AUTHMode:           "ldap_auth",
	common.DatabaseType:       "postgresql",
	common.PostGreSQLHOST:     "127.0.0.1",
	common.PostGreSQLPort:     5432,
	common.PostGreSQLUsername: "postgres",
	common.PostGreSQLPassword: "root123",
	common.PostGreSQLDatabase: "registry",
	// config.SelfRegistration: true,
	common.LDAPURL:              "ldap://127.0.0.1",
	common.LDAPSearchDN:         "cn=admin,dc=example,dc=com",
	common.LDAPSearchPwd:        "admin",
	common.LDAPBaseDN:           "dc=example,dc=com",
	common.LDAPUID:              "uid",
	common.LDAPFilter:           "",
	common.LDAPScope:            3,
	common.LDAPTimeout:          30,
	common.AdminInitialPassword: "password",
}

func TestCachedDriver_updateInterval(t *testing.T) {
	type fields struct {
		IntervalSec int64
		Driver      Driver
	}
	type args struct {
		cfg map[string]interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		expect int64
	}{
		{name: "zero", fields: fields{IntervalSec: 0}, args: args{cfg: map[string]interface{}{common.CfgCacheIntervalSeconds: 0}}, expect: 0},
		{name: "normal", fields: fields{IntervalSec: 10}, args: args{cfg: map[string]interface{}{common.CfgCacheIntervalSeconds: 10}}, expect: 10},
		{name: "negative", fields: fields{IntervalSec: -10}, args: args{cfg: map[string]interface{}{common.CfgCacheIntervalSeconds: -10}}, expect: -10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CachedDriver{
				IntervalSec: tt.fields.IntervalSec,
				Driver:      tt.fields.Driver,
			}
			c.updateInterval(tt.args.cfg)
			assert.Equal(t, tt.expect, c.IntervalSec)
		})
	}
}

func TestCachePutAndGet(t *testing.T) {
	cd := &CachedDriver{
		Driver:      &Database{},
		IntervalSec: DefaultCfgCacheIntSec}
	cfgMap := map[string]interface{}{common.LDAPBaseDN: "dc=example,dc=com"}
	cd.setConfig(cfgMap)
	another, err := cd.getConfig()
	assert.Nil(t, err)
	assert.Equal(t, another[common.LDAPBaseDN], "dc=example,dc=com")
}

func TestCacheSaveAndLoad(t *testing.T) {
	cd := &CachedDriver{
		Driver:      &Database{},
		IntervalSec: DefaultCfgCacheIntSec}
	cfgMap := map[string]interface{}{common.LDAPGroupBaseDN: "dc=group,dc=example,dc=com"}
	err := cd.Save(cfgMap)
	assert.Nil(t, err)
	another, err := cd.Load()
	assert.Nil(t, err)
	assert.Equal(t, another[common.LDAPGroupBaseDN], "dc=group,dc=example,dc=com")
}

func BenchmarkCachedDriver_Load_WithCache(b *testing.B) {
	cd := &CachedDriver{
		Driver:      &Database{},
		IntervalSec: DefaultCfgCacheIntSec}
	cd.Save(TestConfig)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cd.Load()
	}
}

func BenchmarkCachedDriver_Load_NoCache(b *testing.B) {
	cd := &CachedDriver{
		Driver:      &Database{},
		IntervalSec: 0}
	cd.Save(TestConfig)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		cd.Load()
	}
}

func BenchmarkDBDriver_Load_NoCache(b *testing.B) {
	driver := &Database{}
	driver.Save(TestConfig)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		driver.Load()
	}
}
