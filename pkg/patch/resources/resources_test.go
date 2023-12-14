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
package resources_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/resources"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const addOperation = "add"

func TestNullResources(t *testing.T) {
	t.Parallel()

	patch := resources.Patch{}

	containerInfo := &types.ContainerInfo{
		PodContainer: &types.PodContainer{
			Type: "container",
			Container: &corev1.Container{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1"),
					},
				},
			},
		},
		SelectedRules: []*types.Rule{
			{
				AddDefaultResources: types.AddDefaultResources{
					Enabled:         true,
					RemoveResources: true,
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

	if patchOps[0].Op != "remove" || patchOps[0].Path != "/spec/containers/0/resources" {
		t.Fatalf("not corrected op %s", patchOps[0].Op)
	}
}

func TestCreateDefaultResourcesPatch(t *testing.T) {
	t.Parallel()

	patch := resources.Patch{}

	containerInfo := &types.ContainerInfo{
		PodContainer: &types.PodContainer{
			Type: "container",
			Container: &corev1.Container{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1"),
					},
				},
			},
		},
		SelectedRules: []*types.Rule{
			{
				AddDefaultResources: types.AddDefaultResources{
					Enabled: true,
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

	if patchOps[0].Op != addOperation || patchOps[0].Path != "/spec/containers/0/resources" {
		t.Fatalf("not corrected patch %s", patchOps[0].String())
	}
}

func TestGetDefaultResources(t *testing.T) { //nolint:funlen
	t.Parallel()

	patch := resources.Patch{}

	type testType struct {
		PodAnnotations,
		NamespaceAnnotations map[string]string
		ExpectedCPU,
		ExpectedMemory string
	}

	tests := []testType{
		{
			ExpectedCPU:    "100m",
			ExpectedMemory: "500Mi",
		},
		{
			PodAnnotations: map[string]string{
				types.AnnotationDefaultResourcesCPU: "200m",
			},
			ExpectedCPU:    "200m",
			ExpectedMemory: "500Mi",
		},
		{
			PodAnnotations: map[string]string{
				types.AnnotationDefaultResourcesMemory: "200Mi",
			},
			ExpectedCPU:    "100m",
			ExpectedMemory: "200Mi",
		},
		{
			PodAnnotations: map[string]string{
				types.AnnotationDefaultResourcesMemory: "200Mi",
			},
			NamespaceAnnotations: map[string]string{
				types.AnnotationDefaultResourcesMemory: "300Mi",
			},
			ExpectedCPU:    "100m",
			ExpectedMemory: "200Mi",
		},
		{
			PodAnnotations: map[string]string{
				types.AnnotationDefaultResourcesCPU: "200m",
			},
			NamespaceAnnotations: map[string]string{
				types.AnnotationDefaultResourcesMemory: "300Mi",
			},
			ExpectedCPU:    "200m",
			ExpectedMemory: "300Mi",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(fmt.Sprintf("%+v", test), func(t *testing.T) {
			t.Parallel()

			containerInfo := &types.ContainerInfo{
				PodAnnotations:       test.PodAnnotations,
				NamespaceAnnotations: test.NamespaceAnnotations,
			}

			gotCPU, gotMemory := patch.GetDefaultResources(containerInfo)

			if gotCPU.String() != test.ExpectedCPU {
				t.Fatalf("not corrected CPU %s", gotCPU.String())
			}

			if gotMemory.String() != test.ExpectedMemory {
				t.Fatalf("not corrected Memory %s", gotMemory.String())
			}
		})
	}
}
