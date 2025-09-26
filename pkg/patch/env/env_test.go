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
package env_test

import (
	"fmt"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/env"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

const addOperation = "add"

func TestCreateEnvPatchEnv(t *testing.T) { //nolint:funlen
	t.Parallel()

	patch := env.Patch{}

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

	containerInfo := &types.ContainerInfo{
		PodContainer: &types.PodContainer{
			Type: "container",
			Container: &corev1.Container{
				Env: containerEnv,
			},
		},
		SelectedRules: []*types.Rule{
			{
				Env: []corev1.EnvVar{
					{
						Name:  "TEST1",
						Value: "3",
					},
					{
						Name:  "TEST4",
						Value: "4",
					},
				},
			},
		},
	}

	envPatch, err := patch.Create(t.Context(), containerInfo)
	if err != nil {
		t.Fatal(err)
	}

	if len(envPatch) != 1 {
		t.Fatal("1 patch must be created")
	}

	if envPatch[0].Op != addOperation || envPatch[0].Path != "/spec/containers/0/env/-" {
		t.Fatalf("not corrected patch %s", envPatch[0].String())
	}

	// scenario 2 (container env is empty)
	containerInfo.PodContainer = &types.PodContainer{
		Type: "container",
		Container: &corev1.Container{
			Env: []corev1.EnvVar{},
		},
	}

	envPatch, err = patch.Create(t.Context(), containerInfo)
	if err != nil {
		t.Fatal(err)
	}

	if len(envPatch) != 1 {
		t.Fatal("1 patch must be created")
	}

	if envPatch[0].Op != addOperation || envPatch[0].Path != "/spec/containers/0/env" {
		t.Fatalf("not corrected patch %s", envPatch[0].String())
	}
}

func TestFormatEnv(t *testing.T) {
	t.Parallel()

	patch := env.Patch{}

	containerEnv := []corev1.EnvVar{
		{
			Name:  "TEST1",
			Value: `{{ index (regexp "/(.+):(.+)$" .Image.Name) 1 }}`,
		},
		{
			Name:  "TEST2",
			Value: "test",
		},
		{
			Name:  "TEST3",
			Value: "{{ .NamespaceLabels.app }}",
		},
		{
			Name:  "TEST4",
			Value: "{{ .NamespaceLabels.unknown }}",
		},
	}

	containerInfo := &types.ContainerInfo{
		Image:           &types.ContainerImage{Name: "/1/2/3:4"},
		NamespaceLabels: map[string]string{"app": "testapp"},
	}

	result, err := patch.FormatEnv(containerInfo, containerEnv)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != len(containerEnv) {
		t.Fatal("length must be equal")
	}

	returnResults := []string{
		"TEST1:1/2/3",
		"TEST2:test",
		"TEST3:testapp",
		"TEST4:",
	}

	for i, returnResult := range returnResults {
		v := fmt.Sprintf("%s:%s", result[i].Name, result[i].Value)

		if v != returnResult {
			t.Fatalf("must be %s, got %s", returnResult, v)
		}
	}
}
