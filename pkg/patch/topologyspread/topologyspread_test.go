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
package topologyspread_test

import (
	"context"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/topologyspread"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

func TestLifeCyclePatch(t *testing.T) {
	t.Parallel()

	patch := topologyspread.Patch{}

	containerInfo := &types.ContainerInfo{
		PodLabels: map[string]string{
			"app":               "test",
			"env":               "dev",
			"pod-template-hash": "123",
		},
		PodContainer: &types.PodContainer{
			Type:      "container",
			Container: &corev1.Container{},
			Pod:       &corev1.Pod{},
		},
		SelectedRules: []*types.Rule{
			{
				AddTopologySpread: types.AddTopologySpread{
					Enabled: true,
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
						{
							MaxSkew: 1,
						},
					},
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

	if patchOps[0].Op != "add" || patchOps[0].Path != "/spec/topologySpreadConstraints" {
		t.Fatalf("not corrected patch %s", patchOps[0].String())
	}
}
