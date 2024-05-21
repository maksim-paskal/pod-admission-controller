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
package patch_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/patch"
	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/env"
	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/imagehost"
	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/nonroot"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
)

func TestNewPatch(t *testing.T) {
	t.Parallel()

	containerInfo := &types.ContainerInfo{
		Image: &types.ContainerImage{},
	}

	patchOps, err := patch.NewPatch(context.TODO(), containerInfo)
	if err != nil {
		t.Fatal(err)
	}

	if len(patchOps) != 0 {
		t.Fatal("len(patchOps) != 0")
	}
}

func TestIgnorePatch(t *testing.T) { //nolint:funlen
	t.Parallel()

	type Test struct {
		Patch         patch.Patch
		ContainerName string
		PodAnnotations,
		NamespaceAnnotations map[string]string
		Ignore bool
	}

	tests := []Test{
		{
			Patch:         &env.Patch{},
			ContainerName: "test",
			PodAnnotations: map[string]string{
				"pod-admission-controller/ignore-env": "test",
			},
			Ignore: true,
		},
		{
			Patch:         &env.Patch{},
			ContainerName: "test1",
			PodAnnotations: map[string]string{
				"pod-admission-controller/ignore-env": "test",
			},
			Ignore: false,
		},
		{
			Patch:         &env.Patch{},
			ContainerName: "test",
			PodAnnotations: map[string]string{
				"fake": "test",
			},
			Ignore: false,
		},
		{
			Patch:         &nonroot.Patch{},
			ContainerName: "test",
			PodAnnotations: map[string]string{
				"pod-admission-controller/ignore-nonroot": "test",
			},
			Ignore: true,
		},
		{
			Patch:         &imagehost.Patch{},
			ContainerName: "test",
			PodAnnotations: map[string]string{
				"pod-admission-controller/ignore-imagehost": "test",
			},
			Ignore: true,
		},
		{
			Patch:         &imagehost.Patch{},
			ContainerName: "test",
			PodAnnotations: map[string]string{
				"pod-admission-controller/ignore-imagehost": "*",
			},
			Ignore: true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%+v", test), func(t *testing.T) {
			t.Parallel()

			containerInfo := &types.ContainerInfo{
				PodAnnotations: map[string]string{
					"pod-admission-controller/ignore-env": "test",
				},
			}

			containerInfo.ContainerName = test.ContainerName
			containerInfo.NamespaceAnnotations = test.NamespaceAnnotations
			containerInfo.PodAnnotations = test.PodAnnotations

			if patch.IgnoreContainerPatch(test.Patch, containerInfo) != test.Ignore {
				t.Fatal("patch.IgnorePatch(&patch.EnvPatch{}, containerInfo) != test.Ignore")
			}
		})
	}
}
