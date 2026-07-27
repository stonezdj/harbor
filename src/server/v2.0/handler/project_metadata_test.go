// Copyright Project Harbor Authors
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

package handler

// Import context and other required packages
import (
	"context"
	"testing"

	"github.com/goharbor/harbor/src/controller/project"
	"github.com/goharbor/harbor/src/controller/project/metadata"
	"github.com/goharbor/harbor/src/lib/pattern"
	proModels "github.com/goharbor/harbor/src/pkg/project/models"
	"github.com/stretchr/testify/assert"
)

type fakeMetadataController struct {
	metadata.Controller
	getFunc func(ctx context.Context, projectID int64, meta ...string) (map[string]string, error)
}

func (f *fakeMetadataController) Get(ctx context.Context, projectID int64, meta ...string) (map[string]string, error) {
	if f.getFunc != nil {
		return f.getFunc(ctx, projectID, meta...)
	}
	return nil, nil
}

type fakeProjectController struct {
	project.Controller
	getFunc func(ctx context.Context, projectIDOrName any, options ...project.Option) (*proModels.Project, error)
}

func (f *fakeProjectController) Get(ctx context.Context, projectIDOrName any, options ...project.Option) (*proModels.Project, error) {
	if f.getFunc != nil {
		return f.getFunc(ctx, projectIDOrName, options...)
	}
	return nil, nil
}

func TestValidate(t *testing.T) {
	fakeCtl := &fakeMetadataController{}
	fakeProCtl := &fakeProjectController{}
	api := &projectMetadataAPI{
		ctl:    fakeCtl,
		proCtl: fakeProCtl,
	}

	tests := []struct {
		name      string
		metas     map[string]string
		expectErr bool
		setup     func()
	}{
		{
			name:      "Invalid max upstream conn value",
			metas:     map[string]string{proModels.ProMetaMaxUpstreamConn: "invalid"},
			expectErr: true,
		},
		{
			name:      "max upstream conn value 0",
			metas:     map[string]string{proModels.ProMetaMaxUpstreamConn: "0"},
			expectErr: false,
		},
		{
			name:      "max upstream conn value -1",
			metas:     map[string]string{proModels.ProMetaMaxUpstreamConn: "-1"},
			expectErr: false,
		},
		{
			name:      "normal max upstream conn value",
			metas:     map[string]string{proModels.ProMetaMaxUpstreamConn: "30"},
			expectErr: false,
		},
		{
			name:      "Unsupported key",
			metas:     map[string]string{"unsupported_key": "value"},
			expectErr: true,
		},
		{
			name:      "Empty map",
			metas:     map[string]string{},
			expectErr: true,
		},
		{
			name:      "ProxyCacheFilterPattern with KindDoublestar (valid)",
			metas:     map[string]string{proModels.ProMetaProxyCacheFilterPattern: "**"},
			expectErr: false,
			setup: func() {
				fakeProCtl.getFunc = func(ctx context.Context, projectIDOrName any, options ...project.Option) (*proModels.Project, error) {
					return &proModels.Project{RegistryID: 1}, nil
				}
				fakeCtl.getFunc = func(ctx context.Context, projectID int64, meta ...string) (map[string]string, error) {
					return map[string]string{proModels.ProMetaProxyCacheFilterKind: pattern.KindDoublestar}, nil
				}
			},
		},
		{
			name:      "ProxyCacheFilterPattern with KindDoublestar (invalid)",
			metas:     map[string]string{proModels.ProMetaProxyCacheFilterPattern: "[invalid"},
			expectErr: true,
			setup: func() {
				fakeProCtl.getFunc = func(ctx context.Context, projectIDOrName any, options ...project.Option) (*proModels.Project, error) {
					return &proModels.Project{RegistryID: 1}, nil
				}
				fakeCtl.getFunc = func(ctx context.Context, projectID int64, meta ...string) (map[string]string, error) {
					return map[string]string{proModels.ProMetaProxyCacheFilterKind: pattern.KindDoublestar}, nil
				}
			},
		},
		{
			name:      "ProxyCacheFilterPattern with KindRegex (valid)",
			metas:     map[string]string{proModels.ProMetaProxyCacheFilterPattern: "^foo/.*$"},
			expectErr: false,
			setup: func() {
				fakeProCtl.getFunc = func(ctx context.Context, projectIDOrName any, options ...project.Option) (*proModels.Project, error) {
					return &proModels.Project{RegistryID: 1}, nil
				}
				fakeCtl.getFunc = func(ctx context.Context, projectID int64, meta ...string) (map[string]string, error) {
					return map[string]string{proModels.ProMetaProxyCacheFilterKind: pattern.KindRegex}, nil
				}
			},
		},
		{
			name:      "ProxyCacheFilterPattern with KindRegex (invalid)",
			metas:     map[string]string{proModels.ProMetaProxyCacheFilterPattern: "[invalid"},
			expectErr: true,
			setup: func() {
				fakeProCtl.getFunc = func(ctx context.Context, projectIDOrName any, options ...project.Option) (*proModels.Project, error) {
					return &proModels.Project{RegistryID: 1}, nil
				}
				fakeCtl.getFunc = func(ctx context.Context, projectID int64, meta ...string) (map[string]string, error) {
					return map[string]string{proModels.ProMetaProxyCacheFilterKind: pattern.KindRegex}, nil
				}
			},
		},
		{
			name:      "ProxyCacheFilterKind with valid existing pattern",
			metas:     map[string]string{proModels.ProMetaProxyCacheFilterKind: pattern.KindRegex},
			expectErr: false,
			setup: func() {
				fakeProCtl.getFunc = func(ctx context.Context, projectIDOrName any, options ...project.Option) (*proModels.Project, error) {
					return &proModels.Project{RegistryID: 1}, nil
				}
				fakeCtl.getFunc = func(ctx context.Context, projectID int64, meta ...string) (map[string]string, error) {
					return map[string]string{proModels.ProMetaProxyCacheFilterPattern: "^foo/.*$"}, nil
				}
			},
		},
		{
			name:      "ProxyCacheFilterKind with invalid existing pattern for doublestar",
			metas:     map[string]string{proModels.ProMetaProxyCacheFilterKind: pattern.KindDoublestar},
			expectErr: true,
			setup: func() {
				fakeProCtl.getFunc = func(ctx context.Context, projectIDOrName any, options ...project.Option) (*proModels.Project, error) {
					return &proModels.Project{RegistryID: 1}, nil
				}
				fakeCtl.getFunc = func(ctx context.Context, projectID int64, meta ...string) (map[string]string, error) {
					return map[string]string{proModels.ProMetaProxyCacheFilterPattern: "[invalid"}, nil
				}
			},
		},
		{
			name:      "ProxyCacheFilterPattern on normal project (invalid)",
			metas:     map[string]string{proModels.ProMetaProxyCacheFilterPattern: "**"},
			expectErr: true,
			setup: func() {
				fakeProCtl.getFunc = func(ctx context.Context, projectIDOrName any, options ...project.Option) (*proModels.Project, error) {
					return &proModels.Project{RegistryID: 0}, nil
				}
			},
		},
		{
			name:      "ProxyCacheFilterKind on normal project (invalid)",
			metas:     map[string]string{proModels.ProMetaProxyCacheFilterKind: pattern.KindDoublestar},
			expectErr: true,
			setup: func() {
				fakeProCtl.getFunc = func(ctx context.Context, projectIDOrName any, options ...project.Option) (*proModels.Project, error) {
					return &proModels.Project{RegistryID: 0}, nil
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			} else {
				fakeCtl.getFunc = nil
				fakeProCtl.getFunc = nil
			}
			result, err := api.validate(context.TODO(), int64(1), tt.metas)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}
