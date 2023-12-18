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
	"fmt"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/custompatch"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

func TestCustompatch(t *testing.T) { //nolint:funlen
	t.Parallel()

	patch := custompatch.Patch{}

	type testCase struct {
		Error         bool
		Ignore        bool
		CustomPatches types.PatchOperation
		Expected      types.PatchOperation
	}

	tests := []testCase{
		{
			CustomPatches: types.PatchOperation{
				Op:    "{{ .ContainerName }}",
				Path:  "{{ .PodContainer.ContainerPath }}/annotations",
				Value: nil,
			},
			Expected: types.PatchOperation{
				Op:    "test",
				Path:  "/spec/containers/123/annotations",
				Value: nil,
			},
		},
		{
			Ignore: true,
			CustomPatches: types.PatchOperation{
				Op:    "remove",
				Path:  "/spec/affinity",
				Value: nil,
			},
		},
		{
			Ignore: true,
			CustomPatches: types.PatchOperation{
				Op:    "remove",
				Path:  "/spec/nodeselector",
				Value: nil,
			},
		},
		{
			CustomPatches: types.PatchOperation{
				Op:    "remove",
				Path:  "/spec/test",
				Value: nil,
			},
			Expected: types.PatchOperation{
				Op:    "remove",
				Path:  "/spec/test",
				Value: nil,
			},
		},
		{
			Error: true,
			CustomPatches: types.PatchOperation{
				Op: "{{ .FFFFAAAKKEEE }}",
			},
		},
		{
			Error: true,
			CustomPatches: types.PatchOperation{
				Path: "{{ .FFFFAAAKKEEE }}",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(fmt.Sprintf("%+v", test), func(t *testing.T) {
			t.Parallel()

			containerInfo := &types.ContainerInfo{
				ContainerName: "test",
				PodContainer: &types.PodContainer{
					Order: 123,
					Type:  "container",
					Pod: &corev1.Pod{
						Spec: corev1.PodSpec{
							Affinity:     nil,
							NodeSelector: nil,
						},
					},
				},
				SelectedRules: []*types.Rule{
					{
						CustomPatches: []types.PatchOperation{test.CustomPatches},
					},
				},
			}

			patchOps, err := patch.Create(context.TODO(), containerInfo)
			if test.Error && err != nil {
				t.Skip("its ok")
			}

			if err != nil {
				t.Fatal(err)
			}

			if test.Ignore && len(patchOps) == 0 {
				return
			}

			if len(patchOps) != 1 {
				t.Fatal("1 patch must be created")
			}

			if patchOps[0].Op != test.Expected.Op || patchOps[0].Path != test.Expected.Path {
				t.Fatalf("not corrected patch %s", patchOps[0].String())
			}
		})
	}
}
