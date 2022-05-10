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
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/api"
	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	namespace      = "test-pod-admission-controller"
	podNamesPrefix = "test-pod-"
)

var updateRequirements = flag.Bool("updateRequirements", false, "update requirements")

func Test(t *testing.T) { //nolint:cyclop
	t.Parallel()

	if err := flag.Set("config", "testdata/config.yaml"); err != nil {
		t.Fatal(err)
	}

	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	if err := api.Init(); err != nil {
		t.Fatal(err)
	}

	pods, err := api.Clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: "app=test-pod-admission-controller",
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, pod := range pods.Items {
		podIndexString := strings.TrimPrefix(pod.Name, podNamesPrefix)

		podIndex, err := strconv.Atoi(podIndexString)
		if err != nil {
			t.Fatal(err)
		}

		// remove kubernetes annotations from pod annotations
		standartPodAnnotations := pod.Annotations
		delete(standartPodAnnotations, "kubectl.kubernetes.io/last-applied-configuration")
		delete(standartPodAnnotations, "kubernetes.io/psp")

		if err = compareObject(pod.Annotations, podIndex, "Annotations"); err != nil {
			t.Errorf("pod Annotations %d %s", podIndex, err)
		}

		if err = compareObject(pod.Labels, podIndex, "Labels"); err != nil {
			t.Errorf("pod Labels %d %s", podIndex, err)
		}

		for containerOrder := range pod.Spec.InitContainers {
			if err := CheckInitContainer(podIndex, containerOrder, pod.Spec.InitContainers); err != nil {
				t.Errorf("InitContainers %d/%d %s", podIndex, containerOrder, err)
			}
		}

		for containerOrder := range pod.Spec.Containers {
			if err := CheckContainer(podIndex, containerOrder, pod.Spec.Containers); err != nil {
				t.Errorf("Containers %d/%d %s", podIndex, containerOrder, err)
			}
		}
	}
}

func compareObject(obj interface{}, podOrder int, filePath string) error {
	objOut, err := json.Marshal(obj)
	if err != nil {
		return errors.Wrap(err, "error in Marshal")
	}

	jsonPath := fmt.Sprintf("requirements/Pods/%d/%s.json", podOrder, filePath)

	requireEnvByte, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		return errors.Wrap(err, "error in ioutil.ReadFile")
	}

	in := strings.TrimSpace(string(requireEnvByte))
	out := strings.TrimSpace(string(objOut))

	if in != out {
		// write updated requirements
		if *updateRequirements {
			err := ioutil.WriteFile(jsonPath, objOut, 0)
			if err != nil {
				return errors.Wrap(err, "error in ioutil.WriteFile")
			}

			return nil
		}

		return errors.Errorf("not equal %s \n\nin=(%s)\n\nout=(%s)", jsonPath, in, out)
	}

	return nil
}

func CheckInitContainer(podIndex int, containerOrder int, initContainers []corev1.Container) error {
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

func CheckContainer(podIndex int, containerOrder int, containers []corev1.Container) error {
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
