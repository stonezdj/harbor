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

package repoproxy

import (
	"fmt"
	"github.com/docker/distribution/manifest/manifestlist"
	"testing"
)

func TestTrimManifest(t *testing.T) {
	man := &manifestlist.DeserializedManifestList{}
	content := `{"manifests":[{"digest":"sha256:fd4a8673d0344c3a7f427fe4440d4b8dfd4fa59cfabbd9098f9eb0cb4ba905d0","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"amd64","os":"linux"},"size":527},{"digest":"sha256:6cc0997f14702efd436598fa83a4f7ddc7be6d2d9e8e3b6f94b63d3389aeb8c4","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"arm","os":"linux","variant":"v5"},"size":527},{"digest":"sha256:fb1f7b885314372fa4aecbb2ba98a70dfed1d60d85e1191075ee616e55df577b","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"arm","os":"linux","variant":"v6"},"size":527},{"digest":"sha256:0ed7a4588573e91f1601ef93449136a54f57a9277d67835eded3818d873cb6f8","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"arm","os":"linux","variant":"v7"},"size":527},{"digest":"sha256:1ee006886991ad4689838d3a288e0dd3fd29b70e276622f16b67a8922831a853","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"arm64","os":"linux","variant":"v8"},"size":527},{"digest":"sha256:999f1137906d82f896a70c18ed63d2797a1562cd7d4d2c1907f681b35c30459d","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"386","os":"linux"},"size":527},{"digest":"sha256:1a41828fc1a347d7061f7089d6f0c94e5a056a3c674714712a1481a4a33eb56f","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"mips64le","os":"linux"},"size":527},{"digest":"sha256:9ab66e8f62e49ccb7f67234d89b86e315b6bea18b90d5264259f8bba9a7716df","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"ppc64le","os":"linux"},"size":528},{"digest":"sha256:91c15b1ba6f408a648be60f8c047ef79058f26fa640025f374281f31c8704387","mediaType":"application\/vnd.docker.distribution.manifest.v2+json","platform":{"architecture":"s390x","os":"linux"},"size":528}],"mediaType":"application\/vnd.docker.distribution.manifest.list.v2+json","schemaVersion":2}`
	man.UnmarshalJSON([]byte(content))
	for _, m := range man.Manifests {
		fmt.Printf("os:%v, arch:%v\n", m.Platform.OS, m.Platform.Architecture)
	}
	fmt.Println("--------After trimed--------")
	result, err := TrimManifestList(man, "linux", "amd64", "")
	if err != nil {
		t.Error(err)
	}
	if v, ok := result.(manifestlist.DeserializedManifestList); ok {
		for _, m := range v.Manifests {
			fmt.Printf("os:%v, arch:%v\n", m.Platform.OS, m.Platform.Architecture)
		}
	}
	mt, p, _ := result.Payload()
	fmt.Printf("mediatype: %v, payload:%v\n", mt, string(p))

}
