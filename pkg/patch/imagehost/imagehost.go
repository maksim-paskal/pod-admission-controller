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
package imagehost

import (
	"context"
	"regexp"
	"strings"

	"github.com/maksim-paskal/pod-admission-controller/pkg/template"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/pkg/errors"
)

type Patch struct{}

func (p *Patch) Create(_ context.Context, containerInfo *types.ContainerInfo) ([]types.PatchOperation, error) { //nolint:lll
	patch := make([]types.PatchOperation, 0)

	image := containerInfo.Image.Name

	if !strings.HasPrefix(image, containerInfo.Image.Domain) {
		// for short image names like nginx - use library/nginx
		if strings.Count(image, "/") == 0 {
			image = "library/" + image
		}

		// if image name does not contain domain - add it
		image = containerInfo.Image.Domain + "/" + image
	}

	for _, selectedRule := range containerInfo.SelectedRules {
		if !selectedRule.ReplaceContainerImageHost.Enabled {
			continue
		}

		selectedRule.Logf("CreateReplaceContainerImageHost: %+v", selectedRule)

		// use image domain if from is empty
		fromPattern := selectedRule.ReplaceContainerImageHost.From
		if len(fromPattern) == 0 {
			fromPattern = containerInfo.Image.Domain
		}

		selectedRule.Logf("CreateReplaceContainerImageHost: image=%s", image)
		selectedRule.Logf("CreateReplaceContainerImageHost: to=%s", selectedRule.ReplaceContainerImageHost.To)
		selectedRule.Logf("CreateReplaceContainerImageHost: from=%s", fromPattern)

		fromRegexp, err := regexp.Compile(fromPattern)
		if err != nil {
			return nil, errors.Wrap(err, "regexp.Compile")
		}

		value, err := template.Get(containerInfo, selectedRule.ReplaceContainerImageHost.To)
		if err != nil {
			return nil, errors.Wrap(err, "template.Get")
		}

		selectedRule.Logf("CreateReplaceContainerImageHost: value=%s", value)

		result := fromRegexp.ReplaceAll([]byte(image), []byte(value))

		selectedRule.Logf("CreateReplaceContainerImageHost: result=%s", result)

		patch = append(patch, types.PatchOperation{
			Op:    "replace",
			Path:  containerInfo.PodContainer.ContainerPath() + "/image",
			Value: string(result),
		})

		// process only first rule
		break
	}

	return patch, nil
}
