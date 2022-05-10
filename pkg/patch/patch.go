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
package patch

import (
	"fmt"
	"strings"

	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// add env to container.
func CreateEnvPatch(containerOrder int, containerInfo types.ContainerInfo, containerEnv []corev1.EnvVar, newEnv []corev1.EnvVar) []types.PatchOperation { //nolint:lll
	// some containers don't need env
	// pod-admission-controller/ignoreEnv=container1,container2
	if ignore, ok := containerInfo.GetPodAnnotation(types.AnnotationIgnoreEnv); ok {
		containersNames := strings.Split(ignore, ",")
		for _, containerName := range containersNames {
			if containerName == containerInfo.ContainerName {
				return nil
			}
		}
	}

	patch := make([]types.PatchOperation, 0)

	if len(containerEnv) == 0 {
		patch = append(patch, types.PatchOperation{
			Op:    "add",
			Path:  fmt.Sprintf("/spec/containers/%d/env", containerOrder),
			Value: newEnv,
		})
	} else {
		containerEnvName := make(map[string]bool, 0)
		// get all env from container
		for _, env := range containerEnv {
			containerEnvName[env.Name] = true
		}

		for _, env := range newEnv {
			// add env if not exists
			if _, ok := containerEnvName[env.Name]; !ok {
				patch = append(patch, types.PatchOperation{
					Op:    "add",
					Path:  fmt.Sprintf("/spec/containers/%d/env/-", containerOrder),
					Value: env,
				})
			}
		}
	}

	return patch
}

// add default resources to container if not exists.
func CreateDefaultResourcesPatch(selectedRule types.Rule, containerOrder int, containerInfo types.ContainerInfo, containerResources corev1.ResourceRequirements) []types.PatchOperation { //nolint:lll
	// some containers don't need default resources
	// pod-admission-controller/ignoreAddDefaultResources=container1,container2
	if ignore, ok := containerInfo.GetPodAnnotation(types.AnnotationIgnoreAddDefaultResources); ok {
		containersNames := strings.Split(ignore, ",")
		for _, containerName := range containersNames {
			if containerName == containerInfo.ContainerName {
				return nil
			}
		}
	}

	patch := make([]types.PatchOperation, 0)

	newResources := corev1.ResourceRequirements{}

	newResources.Requests = corev1.ResourceList{}
	newResources.Limits = corev1.ResourceList{}

	if containerResources.Requests.Cpu().IsZero() {
		newResources.Requests["cpu"] = resource.MustParse(*config.Get().DefaultRequestCPU)
	} else {
		newResources.Requests["cpu"] = *containerResources.Requests.Cpu()
	}

	if containerResources.Requests.Memory().IsZero() {
		newResources.Requests["memory"] = resource.MustParse(*config.Get().DefaultRequestMemory)
	} else {
		newResources.Requests["memory"] = *containerResources.Requests.Memory()
	}

	// add resource limits if exists
	if containerResources.Limits.Cpu().IsZero() {
		if selectedRule.AddDefaultResources.LimitCPU {
			newResources.Limits["cpu"] = newResources.Requests["cpu"]
		}
	} else {
		newResources.Limits["cpu"] = *containerResources.Limits.Cpu()
	}

	// is no memory limits set, set memory resources
	if containerResources.Limits.Memory().IsZero() {
		newResources.Limits["memory"] = newResources.Requests["memory"]
	} else {
		newResources.Limits["memory"] = *containerResources.Limits.Memory()
	}

	patch = append(patch, types.PatchOperation{
		Op:    "add",
		Path:  fmt.Sprintf("/spec/containers/%d/resources", containerOrder),
		Value: newResources,
	})

	return patch
}

// add RunAsNonRoot policy to all containers (exlude InitContainers).
func CreateRunAsNonRootPatch(selectedRule types.Rule, order int, containerInfo types.ContainerInfo, podSecurityContext *corev1.PodSecurityContext, containerSecurityContext *corev1.SecurityContext) []types.PatchOperation { //nolint:lll,cyclop
	// some containers don't need security context check
	// pod-admission-controller/ignoreRunAsNonRoot=container1,container2
	if ignore, ok := containerInfo.GetPodAnnotation(types.AnnotationIgnoreRunAsNonRoot); ok {
		containersNames := strings.Split(ignore, ",")
		for _, containerName := range containersNames {
			if containerName == containerInfo.ContainerName {
				return nil
			}
		}
	}

	patch := make([]types.PatchOperation, 0)

	var containerRunAsUser *int64

	boolTrue := true
	boolFalse := false

	if podSecurityContext != nil && podSecurityContext.RunAsUser != nil {
		containerRunAsUser = podSecurityContext.RunAsUser
	}

	securityContext := containerSecurityContext

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
		Path:  fmt.Sprintf("/spec/containers/%d/securityContext", order),
		Value: securityContext,
	})

	return patch
}
