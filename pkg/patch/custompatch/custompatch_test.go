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
package custompatch_test

import (
	"context"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/custompatch"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
)

func TestCustompatch(t *testing.T) {
	t.Parallel()

	patch := custompatch.Patch{}

	containerInfo := &types.ContainerInfo{
		ContainerName: "test",
		PodContainer: &types.PodContainer{
			Order: 123,
			Type:  "container",
		},
		SelectedRules: []*types.Rule{
			{
				CustomPatches: []types.PatchOperation{
					{
						Op:    "{{ .ContainerName }}",
						Path:  "{{ .PodContainer.ContainerPath }}/annotations",
						Value: nil,
					},
				},
			},
		},
	}

	patchOps, err := patch.Create(context.TODO(), containerInfo)
	if err != nil {
		t.Fatal(err)
	}

	if len(patchOps) != 1 {
		t.Fatal("1 patch must be created")
	}

	if patchOps[0].Op != "test" || patchOps[0].Path != "/spec/containers/123/annotations" {
		t.Fatalf("not corrected patch %s", patchOps[0].String())
	}
}
