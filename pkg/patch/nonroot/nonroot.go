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
package nonroot

import (
	"context"
	"strings"

	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	corev1 "k8s.io/api/core/v1"
)

type Patch struct{}

func (p *Patch) Create(_ context.Context, containerInfo *types.ContainerInfo) ([]types.PatchOperation, error) { //nolint:lll,funlen,cyclop
	// some containers don't need security context check
	// pod-admission-controller/ignoreRunAsNonRoot=container1,container2
	if ignore, ok := containerInfo.GetPodAnnotation(types.AnnotationIgnoreRunAsNonRoot); ok { //nolint:staticcheck
		containersNames := strings.Split(ignore, ",")
		for _, containerName := range containersNames {
			if containerName == containerInfo.ContainerName {
				return []types.PatchOperation{}, nil
			}
		}
	}

	patch := make([]types.PatchOperation, 0)

	for _, selectedRule := range containerInfo.SelectedRules {
		if !selectedRule.RunAsNonRoot.Enabled {
			continue
		}

		selectedRule.Logf("CreateRunAsNonRootPatch: %+v", selectedRule)

		var containerRunAsUser *int64

		boolTrue := true
		boolFalse := false

		podContainer := containerInfo.PodContainer

		if podContainer.Pod.Spec.SecurityContext != nil && podContainer.Pod.Spec.SecurityContext.RunAsUser != nil {
			containerRunAsUser = podContainer.Pod.Spec.SecurityContext.RunAsUser
		}

		securityContext := podContainer.Container.SecurityContext

		if securityContext == nil {
			securityContext = &corev1.SecurityContext{}
		}

		if securityContext.RunAsUser != nil {
			containerRunAsUser = securityContext.RunAsUser
		}

		if selectedRule.RunAsNonRoot.ReplaceUser.Enabled && containerRunAsUser != nil {
			if *containerRunAsUser == selectedRule.RunAsNonRoot.ReplaceUser.FromUser {
				containerRunAsUser = &selectedRule.RunAsNonRoot.ReplaceUser.ToUser
			}
		}

		if containerRunAsUser != nil {
			securityContext.RunAsUser = containerRunAsUser
		}

		securityContext.RunAsNonRoot = &boolTrue
		securityContext.Privileged = &boolFalse
		securityContext.AllowPrivilegeEscalation = &boolFalse

		if securityContext.Capabilities == nil {
			securityContext.Capabilities = &corev1.Capabilities{}
		}

		securityContext.Capabilities.Drop = []corev1.Capability{corev1.Capability("ALL")}

		patch = append(patch, types.PatchOperation{
			Op:    "add",
			Path:  containerInfo.PodContainer.ContainerPath() + "/securityContext",
			Value: securityContext,
		})

		// process only first rule
		break
	}

	return patch, nil
}
