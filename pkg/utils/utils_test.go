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
package utils_test

import (
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/maksim-paskal/pod-admission-controller/pkg/utils"
)

const namespace = "test"

func TestCheckConditionsImageEqual(t *testing.T) {
	t.Parallel()

	conditions := make([]types.Conditions, 0)

	conditions = append(conditions, types.Conditions{
		Key:      ".Image.Name",
		Operator: "equal",
		Value:    "test",
	})

	match, err := utils.CheckConditions(types.ContainerInfo{Namespace: namespace, Image: &types.ContainerImage{Name: "fake"}}, conditions) //nolint:lll
	if err != nil {
		t.Fatal(err)
	}

	if match {
		t.Fatal("must be false")
	}

	match, err = utils.CheckConditions(types.ContainerInfo{Namespace: namespace, Image: &types.ContainerImage{Name: "test"}}, conditions) //nolint:lll
	if err != nil {
		t.Fatal(err)
	}

	if !match {
		t.Fatal("must be true")
	}
}

func TestCheckConditionsImageRegexp(t *testing.T) {
	t.Parallel()

	conditions := make([]types.Conditions, 0)

	conditions = append(conditions, types.Conditions{
		Key:      ".Image",
		Operator: "regexp",
		Value:    "te(.*)st",
	})

	match, err := utils.CheckConditions(types.ContainerInfo{Namespace: namespace}, conditions)
	if err != nil {
		t.Fatal(err)
	}

	if match {
		t.Fatal("must be false")
	}

	match, err = utils.CheckConditions(types.ContainerInfo{Namespace: namespace, Image: &types.ContainerImage{Name: "test"}}, conditions) //nolint:lll
	if err != nil {
		t.Fatal(err)
	}

	if !match {
		t.Fatal("must be true")
	}
}

func TestCheckConditionsNamespaceEqual(t *testing.T) {
	t.Parallel()

	conditions := make([]types.Conditions, 0)

	conditions = append(conditions, types.Conditions{
		Key:      ".Namespace",
		Operator: "equal",
		Value:    "test",
	})

	match, err := utils.CheckConditions(types.ContainerInfo{Namespace: "fake"}, conditions)
	if err != nil {
		t.Fatal(err)
	}

	if match {
		t.Fatal("must be false")
	}

	match, err = utils.CheckConditions(types.ContainerInfo{Namespace: namespace}, conditions)
	if err != nil {
		t.Fatal(err)
	}

	if !match {
		t.Fatal("must be true")
	}
}

func TestCheckConditionsPodAnnotationEqual(t *testing.T) {
	t.Parallel()

	conditions := make([]types.Conditions, 0)

	conditions = append(conditions, types.Conditions{
		Key:      ".PodAnnotations.env",
		Operator: "equal",
		Value:    "test",
	})

	podAnnotations := map[string]string{
		"1": "2",
		"3": "4",
		"5": "6",
	}

	match, err := utils.CheckConditions(types.ContainerInfo{PodAnnotations: podAnnotations}, conditions)
	if err != nil {
		t.Fatal(err)
	}

	if match {
		t.Fatal("must be false")
	}

	podAnnotations["env"] = "test"

	match, err = utils.CheckConditions(types.ContainerInfo{PodAnnotations: podAnnotations}, conditions)
	if err != nil {
		t.Fatal(err)
	}

	if !match {
		t.Fatal("must be true")
	}
}

func TestCheckConditionsNamespaceAnnotationEqual(t *testing.T) {
	t.Parallel()

	conditions := make([]types.Conditions, 0)

	conditions = append(conditions, types.Conditions{
		Key:      ".NamespaceAnnotations.env",
		Operator: "equal",
		Value:    "test",
	})

	namespaceAnnotations := map[string]string{
		"1": "2",
		"3": "4",
		"5": "6",
	}

	match, err := utils.CheckConditions(types.ContainerInfo{NamespaceAnnotations: namespaceAnnotations}, conditions)
	if err != nil {
		t.Fatal(err)
	}

	if match {
		t.Fatal("must be false")
	}

	namespaceAnnotations["env"] = "test"

	match, err = utils.CheckConditions(types.ContainerInfo{NamespaceAnnotations: namespaceAnnotations}, conditions)
	if err != nil {
		t.Fatal(err)
	}

	if !match {
		t.Fatal("must be true")
	}
}
