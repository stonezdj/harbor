// Copyright (c) 2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/harbor/src/common"
	"github.com/vmware/harbor/src/ui/config"
)

func TestGetConfig(t *testing.T) {
	fmt.Println("Testing getting configurations")
	assert := assert.New(t)
	apiTest := newHarborAPI()

	//case 1: get configurations without admin role
	code, _, err := apiTest.GetConfig(*testUser)
	if err != nil {
		t.Fatalf("failed to get configurations: %v", err)
	}

	assert.Equal(401, code, "the status code of getting configurations with non-admin user should be 401")

	//case 2: get configurations with admin role
	code, cfg, err := apiTest.GetConfig(*admin)
	if err != nil {
		t.Fatalf("failed to get configurations: %v", err)
	}

	if !assert.Equal(200, code, "the status code of getting configurations with admin user should be 200") {
		return
	}

	mode := cfg[common.AUTHMode].Value.(string)
	assert.Equal(common.DBAuth, mode, fmt.Sprintf("the auth mode should be %s", common.DBAuth))
	ccc, err := config.GetSystemCfg()
	if err != nil {
		t.Logf("failed to get system configurations: %v", err)
	}
	t.Logf("%v", ccc)
}

func TestPutConfig(t *testing.T) {
	fmt.Println("Testing modifying configurations")
	assert := assert.New(t)
	apiTest := newHarborAPI()

	cfg := map[string]interface{}{
		common.TokenExpiration: 60,
	}

	code, err := apiTest.PutConfig(*admin, cfg)
	if err != nil {
		t.Fatalf("failed to get configurations: %v", err)
	}

	if !assert.Equal(200, code, "the status code of modifying configurations with admin user should be 200") {
		return
	}
	ccc, err := config.GetSystemCfg()
	if err != nil {
		t.Logf("failed to get system configurations: %v", err)
	}
	t.Logf("%v", ccc)
}

func TestResetConfig(t *testing.T) {
	fmt.Println("Testing resetting configurations")
	assert := assert.New(t)
	apiTest := newHarborAPI()

	code, err := apiTest.ResetConfig(*admin)
	if err != nil {
		t.Errorf("failed to get configurations: %v", err)
		return
	}

	if !assert.Equal(200, code, "unexpected response code") {
		return
	}

	code, cfgs, err := apiTest.GetConfig(*admin)
	if err != nil {
		t.Errorf("failed to get configurations: %v", err)
		return
	}

	if !assert.Equal(200, code, "unexpected response code") {
		return
	}

	value, ok := cfgs[common.TokenExpiration]
	if !ok {
		t.Errorf("%s not found", common.TokenExpiration)
		return
	}

	assert.Equal(int(value.Value.(float64)), 30, "unexpected 30")

	ccc, err := config.GetSystemCfg()
	if err != nil {
		t.Logf("failed to get system configurations: %v", err)
	}
	t.Logf("%v", ccc)
}

func Test_addMissingKey(t *testing.T) {

	cfg := map[string]interface{}{
		common.LDAPURL:        "sampleurl",
		common.EmailPort:      555,
		common.LDAPVerifyCert: true,
	}

	type args struct {
		cfg map[string]interface{}
	}
	tests := []struct {
		name string
		args args
	}{
		{"Add default value", args{cfg}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addMissingKey(tt.args.cfg)
		})
	}

	if _, ok := cfg[common.LDAPBaseDN]; !ok {
		t.Errorf("Can not found default value for %v", common.LDAPBaseDN)
	}

}
