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
package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	defaultContainerResourceCPU    = "100m"
	defaultContainerResourceMemory = "500Mi"
)

type Patch struct{}

// get default resources from pod annotations
// pod-admission-controller/defaultResourcesCPU=100m
// pod-admission-controller/defaultResourcesMemory=500Mi.
func (p *Patch) GetDefaultResources(containerInfo *types.ContainerInfo) (resource.Quantity, resource.Quantity) {
	defaultRequestCPU := resource.MustParse(defaultContainerResourceCPU)
	defaultRequestMemory := resource.MustParse(defaultContainerResourceMemory)

	if defaultResource, ok := containerInfo.GetPodAnnotation(types.AnnotationDefaultResourcesCPU); ok {
		resourceCPU, err := resource.ParseQuantity(defaultResource)
		if err != nil {
			log.WithError(err).Errorf("ParseQuantity: %+v", defaultResource)
		} else {
			defaultRequestCPU = resourceCPU
		}
	}

	if defaultResource, ok := containerInfo.GetPodAnnotation(types.AnnotationDefaultResourcesMemory); ok {
		resourceMemory, err := resource.ParseQuantity(defaultResource)
		if err != nil {
			log.WithError(err).Errorf("ParseQuantity: %+v", defaultResource)
		} else {
			defaultRequestMemory = resourceMemory
		}
	}

	return defaultRequestCPU, defaultRequestMemory
}

func (p *Patch) Create(_ context.Context, containerInfo *types.ContainerInfo) ([]types.PatchOperation, error) { //nolint:lll,funlen,cyclop
	// some containers don't need default resources
	// pod-admission-controller/ignoreAddDefaultResources=container1,container2
	if ignore, ok := containerInfo.GetPodAnnotation(types.AnnotationIgnoreAddDefaultResources); ok { //nolint:staticcheck
		containersNames := strings.Split(ignore, ",")
		for _, containerName := range containersNames {
			if containerName == containerInfo.ContainerName {
				return []types.PatchOperation{}, nil
			}
		}
	}

	patch := make([]types.PatchOperation, 0)

	for _, selectedRule := range containerInfo.SelectedRules {
		if !selectedRule.AddDefaultResources.Enabled {
			continue
		}

		// remove pod resources from all containers
		if selectedRule.AddDefaultResources.RemoveResources { //nolint:staticcheck
			return []types.PatchOperation{{
				Op:   "remove",
				Path: fmt.Sprintf("%s/resources", containerInfo.PodContainer.ContainerPath()),
			}}, nil
		}

		selectedRule.Logf("CreateDefaultResourcesPatch: %+v", selectedRule)

		newResources := corev1.ResourceRequirements{}

		newResources.Requests = corev1.ResourceList{}
		newResources.Limits = corev1.ResourceList{}

		defaultRequestCPU, defaultRequestMemory := p.GetDefaultResources(containerInfo)

		if containerInfo.PodContainer.Container.Resources.Requests.Cpu().IsZero() {
			newResources.Requests["cpu"] = defaultRequestCPU
		} else {
			newResources.Requests["cpu"] = *containerInfo.PodContainer.Container.Resources.Requests.Cpu()
		}

		if containerInfo.PodContainer.Container.Resources.Requests.Memory().IsZero() {
			newResources.Requests["memory"] = defaultRequestMemory
		} else {
			newResources.Requests["memory"] = *containerInfo.PodContainer.Container.Resources.Requests.Memory()
		}

		// add resource limits if exists
		if containerInfo.PodContainer.Container.Resources.Limits.Cpu().IsZero() {
			if selectedRule.AddDefaultResources.LimitCPU {
				newResources.Limits["cpu"] = newResources.Requests["cpu"]
			}
		} else {
			newResources.Limits["cpu"] = *containerInfo.PodContainer.Container.Resources.Limits.Cpu()
		}

		// is no memory limits set, set memory resources
		if containerInfo.PodContainer.Container.Resources.Limits.Memory().IsZero() {
			newResources.Limits["memory"] = newResources.Requests["memory"]
		} else {
			newResources.Limits["memory"] = *containerInfo.PodContainer.Container.Resources.Limits.Memory()
		}

		patch = append(patch, types.PatchOperation{
			Op:    "add",
			Path:  fmt.Sprintf("%s/resources", containerInfo.PodContainer.ContainerPath()),
			Value: newResources,
		})

		// process only first rule
		break
	}

	return patch, nil
}
