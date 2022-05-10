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
package api_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/api"
	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestFormatEnv(t *testing.T) {
	t.Parallel()

	containerEnv := []corev1.EnvVar{
		{
			Name:  "TEST1",
			Value: `{{ index (regexp "/(.+):(.+)$" .Image) 1 }}`,
		},
		{
			Name:  "TEST2",
			Value: "test",
		},
		{
			Name:  "TEST3",
			Value: "{{ .NamespaceLabels.app }}",
		},
		{
			Name:  "TEST4",
			Value: "{{ .NamespaceLabels.unknown }}",
		},
	}

	containerInfo := types.ContainerInfo{
		Image:           "/1/2/3:4",
		NamespaceLabels: map[string]string{"app": "testapp"},
	}

	result, err := api.FormatEnv(containerInfo, containerEnv)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != len(containerEnv) {
		t.Fatal("length must be equal")
	}

	returnResults := []string{
		"TEST1:1/2/3",
		"TEST2:test",
		"TEST3:testapp",
		"TEST4:<no value>",
	}

	for i, returnResult := range returnResults {
		v := fmt.Sprintf("%s:%s", result[i].Name, result[i].Value)

		if v != returnResult {
			t.Fatalf("must be %s, got %s", returnResult, v)
		}
	}
}

func TestMutation(t *testing.T) { //nolint:funlen
	t.Parallel()

	if err := flag.Set("config", "testdata/config-test.yaml"); err != nil {
		t.Fatal(err)
	}

	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	pods := make([]corev1.Pod, 0)

	// test all rules env
	pods = append(pods, corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "test",
				},
			},
		},
	})

	// test resources
	pods = append(pods, corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "test-adddefaultresources",
				},
			},
		},
	})

	// test RunAsNotRoot
	pods = append(pods, corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "test-runasnonroot",
				},
			},
		},
	})

	for podIndex, pod := range pods {
		podJSON, err := json.Marshal(pod)
		if err != nil {
			t.Fatal(err)
		}

		requestedAdmissionReview := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Namespace: "test",
				Object: runtime.RawExtension{
					Raw: podJSON,
				},
			},
		}

		namespace := corev1.Namespace{}
		response := api.Mutate(&namespace, &requestedAdmissionReview)

		if response.Result.Status != "Success" {
			t.Fatal("status must be Success")
		}

		jsonPath := fmt.Sprintf("testdata/patch-%d.json", podIndex)

		requireByte, err := ioutil.ReadFile(jsonPath)
		if err != nil {
			t.Fatal(err)
		}

		in := strings.TrimSpace(string(requireByte))

		if in != string(response.Patch) {
			t.Fatalf("must be equal\n\nin=(%s)\n\nout=(%s)", in, string(response.Patch))
		}
	}
}
