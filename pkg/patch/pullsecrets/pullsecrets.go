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
package pullsecrets

import (
	"context"

	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

type Patch struct{}

// append pull secrets to pod.
func (p *Patch) Create(_ context.Context, containerInfo *types.ContainerInfo) ([]types.PatchOperation, error) {
	podPullSecrets := []corev1.LocalObjectReference{}

	if containerInfo.PodContainer != nil && containerInfo.PodContainer.Pod != nil {
		podPullSecrets = containerInfo.PodContainer.Pod.Spec.ImagePullSecrets
	}

	for _, rule := range containerInfo.SelectedRules {
		if len(rule.ImagePullSecrets) > 0 {
			podPullSecrets = append(podPullSecrets, rule.ImagePullSecrets...)
		}
	}

	if len(podPullSecrets) == 0 {
		return []types.PatchOperation{}, nil
	}

	return []types.PatchOperation{
		{
			Op:    "add",
			Path:  "/spec/imagePullSecrets",
			Value: podPullSecrets,
		},
	}, nil
}
