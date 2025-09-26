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
package pullsecrets_test

import (
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/pullsecrets"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

func TestPullSecrets(t *testing.T) {
	t.Parallel()

	containerInfo := &types.ContainerInfo{
		SelectedRules: []*types.Rule{
			{
				ImagePullSecrets: []corev1.LocalObjectReference{
					{
						Name: "test",
					},
				},
			},
		},
	}

	patch := pullsecrets.Patch{}

	patchOps, err := patch.Create(t.Context(), containerInfo)
	if err != nil {
		t.Fatal(err)
	}

	if len(patchOps) != 1 {
		t.Fatal("1 patch must be created")
	}

	secrets, ok := patchOps[0].Value.([]corev1.LocalObjectReference)
	if !ok {
		t.Fatal("patchOps[0].Value must be []corev1.LocalObjectReference")
	}

	if len(secrets) != 1 {
		t.Fatal("1 secret must be added")
	}

	if patchOps[0].Op != "add" || patchOps[0].Path != "/spec/imagePullSecrets" || secrets[0].Name != "test" {
		t.Fatalf("not corrected patch %s", patchOps[0].String())
	}
}

func TestPullSecretsExists(t *testing.T) {
	t.Parallel()

	containerInfo := &types.ContainerInfo{
		PodContainer: &types.PodContainer{
			Pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "first",
						},
					},
				},
			},
		},
		SelectedRules: []*types.Rule{
			{
				ImagePullSecrets: []corev1.LocalObjectReference{
					{
						Name: "test",
					},
				},
			},
		},
	}

	patch := pullsecrets.Patch{}

	patchOps, err := patch.Create(t.Context(), containerInfo)
	if err != nil {
		t.Fatal(err)
	}

	if len(patchOps) != 1 {
		t.Fatal("1 patch must be created")
	}

	secrets, ok := patchOps[0].Value.([]corev1.LocalObjectReference)
	if !ok {
		t.Fatal("patchOps[0].Value must be []corev1.LocalObjectReference")
	}

	if len(secrets) != 2 {
		t.Fatal("2 secret must be added")
	}

	if patchOps[0].Op != "add" || patchOps[0].Path != "/spec/imagePullSecrets" {
		t.Fatalf("not corrected patch %s", patchOps[0].String())
	}

	if secrets[0].Name != "first" || secrets[1].Name != "test" {
		t.Fatalf("not corrected patch %s", patchOps[0].String())
	}
}
