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
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/patch"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const addOperation = "add"

func TestCreateEnvPatchEnv(t *testing.T) {
	t.Parallel()

	containerInfo := types.ContainerInfo{}

	containerEnv := []corev1.EnvVar{
		{
			Name:  "TEST1",
			Value: "1",
		},
		{
			Name:  "TEST2",
			Value: "2",
		},
	}

	newEnv := []corev1.EnvVar{
		{
			Name:  "TEST1",
			Value: "3",
		},
		{
			Name:  "TEST4",
			Value: "4",
		},
	}

	envPatch := patch.CreateEnvPatch(0, containerInfo, containerEnv, newEnv)

	if len(envPatch) != 1 {
		t.Fatal("1 patch must be created")
	}

	if envPatch[0].Op != addOperation || envPatch[0].Path != "/spec/containers/0/env/-" {
		t.Fatal("not corrected patch")
	}

	// scenario 2 (container env is empty)
	containerEnv = []corev1.EnvVar{}

	envPatch = patch.CreateEnvPatch(0, containerInfo, containerEnv, newEnv)

	if len(envPatch) != 1 {
		t.Fatal("1 patch must be created")
	}

	if envPatch[0].Op != addOperation || envPatch[0].Path != "/spec/containers/0/env" {
		t.Fatal("not corrected patch")
	}
}

func TestNullResources(t *testing.T) {
	t.Parallel()

	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("1"),
		},
	}

	rule := types.Rule{
		AddDefaultResources: types.AddDefaultResources{
			Enabled:         true,
			RemoveResources: true,
		},
	}
	patch := patch.CreateDefaultResourcesPatch(&rule, 0, types.ContainerInfo{}, resources)

	if len(patch) != 1 {
		t.Fatal("1 patch must be created")
	}

	if patch[0].Op != "remove" || patch[0].Path != "/spec/containers/0/resources" {
		t.Fatalf("not corrected op %s", patch[0].Op)
	}
}

func TestCreateDefaultResourcesPatch(t *testing.T) {
	t.Parallel()

	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("1"),
		},
	}

	rule := types.Rule{
		AddDefaultResources: types.AddDefaultResources{
			Enabled: true,
		},
	}
	patch := patch.CreateDefaultResourcesPatch(&rule, 0, types.ContainerInfo{}, resources)

	if len(patch) != 1 {
		t.Fatal("1 patch must be created")
	}

	if patch[0].Op != addOperation || patch[0].Path != "/spec/containers/0/resources" {
		t.Fatal("not corrected patch")
	}
}

func TestCreateRunAsNonRootPatch(t *testing.T) {
	t.Parallel()

	rule := types.Rule{
		RunAsNonRoot: types.RunAsNonRoot{
			Enabled: true,
		},
	}

	podSecurityContext := corev1.PodSecurityContext{}
	securityContext := corev1.SecurityContext{}

	patch := patch.CreateRunAsNonRootPatch(&rule, 0, types.ContainerInfo{}, &podSecurityContext, &securityContext)

	if len(patch) != 1 {
		t.Fatal("1 patch must be created")
	}

	if patch[0].Op != addOperation || patch[0].Path != "/spec/containers/0/securityContext" {
		t.Fatalf("not corrected patch %s/%s", patch[0].Op, patch[0].Path)
	}
}
