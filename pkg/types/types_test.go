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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPodAnnotation(t *testing.T) {
	t.Parallel()

	const annotationEnv = "pod-admission-controller/env"

	containerInfo := types.ContainerInfo{
		PodAnnotations: map[string]string{
			annotationEnv: "test",
			"test444":     "test444Value",
		},
		NamespaceAnnotations: map[string]string{
			annotationEnv:       "test1",
			annotationEnv + "2": "test2",
			"adsasdasd":         "1234567",
		},
	}

	testsOk := make(map[string]string)

	testsOk[annotationEnv] = "test"
	testsOk[annotationEnv+"2"] = "test2"
	testsOk["adsasdasd"] = "1234567"
	testsOk["test444"] = "test444Value"

	for annotation, expected := range testsOk {
		got, ok := containerInfo.GetPodAnnotation(annotation)
		if !ok {
			t.Fatal("expected to find pod annotation")
		}

		if got != expected {
			t.Fatalf("expected %s, got %s", expected, got)
		}
	}

	if _, ok := containerInfo.GetPodAnnotation("someother"); ok {
		t.Fatal("annotation should not be found")
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

func TestContainerPath(t *testing.T) {
	t.Parallel()

	podContainer := types.PodContainer{
		Type:  "container",
		Order: 0,
	}

	if got := podContainer.ContainerPath(); got != "/spec/containers/0" {
		t.Fatalf("expected /spec/containers/0, got %s", got)
	}

	podContainer = types.PodContainer{
		Type:  "initContainer",
		Order: 0,
	}

	if got := podContainer.ContainerPath(); got != "/spec/initContainers/0" {
		t.Fatalf("expected /spec/initContainers/0, got %s", got)
	}
}

func TestPodContainer(t *testing.T) {
	t.Parallel()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Pod",
				},
			},
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name: "test-init",
				},
			},
			Containers: []corev1.Container{
				{
					Name: "test",
				},
			},
		},
	}

	podContainers := types.PodContainersFromPod(namespace, pod)

	if req := 2; len(podContainers) != req {
		t.Fatalf("expected to find %d, got %d", req, len(podContainers))
	}

	if req := "Pod"; podContainers[0].OwnerKind() != req {
		t.Fatalf("expected to find %s, got %s", req, podContainers[0].OwnerKind())
	}
}
