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
	"slices"
	"strings"

	"github.com/maksim-paskal/pod-admission-controller/pkg/template"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/pkg/errors"
)

const negate = "not"

func CheckConditions(containerInfo *types.ContainerInfo, conditions []types.Conditions) (bool, error) { //nolint:cyclop
	if len(conditions) == 0 {
		return true, nil
	}

	var found int

	for _, condition := range conditions {
		if len(condition.Key) == 0 {
			return false, errors.Errorf("empty key")
		}

		key, err := template.Get(containerInfo, fmt.Sprintf("{{ %s }}", condition.Key))
		if err != nil {
			return false, errors.Wrap(err, "error matching key")
		}

		conditionRequired := !strings.HasPrefix(strings.ToLower(condition.Operator), negate)

		switch strings.ToLower(condition.Operator) {
		case "equal", negate + "equal":
			if len(condition.Value) == 0 {
				return false, errors.Errorf("empty value for operator %s", condition.Operator)
			}

			if (key == condition.Value) == conditionRequired {
				found++
			}
		case "regexp", negate + "regexp":
			if len(condition.Value) == 0 {
				return false, errors.Errorf("empty value for operator %s", condition.Operator)
			}

			match, err := regexp.MatchString(condition.Value, key)
			if err != nil {
				return false, errors.Wrap(err, "error matching regexp")
			}

			if match == conditionRequired {
				found++
			}
		case "in", negate + "in":
			if len(condition.Values) == 0 {
				return false, errors.Errorf("empty values for operator %s", condition.Operator)
			}

			if slices.Contains(condition.Values, key) == conditionRequired {
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
