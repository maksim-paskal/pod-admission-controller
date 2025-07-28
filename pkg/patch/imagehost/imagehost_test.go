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
package imagehost_test

import (
	"context"
	"os"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/api"
	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/imagehost"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

func TestReplaceImageHostPatch(t *testing.T) {
	t.Parallel()

	patch := imagehost.Patch{}

	const requiredValue = "fromenv"

	os.Setenv("TEST_ENV", requiredValue) //nolint:tenv,usetesting

	containerInfo := &types.ContainerInfo{
		PodContainer: &types.PodContainer{
			Type:      "container",
			Container: &corev1.Container{},
		},
		Image: &types.ContainerImage{
			Name: "test1",
		},
		SelectedRules: []*types.Rule{
			{
				ReplaceContainerImageHost: types.ReplaceContainerImageHost{
					Enabled: true,
					From:    "test1",
					To:      `{{ env "TEST_ENV" }}`,
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

	if patchOps[0].Value != requiredValue {
		t.Fatalf("not corrected value %s/%s", patchOps[0].Value, requiredValue)
	}
}

func TestReplaceImageHostPatchDockerIo(t *testing.T) { //nolint:funlen
	t.Parallel()

	patch := imagehost.Patch{}

	type Test struct {
		Image    string
		Required string
	}

	tests := []Test{
		{
			Image:    "alpine:3.12",
			Required: "my.docker.io/library/alpine:3.12",
		},
		{
			Image:    "alpine",
			Required: "my.docker.io/library/alpine",
		},
		{
			Image:    "alpine:3.12@sha256:8fd21d59428507671ce0fb47f818b1d859c92d2ad07bb7c947268d433030ba98",
			Required: "my.docker.io/library/alpine:3.12@sha256:8fd21d59428507671ce0fb47f818b1d859c92d2ad07bb7c947268d433030ba98",
		},
		{
			Image:    "alpine@sha256:8fd21d59428507671ce0fb47f818b1d859c92d2ad07bb7c947268d433030ba98",
			Required: "my.docker.io/library/alpine@sha256:8fd21d59428507671ce0fb47f818b1d859c92d2ad07bb7c947268d433030ba98",
		},
		{
			Image:    "test/alpine",
			Required: "my.docker.io/test/alpine",
		},
		{
			Image:    "test/alpine:3.12",
			Required: "my.docker.io/test/alpine:3.12",
		},
	}

	for _, test := range tests {
		imageInfo, err := api.GetImageInfo(test.Image)
		if err != nil {
			t.Fatal(err)
		}

		containerInfo := &types.ContainerInfo{
			PodContainer: &types.PodContainer{
				Type:      "container",
				Container: &corev1.Container{},
			},
			Image: imageInfo,
			SelectedRules: []*types.Rule{
				{
					ReplaceContainerImageHost: types.ReplaceContainerImageHost{
						Enabled: true,
						From:    "docker.io",
						To:      `my.docker.io`,
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

		if patchOps[0].Value != test.Required {
			t.Fatalf("not corrected value got=%s, required=%s", patchOps[0].Value, test.Required)
		}
	}
}

func TestReplaceImageHostPatchNotDockerIo(t *testing.T) {
	t.Parallel()

	patch := imagehost.Patch{}

	containerInfo := &types.ContainerInfo{
		PodContainer: &types.PodContainer{
			Type:      "container",
			Container: &corev1.Container{},
		},
		Image: &types.ContainerImage{
			Name: "some.docker.io/test1/test2",
		},
		SelectedRules: []*types.Rule{
			{
				ReplaceContainerImageHost: types.ReplaceContainerImageHost{
					Enabled: true,
					From:    "some.docker.io",
					To:      `my.docker.io`,
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

	requiredValue := "my.docker.io/test1/test2"

	if patchOps[0].Value != requiredValue {
		t.Fatalf("not corrected value got=%s, required=%s", patchOps[0].Value, requiredValue)
	}
}

func TestReplaceImageTag(t *testing.T) {
	t.Parallel()

	patch := imagehost.Patch{}

	containerInfo := &types.ContainerInfo{
		PodContainer: &types.PodContainer{
			Type:      "container",
			Container: &corev1.Container{},
		},
		Image: &types.ContainerImage{
			Name: "some.docker.io/test1/test2:release-1234",
		},
		SelectedRules: []*types.Rule{
			{
				ReplaceContainerImageHost: types.ReplaceContainerImageHost{
					Enabled: true,
					From:    "test1/test2:release-1234",
					To:      "test1/test2:release-9876",
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

	requiredValue := "some.docker.io/test1/test2:release-9876"

	if patchOps[0].Value != requiredValue {
		t.Fatalf("not corrected value got=%s, required=%s", patchOps[0].Value, requiredValue)
	}
}
