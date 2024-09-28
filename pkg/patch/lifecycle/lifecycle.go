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
package lifecycle

import (
	"context"

	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
)

type Patch struct{}

func (p *Patch) Create(_ context.Context, containerInfo *types.ContainerInfo) ([]types.PatchOperation, error) {
	patch := make([]types.PatchOperation, 0)

	for _, selectedRule := range containerInfo.SelectedRules {
		if !selectedRule.AddLifecycle.Enabled {
			continue
		}

		patch = append(patch, types.PatchOperation{
			Op:    "add",
			Path:  containerInfo.PodContainer.ContainerPath() + "/lifecycle",
			Value: selectedRule.AddLifecycle.Lifecycle,
		})

		// only one lifecycle can be added
		break
	}

	return patch, nil
}
