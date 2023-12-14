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
package tolerations

import (
	"context"

	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

type Patch struct{}

func (p *Patch) Create(_ context.Context, containerInfo *types.ContainerInfo) ([]types.PatchOperation, error) {
	podTolerations := make([]corev1.Toleration, 0)

	for _, rule := range containerInfo.SelectedRules {
		if len(rule.Tolerations) > 0 {
			podTolerations = append(podTolerations, rule.Tolerations...)
		}
	}

	if len(podTolerations) == 0 {
		return []types.PatchOperation{}, nil
	}

	return []types.PatchOperation{
		{
			Op:    "add",
			Path:  "/spec/tolerations",
			Value: podTolerations,
		},
	}, nil
}
