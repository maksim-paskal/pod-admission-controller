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
package e2e_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/client"
	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	namespace      = "test-pod-admission-controller"
	podNamesPrefix = "test-pod-"
)

var updateRequirements = flag.Bool("updateRequirements", false, "update requirements")

func TestMutation(t *testing.T) { //nolint:funlen
	t.Parallel()

	if err := flag.Set("config", "testdata/config.yaml"); err != nil {
		t.Fatal(err)
	}

	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	if err := client.Init(); err != nil {
		t.Fatal(err)
	}

	ctx := context.TODO()

	t.Run("pods", func(t *testing.T) {
		t.Parallel()

		pods, err := client.KubeClient().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app=test-pod-admission-controller",
		})
		if err != nil {
			t.Fatal(err)
		}

		for _, pod := range pods.Items {
			t.Run(pod.Name, func(t *testing.T) {
				t.Parallel()

				if err := testPod(pod); err != nil {
					t.Fatal(err)
				}
			})
		}
	})

	t.Run("secrets", func(t *testing.T) {
		t.Parallel()

		if err := testSecret(ctx); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("ingresses", func(t *testing.T) {
		t.Parallel()

		type test struct {
			name              string
			requireAnnotation bool
			requiredHosts     []string
			requiredTLSHosts  []string
		}

		tests := []test{
			{
				name:              "test-ingress-0",
				requireAnnotation: true,
				requiredHosts: []string{
					"aaaa.sometestdomain.com",
					"aaaa.com",
					"aaaa.sometestdomain.com",
				},
				requiredTLSHosts: []string{
					"aaaa.sometestdomain.com",
					"aaaa.com",
				},
			},
			{
				name:              "test-ingress-1",
				requireAnnotation: true,
				requiredHosts: []string{
					"bbbb.abracodabra.com",
					"bbbb.com",
					"bbbb.abracodabra.com",
				},
				requiredTLSHosts: []string{
					"bbbb.abracodabra.com",
					"bbbb.com",
				},
			},
			{
				name:              "test-ingress-2",
				requireAnnotation: false,
				requiredHosts: []string{
					"cccc.com",
				},
				requiredTLSHosts: []string{
					"cccc.com",
				},
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				if err := testIngress(ctx,
					test.name,
					test.requireAnnotation,
					test.requiredHosts,
					test.requiredTLSHosts,
				); err != nil {
					t.Fatal(err)
				}
			})
		}
	})
}

func testPod(pod corev1.Pod) error {
	podIndexString := strings.TrimPrefix(pod.Name, podNamesPrefix)

	podIndex, err := strconv.Atoi(podIndexString)
	if err != nil {
		return errors.Wrap(err, "error in strconv.Atoi")
	}

	// remove kubernetes annotations from pod annotations
	standartPodAnnotations := pod.Annotations
	delete(standartPodAnnotations, "kubectl.kubernetes.io/last-applied-configuration")
	delete(standartPodAnnotations, "kubernetes.io/psp")

	if err = compareObject(pod.Annotations, podIndex, "Annotations"); err != nil {
		return errors.Wrap(err, "error in compareObject pod.Annotations")
	}

	if err = compareObject(pod.Labels, podIndex, "Labels"); err != nil {
		return errors.Wrap(err, "error in compareObject pod.Labels")
	}

	for containerOrder := range pod.Spec.InitContainers {
		if err := checkInitContainer(podIndex, containerOrder, pod.Spec.InitContainers); err != nil {
			return errors.Errorf("InitContainers %d/%d %s", podIndex, containerOrder, err)
		}
	}

	for containerOrder := range pod.Spec.Containers {
		if err := checkContainer(podIndex, containerOrder, pod.Spec.Containers); err != nil {
			return errors.Errorf("Containers %d/%d %s", podIndex, containerOrder, err)
		}
	}

	return nil
}

func testIngress(ctx context.Context, name string, requireAnnotation bool, requiredHosts, requiredTLSHosts []string) error { //nolint:lll
	ingress, err := client.KubeClient().NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "error in ingress get")
	}

	if _, ok := ingress.Annotations[types.AnnotationInjected]; !ok == requireAnnotation {
		return errors.Errorf("ingress must have injected annotation %s", ingress.Annotations)
	}

	if req := len(requiredHosts); len(ingress.Spec.Rules) != req {
		return errors.Errorf("ingress must have %d rules", req)
	}

	for hostID, host := range requiredHosts {
		if ingress.Spec.Rules[hostID].Host != host {
			return errors.Errorf("host %d, req=%s, got=%s", hostID, host, ingress.Spec.Rules[hostID].Host)
		}
	}

	if req := 1; len(ingress.Spec.TLS) != req {
		return errors.Errorf("ingress must have %d tls", req)
	}

	if req := len(requiredTLSHosts); len(ingress.Spec.TLS[0].Hosts) != req {
		return errors.Errorf("ingress must have %d tls hosts", req)
	}

	for hostID, host := range requiredTLSHosts {
		if ingress.Spec.TLS[0].Hosts[hostID] != host {
			return errors.Errorf("tls host %d, req=%s, got=%s", hostID, host, ingress.Spec.TLS[0].Hosts[hostID])
		}
	}

	return nil
}

func testSecret(ctx context.Context) error {
	secret, err := client.KubeClient().CoreV1().Secrets(namespace).Get(ctx, "test-secret", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "error in secret get")
	}

	if req := 1; len(secret.Data) != req {
		return errors.Errorf("secret must have %d data", req)
	}

	if req := 1; len(secret.Data) != req {
		return errors.Errorf("secret must have %d data", req)
	}

	secretValue, ok := secret.Data["test"]
	if !ok {
		return errors.Errorf("secret must have test key")
	}

	if val := string(secretValue); val != "value\n" {
		return errors.Errorf("secret value not correct %s", val)
	}

	return nil
}

func compareObject(obj interface{}, podOrder int, filePath string) error {
	objOut, err := json.Marshal(obj)
	if err != nil {
		return errors.Wrap(err, "error in Marshal")
	}

	jsonPath := fmt.Sprintf("requirements/Pods/%d/%s.json", podOrder, filePath)

	requireEnvByte, err := os.ReadFile(jsonPath)
	if err != nil {
		return errors.Wrap(err, "error in os.ReadFile")
	}

	in := strings.TrimSpace(string(requireEnvByte))
	out := strings.TrimSpace(string(objOut))

	if in != out {
		// write updated requirements
		if *updateRequirements {
			err := os.WriteFile(jsonPath, objOut, 0)
			if err != nil {
				return errors.Wrap(err, "error in os.WriteFile")
			}

			return nil
		}

		return errors.Errorf("not equal %s \n\nin=(%s)\n\nout=(%s)", jsonPath, in, out)
	}

	return nil
}

func checkInitContainer(podIndex int, containerOrder int, initContainers []corev1.Container) error {
	initContainer := initContainers[containerOrder]
	filePrefix := fmt.Sprintf("initContainers/%d", containerOrder)

	if err := compareObject(initContainer.Env, podIndex, filePrefix+"/Env"); err != nil {
		return errors.Wrap(err, "error in compareObject init.Env")
	}

	if err := compareObject(initContainer.SecurityContext, podIndex, filePrefix+"/SecurityContext"); err != nil {
		return errors.Wrap(err, "error in compareObject init.SecurityContext")
	}

	if err := compareObject(initContainer.Resources, podIndex, filePrefix+"/Resources"); err != nil {
		return errors.Wrap(err, "error in compareObject init.Resources")
	}

	return nil
}

func checkContainer(podIndex int, containerOrder int, containers []corev1.Container) error {
	container := containers[containerOrder]
	filePrefix := fmt.Sprintf("containers/%d", containerOrder)

	if err := compareObject(container.Env, podIndex, filePrefix+"/Env"); err != nil {
		return errors.Wrap(err, "error in compareObject container.Env")
	}

	if err := compareObject(container.SecurityContext, podIndex, filePrefix+"/SecurityContext"); err != nil {
		return errors.Wrap(err, "error in compareObject container.SecurityContext")
	}

	if err := compareObject(container.Resources, podIndex, filePrefix+"/Resources"); err != nil {
		return errors.Wrap(err, "error in compareObject container.Resources")
	}

	return nil
}
