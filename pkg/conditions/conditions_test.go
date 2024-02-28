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
	"os"
	"strconv"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/conditions"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseConditionKey(t *testing.T) {
	t.Parallel()

	condition := types.Condition{
		Key: ".Namespace",
	}

	containerInfo := &types.ContainerInfo{
		Namespace: "1234567890",
	}

	key, err := conditions.ParseConditionKey(containerInfo, condition)
	if err != nil {
		t.Fatal(err)
	}

	if key != "1234567890" {
		t.Fatalf("must be 1234567890, got %s", key)
	}
}

func TestCheckConditions(t *testing.T) { //nolint:funlen,maintidx
	t.Parallel()
	log.SetLevel(log.DebugLevel)

	os.Setenv("TEST_ENV", "test") //nolint:tenv

	containerInfo := &types.ContainerInfo{
		Namespace: "1234567890",
		Image:     &types.ContainerImage{Name: "alpine:3.12"},
		PodContainer: &types.PodContainer{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "v1",
							Kind:       "ReplicaSet",
							Name:       "test",
						},
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "test-volume-01",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "test-pvc-01",
								},
							},
						},
						{
							Name: "test-volume-02",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "test-pvc-02",
								},
							},
						},
						{
							Name: "test-volume-03",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "test-pvc-02",
								},
							},
						},
					},
				},
			},
		},
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
		Conditions []types.Condition
		Match      bool
	}

	tests := []testStruct{
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".NamespaceAnnotations.ABC",
					Operator: "in",
					Values:   []string{"ABC", "123", "DEF", "345"},
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Condition{
				{
					Key:      ".NamespaceAnnotations.ABC",
					Operator: "NotIn",
					Values:   []string{"ABC", "123", "DEF", "345"},
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Condition{
				{
					Key:      ".NamespaceAnnotations.ABC",
					Operator: "in",
					Values:   []string{"ff"},
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".NamespaceAnnotations.ABC",
					Operator: "notin",
					Values:   []string{"ff"},
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".NamespaceAnnotations.SOMEFAKE",
					Operator: "notin",
					Values:   []string{"ff"},
				},
			},
		},
		{
			Error: true,
			Conditions: []types.Condition{
				{
					Key:      ".NamespaceAnnotations.ABC",
					Operator: "in",
					Value:    "dd",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Condition{
				{
					Key:      ".Image.Name",
					Operator: "equal",
					Value:    "test",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".Image.Name",
					Operator: "notequal",
					Value:    "test",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".Image.Name",
					Operator: "equal",
					Value:    "alpine:3.12",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Condition{
				{
					Key:      ".Image",
					Operator: "regexp",
					Value:    "te(.*)st",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".Image",
					Operator: "notRegexp",
					Value:    "te(.*)st",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".Image",
					Operator: "regexp",
					Value:    "alpine.+",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".Namespace",
					Operator: "equal",
					Value:    "1234567890",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Condition{
				{
					Key:      ".Namespace",
					Operator: "equal",
					Value:    "dasasdsad",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".PodAnnotations.env",
					Operator: "equal",
					Value:    "test",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Condition{
				{
					Key:      ".PodAnnotations.env",
					Operator: "equal",
					Value:    "asdasdq",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".NamespaceAnnotations.env",
					Operator: "equal",
					Value:    "asdasdq",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Condition{
				{
					Key:      ".NamespaceAnnotations.env",
					Operator: "equal",
					Value:    "123123",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".NamespaceAnnotations.KKK",
					Operator: "notequal",
					Value:    "dd",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      `env "TEST_ENV"`,
					Operator: "equal",
					Value:    "test",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Condition{
				{
					Key:      `env "TEST_ENV"`,
					Operator: "equal",
					Value:    "01lw8",
				},
			},
		},
		{
			Match:      true,
			Conditions: []types.Condition{},
		},
		{
			Error: true,
			Conditions: []types.Condition{
				{
					Key:      ".Namespace",
					Operator: "unknownOperator",
				},
			},
		},
		{
			Error: true,
			Conditions: []types.Condition{
				{
					Key: "unknownKey",
				},
			},
		},
		{
			Error: true,
			Conditions: []types.Condition{
				{
					Key: "",
				},
			},
		},
		{
			Error: true,
			Conditions: []types.Condition{
				{
					Key:      ".Image",
					Operator: "regexp",
					Value:    `\`,
				},
			},
		},
		{
			Error: true,
			Conditions: []types.Condition{
				{
					Key:      ".Image",
					Operator: "equal",
				},
			},
		},
		{
			Error: true,
			Conditions: []types.Condition{
				{
					Key:      ".Image",
					Operator: "regexp",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".ContainerName",
					Operator: "empty",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Condition{
				{
					Key:      ".ContainerName",
					Operator: "NotEmpty",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".NamespaceAnnotations.SOMEFAKE",
					Operator: "Empty",
				},
			},
		},
		{
			Match: false,
			Conditions: []types.Condition{
				{
					Key:      ".NamespaceAnnotations.SOMEFAKE",
					Operator: "NotEmpty",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".PodContainer.PodPVCNames",
					Operator: "equal",
					Value:    "[test-pvc-01 test-pvc-02]",
				},
			},
		},
		{
			Match: true,
			Conditions: []types.Condition{
				{
					Key:      ".PodContainer.OwnerKind",
					Operator: "equal",
					Value:    "ReplicaSet",
				},
			},
		},
	}

	for testID, test := range tests {
		test := test

		t.Run(strconv.Itoa(testID), func(t *testing.T) {
			t.Parallel()

			t.Logf("%+v", test)

			for conditionID, condition := range test.Conditions {
				test.Conditions[conditionID].Operator = condition.Operator.Value()

				keyValue, err := conditions.ParseConditionKey(containerInfo, condition)
				if err != nil {
					t.Logf("error in parsing key: %s, %s", condition.Key, err)
				} else {
					t.Logf("key %s=%s", condition.Key, keyValue)
				}
			}

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
