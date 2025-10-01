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
package topologyspread

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/maksim-paskal/pod-admission-controller/pkg/template"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var podLabelsIgnore = []string{
	"pod-template-hash",
	"controller-revision-hash",
	"statefulset.kubernetes.io/pod-name",
	"apps.kubernetes.io/pod-index",
}

type Patch struct{}

func (p *Patch) Create(_ context.Context, containerInfo *types.ContainerInfo) ([]types.PatchOperation, error) {
	patch := make([]types.PatchOperation, 0)

	podLabels := map[string]string{}

	for key, value := range containerInfo.PodLabels {
		if slices.Contains(podLabelsIgnore, key) {
			continue
		}

		podLabels[key] = value
	}

	for _, selectedRule := range containerInfo.SelectedRules {
		if !selectedRule.AddTopologySpread.Enabled {
			continue
		}

		topologySpreadConstraints := selectedRule.AddTopologySpread.Clone().TopologySpreadConstraints

		topologySpreadConstraintsJSON, err := json.Marshal(topologySpreadConstraints)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal topologySpreadConstraints")
		}

		topologySpreadConstraintsFormatted, err := template.Get(containerInfo, string(topologySpreadConstraintsJSON))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get formatted topologySpreadConstraints")
		}

		if err := json.Unmarshal([]byte(topologySpreadConstraintsFormatted), &topologySpreadConstraints); err != nil {
			return nil, errors.Wrap(err, "error unmarshal topologySpreadConstraints")
		}

		for i := range topologySpreadConstraints {
			topologySpreadConstraints[i].LabelSelector = &metav1.LabelSelector{
				MatchLabels: podLabels,
			}
		}

		patch = append(patch, types.PatchOperation{
			Op:    "add",
			Path:  "/spec/topologySpreadConstraints",
			Value: topologySpreadConstraints,
		})

		// only one rule can be applied
		break
	}

	return patch, nil
}
