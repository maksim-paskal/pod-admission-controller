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
package types_test

import (
	"fmt"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

func TestGetPodAnnotation(t *testing.T) {
	t.Parallel()

	const annotationEnv = "pod-admission-controller/env"

	containerInfo := types.ContainerInfo{
		PodAnnotations: map[string]string{
			annotationEnv: "test",
		},
	}

	env, ok := containerInfo.GetPodAnnotation(annotationEnv)

	if !ok {
		t.Fatal("expected to find pod annotation")
	}

	if env != "test" {
		t.Fatal("expected to find pod annotation value")
	}

	if _, ok := containerInfo.GetPodAnnotation("someother"); ok {
		t.Fatal("annotation should not be found")
	}
}

func TestGetSelectedRuleEnabled(t *testing.T) { //nolint:funlen
	t.Parallel()

	containerInfo := types.ContainerInfo{
		SelectedRules: []*types.Rule{
			{
				Name: "test1",
			},
			{
				Name: "test2",
				RunAsNonRoot: types.RunAsNonRoot{
					Enabled: true,
				},
			},
			{
				Name: "test3",
				AddDefaultResources: types.AddDefaultResources{
					Enabled: true,
				},
			},
			{
				Name: "test4",
				RunAsNonRoot: types.RunAsNonRoot{
					Enabled: true,
				},
			},
		},
	}

	selectedRule, ok := containerInfo.GetSelectedRuleEnabled(types.SelectedRuleRunAsNonRoot)

	if !ok {
		t.Fatal("expected to find selected rule")
	}

	if selectedRule.Name != "test2" {
		t.Fatalf("expected to test2, got %s", selectedRule.Name)
	}

	selectedRule, ok = containerInfo.GetSelectedRuleEnabled(types.SelectedRuleAddDefaultResources)

	if !ok {
		t.Fatal("expected to find selected rule")
	}

	if selectedRule.Name != "test3" {
		t.Fatalf("expected to test3, got %s", selectedRule.Name)
	}

	// invalid rule
	_, ok = containerInfo.GetSelectedRuleEnabled("fake")
	if ok {
		t.Fatal("expected not to find selected rule")
	}

	containerInfo = types.ContainerInfo{
		SelectedRules: []*types.Rule{
			{
				Name: "test1",
			},
			{
				Name: "test2",
			},
			{
				Name: "test3",
			},
		},
	}

	if _, ok = containerInfo.GetSelectedRuleEnabled(types.SelectedRuleRunAsNonRoot); ok {
		t.Fatal("expected not to find selected rule")
	}

	if _, ok = containerInfo.GetSelectedRuleEnabled(types.SelectedRuleAddDefaultResources); ok {
		t.Fatal("expected not to find selected rule")
	}
}

func TestGetSelectedRulesEnv(t *testing.T) {
	t.Parallel()

	containerInfo := types.ContainerInfo{
		SelectedRules: []*types.Rule{
			{
				Env: []corev1.EnvVar{
					{
						Name: "test1",
					},
				},
			},
			{
				Env: []corev1.EnvVar{
					{
						Name: "test2",
					},
				},
			},
			{
				Env: []corev1.EnvVar{
					{
						Name: "test3",
					},
				},
			},
		},
	}

	allEnv := containerInfo.GetSelectedRulesEnv()

	if len(allEnv) != 3 {
		t.Fatal("expected to find 3 env")
	}

	for i, env := range allEnv {
		if env.Name != fmt.Sprintf("test%d", i+1) {
			t.Fatalf("expected to find test%d, got %s", i+1, env.Name)
		}
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	containerInfo := types.ContainerInfo{}

	if got := containerInfo.String(); got == "" {
		t.Fatal("expected to find json")
	}
}
