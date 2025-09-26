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
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/api"
	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

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
					Name:  "test",
					Image: "test/test:test",
				},
			},
		},
	})

	// test resources
	pods = append(pods, corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-adddefaultresources",
					Image: "test/test:test",
				},
			},
		},
	})

	// test RunAsNotRoot
	pods = append(pods, corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-runasnonroot",
					Image: "test/test:test",
				},
			},
		},
	})

	for podIndex, pod := range pods {
		podJSON, err := json.Marshal(pod)
		if err != nil {
			t.Fatal(err)
		}

		input := api.MutateInput{
			Namespace: &corev1.Namespace{},
			AdmissionReview: &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Namespace: "test",
					Resource: metav1.GroupVersionResource{
						Resource: "pods",
						Version:  "v1",
					},
					Object: runtime.RawExtension{
						Raw: podJSON,
					},
				},
			},
		}

		response := api.NewMutation().Mutate(t.Context(), &input)

		if response.Result.Status != "Success" {
			t.Fatalf("status must be Success, got %s, %s", response.Result.Status, response.Result.Message)
		}

		jsonPath := fmt.Sprintf("testdata/patch-%d.json", podIndex)

		requireByte, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatal(err)
		}

		in := strings.TrimSpace(string(requireByte))

		if in != string(response.Patch) {
			t.Fatalf("%s,must be equal\n\nin=(%s)\n\nout=(%s)", jsonPath, in, string(response.Patch))
		}
	}
}

func TestGetImageInfo(t *testing.T) {
	t.Parallel()

	tests := make(map[string]*types.ContainerImage)

	tests["10.10.10.10:5000/product/main/backend:release-20220516-1"] = &types.ContainerImage{
		Domain: "10.10.10.10:5000",
		Path:   "product/main/backend",
		Name:   "10.10.10.10:5000/product/main/backend:release-20220516-1",
		Slug:   "product-main-backend",
		Tag:    "release-20220516-1",
	}
	tests["10.10.10.10:5000/product/main/front:release-20220516-1"] = &types.ContainerImage{
		Domain: "10.10.10.10:5000",
		Path:   "product/main/front",
		Name:   "10.10.10.10:5000/product/main/front:release-20220516-1",
		Slug:   "product-main-front",
		Tag:    "release-20220516-1",
	}
	tests["domain.com/hipages/php-fpm_exporter:1"] = &types.ContainerImage{
		Domain: "domain.com",
		Path:   "hipages/php-fpm_exporter",
		Name:   "domain.com/hipages/php-fpm_exporter:1",
		Slug:   "hipages-php-fpm-exporter",
		Tag:    "1",
	}
	tests["domain.com/paskalmaksim/envoy-docker-image:v0.3.8"] = &types.ContainerImage{
		Domain: "domain.com",
		Path:   "paskalmaksim/envoy-docker-image",
		Name:   "domain.com/paskalmaksim/envoy-docker-image:v0.3.8",
		Slug:   "paskalmaksim-envoy-docker-image",
		Tag:    "v0.3.8",
	}
	tests["paskalmaksim/envoy-docker-image:v0.3.8"] = &types.ContainerImage{
		Domain: "docker.io",
		Path:   "paskalmaksim/envoy-docker-image",
		Name:   "paskalmaksim/envoy-docker-image:v0.3.8",
		Slug:   "paskalmaksim-envoy-docker-image",
		Tag:    "v0.3.8",
	}
	tests["paskalmaksim/envoy-docker-image"] = &types.ContainerImage{
		Domain: "docker.io",
		Path:   "paskalmaksim/envoy-docker-image",
		Name:   "paskalmaksim/envoy-docker-image",
		Slug:   "paskalmaksim-envoy-docker-image",
		Tag:    "latest",
	}

	for test, requre := range tests {
		formattedImage, err := api.GetImageInfo(test)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(requre, formattedImage) {
			t.Fatalf("must be:\n%+v, got:\n%+v", requre, formattedImage)
		}
	}
}
