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
package conditions_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/conditions"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
)

func TestCheckConditions(t *testing.T) { //nolint:funlen,maintidx
	t.Parallel()

	os.Setenv("TEST_ENV", "test") //nolint:tenv

	containerInfo := &types.ContainerInfo{
		Namespace: "1234567890",
		Image:     &types.ContainerImage{Name: "alpine:3.12"},
		PodAnnotations: map[string]string{
			"asd": "erty",
			"env": "test",
		},
		NamespaceAnnotations: map[string]string{
			"123":    "456",
			"ABC":    "DEF",
			"qwerty": "1234567890",
			"env":    "asdasdq",
		},
	}

	type testStruct struct {
		Error      bool
		Conditions []types.Conditions
		Match      bool
	}

	tests := []testStruct{
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".NamespaceAnnotations.ABC",
					Operator: "in",
					Values:   []string{"ABC", "123", "DEF", "345"},
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Conditions{
				{
					Key:      ".NamespaceAnnotations.ABC",
					Operator: "NotIn",
					Values:   []string{"ABC", "123", "DEF", "345"},
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Conditions{
				{
					Key:      ".NamespaceAnnotations.ABC",
					Operator: "in",
					Values:   []string{"ff"},
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".NamespaceAnnotations.ABC",
					Operator: "notin",
					Values:   []string{"ff"},
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".NamespaceAnnotations.SOMEFAKE",
					Operator: "notin",
					Values:   []string{"ff"},
				},
			},
		},
		{
			Error: true,
			Conditions: []types.Conditions{
				{
					Key:      ".NamespaceAnnotations.ABC",
					Operator: "in",
					Value:    "dd",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Conditions{
				{
					Key:      ".Image.Name",
					Operator: "equal",
					Value:    "test",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".Image.Name",
					Operator: "notequal",
					Value:    "test",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".Image.Name",
					Operator: "equal",
					Value:    "alpine:3.12",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Conditions{
				{
					Key:      ".Image",
					Operator: "regexp",
					Value:    "te(.*)st",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".Image",
					Operator: "notRegexp",
					Value:    "te(.*)st",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".Image",
					Operator: "regexp",
					Value:    "alpine.+",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".Namespace",
					Operator: "equal",
					Value:    "1234567890",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Conditions{
				{
					Key:      ".Namespace",
					Operator: "equal",
					Value:    "dasasdsad",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".PodAnnotations.env",
					Operator: "equal",
					Value:    "test",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Conditions{
				{
					Key:      ".PodAnnotations.env",
					Operator: "equal",
					Value:    "asdasdq",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".NamespaceAnnotations.env",
					Operator: "equal",
					Value:    "asdasdq",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Conditions{
				{
					Key:      ".NamespaceAnnotations.env",
					Operator: "equal",
					Value:    "123123",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".NamespaceAnnotations.KKK",
					Operator: "notequal",
					Value:    "dd",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      `env "TEST_ENV"`,
					Operator: "equal",
					Value:    "test",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Conditions{
				{
					Key:      `env "TEST_ENV"`,
					Operator: "equal",
					Value:    "01lw8",
				},
			},
		},
		{
			Match:      true,
			Conditions: []types.Conditions{},
		},
		{
			Error: true,
			Conditions: []types.Conditions{
				{
					Key:      ".Namespace",
					Operator: "unknownOperator",
				},
			},
		},
		{
			Error: true,
			Conditions: []types.Conditions{
				{
					Key: "unknownKey",
				},
			},
		},
		{
			Error: true,
			Conditions: []types.Conditions{
				{
					Key: "",
				},
			},
		},
		{
			Error: true,
			Conditions: []types.Conditions{
				{
					Key:      ".Image",
					Operator: "regexp",
					Value:    `\`,
				},
			},
		},
		{
			Error: true,
			Conditions: []types.Conditions{
				{
					Key:      ".Image",
					Operator: "equal",
				},
			},
		},
		{
			Error: true,
			Conditions: []types.Conditions{
				{
					Key:      ".Image",
					Operator: "regexp",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".ContainerName",
					Operator: "empty",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Conditions{
				{
					Key:      ".ContainerName",
					Operator: "NotEmpty",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Conditions{
				{
					Key:      ".Namespace",
					Operator: "NotEmpty",
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(fmt.Sprintf("%+v", test), func(t *testing.T) {
			t.Parallel()

			match, err := conditions.Check(containerInfo, test.Conditions)
			if test.Error {
				if err == nil {
					t.Fatal("must be error")
				} else {
					t.Skip("is error")
				}
			}

			if err != nil {
				t.Fatal(err)
			}

			if match != test.Match {
				t.Fatalf("must be %v, got %v", test.Match, match)
			}
		})
	}
}
