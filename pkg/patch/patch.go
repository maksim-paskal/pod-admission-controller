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
	"context"
	"reflect"
	"strings"

	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/custompatch"
	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/env"
	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/imagehost"
	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/nonroot"
	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/pullsecrets"
	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/resources"
	"github.com/maksim-paskal/pod-admission-controller/pkg/patch/tolerations"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/pkg/errors"
)

type Patch interface {
	// create patch
	Create(ctx context.Context, containerInfo *types.ContainerInfo) ([]types.PatchOperation, error)
}

var allPatchs = []Patch{
	&env.Patch{},
	&nonroot.Patch{},
	&resources.Patch{},
	&imagehost.Patch{},
	&tolerations.Patch{},
	&pullsecrets.Patch{},
	&custompatch.Patch{},
}

func NewPatch(ctx context.Context, containerInfo *types.ContainerInfo) ([]types.PatchOperation, error) {
	result := make([]types.PatchOperation, 0)

	for _, patch := range allPatchs {
		if IgnoreContainerPatch(patch, containerInfo) {
			continue
		}

		patchOps, err := patch.Create(ctx, containerInfo)
		if err != nil {
			return nil, errors.Wrapf(err, "error in %s", getPatchName(patch))
		}

		result = append(result, patchOps...)
	}

	return result, nil
}

func getPatchName(patch Patch) string {
	patchName := reflect.TypeOf(patch).String()

	patchName = strings.TrimPrefix(patchName, "*")
	patchName = strings.TrimSuffix(patchName, ".Patch")

	return patchName
}

const annotationIgnorePrefix = "pod-admission-controller/ignore-"

// ignore patch if annotation exists:
// pod-admission-controller/ignore-<patch-name>=<container-name>[,<container-name>].
func IgnoreContainerPatch(patch Patch, containerInfo *types.ContainerInfo) bool {
	ignore, ok := containerInfo.GetPodAnnotation(annotationIgnorePrefix + getPatchName(patch))
	if !ok {
		return false
	}

	if ignore == "*" {
		return true
	}

	containersNames := strings.Split(ignore, ",")
	for _, containerName := range containersNames {
		if containerName == containerInfo.ContainerName {
			return true
		}
	}

	return false
}
