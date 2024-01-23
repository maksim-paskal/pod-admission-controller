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
package custompatch

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/maksim-paskal/pod-admission-controller/pkg/template"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/pkg/errors"
)

type Patch struct{}

func (p *Patch) Create(_ context.Context, containerInfo *types.ContainerInfo) ([]types.PatchOperation, error) { //nolint:lll
	patch := make([]types.PatchOperation, 0)

	for _, selectedRule := range containerInfo.SelectedRules {
		for _, customPatch := range selectedRule.CustomPatches {
			newPatchBytes, err := json.Marshal(customPatch)
			if err != nil {
				return nil, errors.Wrap(err, "error marshal newPatch")
			}

			newPatchJSON, err := template.Get(containerInfo, string(newPatchBytes))
			if err != nil {
				return nil, errors.Wrap(err, "error parsing template Op")
			}

			newPatch := types.PatchOperation{}

			if err := json.Unmarshal([]byte(newPatchJSON), &newPatch); err != nil {
				return nil, errors.Wrap(err, "error unmarshal newPatch")
			}

			if p.ignorePatch(newPatch, containerInfo) {
				continue
			}

			patch = append(patch, newPatch)
		}
	}

	return patch, nil
}

// check well known operations and ignore them.
func (p *Patch) ignorePatch(patch types.PatchOperation, containerInfo *types.ContainerInfo) bool { //nolint:cyclop
	if patch.Op != "remove" {
		return false
	}

	pod := containerInfo.PodContainer.Pod

	switch strings.ToLower(patch.Path) {
	case "/spec/affinity":
		if pod.Spec.Affinity == nil {
			return true
		}
	case "/spec/nodeselector":
		if pod.Spec.NodeSelector == nil || len(pod.Spec.NodeSelector) == 0 {
			return true
		}
	case strings.ToLower(containerInfo.PodContainer.ContainerPath() + "/readinessProbe"):
		if containerInfo.PodContainer.Container.ReadinessProbe == nil {
			return true
		}
	case strings.ToLower(containerInfo.PodContainer.ContainerPath() + "/livenessProbe"):
		if containerInfo.PodContainer.Container.LivenessProbe == nil {
			return true
		}
	}

	return false
}
