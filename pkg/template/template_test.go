/*
Copyright paskal.maksim@gmail.com
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package template_test

import (
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/template"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
)

func TestTemplateValue(t *testing.T) {
	t.Parallel()

	containerInfo := types.ContainerInfo{
		Image: "/a/b/c/d:e",
	}

	cases := make(map[string]string)

	cases[`{{ index (regexp "/(.+):(.+)$" .Image) 2 }}`] = "e"
	cases[`{{ GetSentryDSN (index (regexp "/(.+):(.+)$" .Image) 1) }}`] = ""
	cases[`{{ index (regexp "/(.+):(.+)$" "/1/2/3/4:main") 2 }}`] = "main"
	cases[`{{ indexUnknown (regexp "/(.+):(.+)$" "/1/2/3/4:main") 3 }}`] = "unknown"
	cases[`{{ indexUnknown (regexp "/(.+):(.+)$" .Image) 2 }}`] = "e"

	for k, v := range cases {
		value, err := template.Get(containerInfo, k)
		if err != nil {
			t.Fatal(err)
		}

		if value != v {
			t.Fatalf("must be %s, got=%s", v, value)
		}
	}
}
