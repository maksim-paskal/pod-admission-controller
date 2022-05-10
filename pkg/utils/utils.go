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
package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/maksim-paskal/pod-admission-controller/pkg/template"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/pkg/errors"
)

func CheckConditions(containerInfo types.ContainerInfo, conditions []types.Conditions) (bool, error) { //nolint:cyclop
	if len(conditions) == 0 {
		return true, nil
	}

	var found int

	for _, condition := range conditions {
		key, err := template.Get(containerInfo, fmt.Sprintf("{{ %s }}", condition.Key))
		if err != nil {
			return false, errors.Wrap(err, "error matching key")
		}

		switch strings.ToLower(condition.Operator) {
		case "equal":
			if key == condition.Value {
				found++
			}
		case "regexp":
			match, err := regexp.MatchString(condition.Value, key)
			if err != nil {
				return false, errors.Wrap(err, "error matching regexp")
			}

			if match {
				found++
			}
		default:
			return false, errors.Errorf("unknown operator %s", condition.Operator)
		}
	}

	if found == len(conditions) {
		return true, nil
	}

	return false, nil
}
