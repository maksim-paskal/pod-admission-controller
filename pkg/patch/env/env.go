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
package env

import (
	"context"
	"strings"

	"github.com/maksim-paskal/pod-admission-controller/pkg/template"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type Patch struct{}

type processContainerEnvInput struct {
	containerInfo *types.ContainerInfo
	formattedEnv  []corev1.EnvVar
	replace       bool
}

func (p *Patch) processContainerEnv(input *processContainerEnvInput) []types.PatchOperation {
	if len(input.formattedEnv) == 0 {
		return []types.PatchOperation{}
	}

	// if container has no env, return one patch to add all env
	if len(input.containerInfo.PodContainer.Container.Env) == 0 {
		return []types.PatchOperation{
			{
				Op:    "add",
				Path:  input.containerInfo.PodContainer.ContainerPath() + "/env",
				Value: input.formattedEnv,
			},
		}
	}

	patch := make([]types.PatchOperation, 0)

	containerEnvName := make(map[string]bool, 0)
	// get all env from container
	for _, env := range input.containerInfo.PodContainer.Container.Env {
		containerEnvName[env.Name] = true
	}

	for _, env := range input.formattedEnv {
		_, containerHasThisEnv := containerEnvName[env.Name]

		if input.replace {
			if containerHasThisEnv {
				patch = append(patch, types.PatchOperation{
					Op:    "replace",
					Path:  input.containerInfo.PodContainer.ContainerPath() + "/env/" + env.Name,
					Value: env.Value,
				})
			} else {
				patch = append(patch, types.PatchOperation{
					Op:    "add",
					Path:  input.containerInfo.PodContainer.ContainerPath() + "/env/-",
					Value: env,
				})
			}
		} else {
			if !containerHasThisEnv {
				patch = append(patch, types.PatchOperation{
					Op:    "add",
					Path:  input.containerInfo.PodContainer.ContainerPath() + "/env/-",
					Value: env,
				})
			}
		}
	}

	return patch
}

func (p *Patch) Create(_ context.Context, containerInfo *types.ContainerInfo) ([]types.PatchOperation, error) {
	// some containers don't need env
	// pod-admission-controller/ignoreEnv=container1,container2
	if ignore, ok := containerInfo.GetPodAnnotation(types.AnnotationIgnoreEnv); ok {
		containersNames := strings.Split(ignore, ",")
		for _, containerName := range containersNames {
			if containerName == containerInfo.ContainerName {
				return nil, nil
			}
		}
	}

	patch := make([]types.PatchOperation, 0)

	formattedEnv, err := p.FormatEnv(containerInfo, containerInfo.GetSelectedRulesEnv())
	if err != nil {
		return nil, errors.Wrap(err, "error format env")
	}

	patch = append(patch, p.processContainerEnv(&processContainerEnvInput{
		containerInfo: containerInfo,
		formattedEnv:  formattedEnv,
	})...)

	formattedEnv, err = p.FormatEnv(containerInfo, containerInfo.GetSelectedRulesEnvReplace())
	if err != nil {
		return nil, errors.Wrap(err, "error format env")
	}

	patch = append(patch, p.processContainerEnv(&processContainerEnvInput{
		containerInfo: containerInfo,
		formattedEnv:  formattedEnv,
		replace:       true,
	})...)

	return patch, nil
}

func (p *Patch) FormatEnv(containerInfo *types.ContainerInfo, containersEnv []corev1.EnvVar) ([]corev1.EnvVar, error) {
	var err error

	formattedEnv := make([]corev1.EnvVar, 0)

	for _, containerEnv := range containersEnv {
		item := containerEnv

		item.Value, err = template.Get(containerInfo, item.Value)
		if err != nil {
			return nil, errors.Wrap(err, "error template value")
		}

		formattedEnv = append(formattedEnv, item)
	}

	return formattedEnv, nil
}
