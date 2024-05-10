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
		Container     *corev1.Container
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
				Path:  "/spec/containers/1/annotations",
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
		{
			Container: &corev1.Container{
				Name:           "test",
				ReadinessProbe: &corev1.Probe{},
			},
			CustomPatches: types.PatchOperation{
				Op:   "remove",
				Path: "/spec/containers/1/readinessProbe",
			},
			Expected: types.PatchOperation{
				Op:   "remove",
				Path: "/spec/containers/1/readinessProbe",
			},
		},
		{
			Ignore: true,
			Container: &corev1.Container{
				Name: "test",
			},
			CustomPatches: types.PatchOperation{
				Op:   "remove",
				Path: "/spec/containers/1/readinessProbe",
			},
		},
		{
			Container: &corev1.Container{
				Name:          "test",
				LivenessProbe: &corev1.Probe{},
			},
			CustomPatches: types.PatchOperation{
				Op:   "remove",
				Path: "/spec/containers/1/livenessProbe",
			},
			Expected: types.PatchOperation{
				Op:   "remove",
				Path: "/spec/containers/1/livenessProbe",
			},
		},
		{
			Container: &corev1.Container{
				Name: "testName",
			},
			CustomPatches: types.PatchOperation{
				Op:   "op={{ .PodContainer.Container.Name }}",
				Path: "path={{ .PodContainer.Container.Name }}",
				Value: []string{
					"test1={{ .PodContainer.Container.Name }}",
				},
			},
			Expected: types.PatchOperation{
				Op:   "op=testName",
				Path: "path=testName",
				Value: []string{
					"test1=testName",
				},
			},
		},
		{
			Ignore: true,
			Container: &corev1.Container{
				Name: "test",
			},
			CustomPatches: types.PatchOperation{
				Op:   "remove",
				Path: "/spec/containers/1/livenessProbe",
			},
		},
		{
			Ignore: true,
			Container: &corev1.Container{
				Name: "test",
			},
			CustomPatches: types.PatchOperation{
				Op:   "remove",
				Path: "/spec/topologySpreadConstraints",
			},
		},
		{
			Ignore: true,
			Container: &corev1.Container{
				Name: "test",
			},
			CustomPatches: types.PatchOperation{
				Op:   "remove",
				Path: "{{ .PodContainer.ContainerPath }}/readinessProbe",
			},
		},
		{
			Error: true,
			CustomPatches: types.PatchOperation{
				Value: make(chan int),
			},
		},
		{
			Error: true,
			Container: &corev1.Container{
				Name: "\"",
			},
			CustomPatches: types.PatchOperation{
				Op: "{{ .PodContainer.Container.Name }}",
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%+v", test), func(t *testing.T) {
			t.Parallel()

			containerInfo := &types.ContainerInfo{
				ContainerName: "test",
				PodContainer: &types.PodContainer{
					Order:     1,
					Type:      "container",
					Container: test.Container,
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

			if patchOps[0].String() != test.Expected.String() {
				t.Fatalf("not corrected patch=%s, expected=%s", patchOps[0].String(), test.Expected.String())
			}
		})
	}
}
