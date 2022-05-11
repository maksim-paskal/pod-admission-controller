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
package api

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/maksim-paskal/pod-admission-controller/pkg/metrics"
	"github.com/maksim-paskal/pod-admission-controller/pkg/patch"
	"github.com/maksim-paskal/pod-admission-controller/pkg/template"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/maksim-paskal/pod-admission-controller/pkg/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
	Clientset     *kubernetes.Clientset
	Restconfig    *rest.Config
)

func Init() error {
	var err error

	if len(*config.Get().KubeConfigFile) > 0 {
		Restconfig, err = clientcmd.BuildConfigFromFlags("", *config.Get().KubeConfigFile)
		if err != nil {
			return errors.Wrap(err, "error in clientcmd.BuildConfigFromFlags")
		}
	} else {
		log.Debug("No kubeconfig file use incluster")
		Restconfig, err = rest.InClusterConfig()
		if err != nil {
			return errors.Wrap(err, "error in rest.InClusterConfig")
		}
	}

	Clientset, err = kubernetes.NewForConfig(Restconfig)
	if err != nil {
		log.WithError(err).Fatal()
	}

	return nil
}

func Mutate(namespace *corev1.Namespace, requestedAdmissionReview *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse { //nolint:funlen,cyclop,lll
	req := requestedAdmissionReview.Request

	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		return mutateError(namespace.Name, err)
	}

	// some pods does not need mutation
	// pod-admission-controller/ignore=true
	if ignore, ok := pod.Annotations[types.AnnotationIgnore]; ok {
		if strings.EqualFold(ignore, "true") {
			metrics.MutationsIgnored.WithLabelValues(namespace.Name).Inc()

			return &admissionv1.AdmissionResponse{
				Allowed:  true,
				Warnings: []string{types.WarningPodDoedNotNeedMutation},
			}
		}
	}

	mutationPatch := make([]types.PatchOperation, 0)

	for order, container := range pod.Spec.Containers {
		containerInfo := types.ContainerInfo{
			ContainerName:        container.Name,
			Image:                container.Image,
			Namespace:            namespace.Name,
			NamespaceAnnotations: namespace.Annotations,
			NamespaceLabels:      namespace.Labels,
			PodAnnotations:       pod.Annotations,
			PodLabels:            pod.Labels,
			SelectedRules:        []types.Rule{},
		}

		// check rule that corresponds to container
		for _, rule := range config.Get().Rules {
			match, err := utils.CheckConditions(containerInfo, rule.Conditions)
			if err != nil {
				return mutateError(namespace.Name, err)
			}

			if match {
				containerInfo.SelectedRules = append(containerInfo.SelectedRules, rule)
			}
		}

		// if no rules found for container continue to next container
		if len(containerInfo.SelectedRules) == 0 {
			continue
		}

		// get all formatted envs
		formattedEnv, err := FormatEnv(containerInfo, containerInfo.GetSelectedRulesEnv())
		if err != nil {
			return mutateError(namespace.Name, err)
		}

		// create env patch if env is not empty
		if len(formattedEnv) > 0 {
			mutationPatch = append(mutationPatch, patch.CreateEnvPatch(order, containerInfo, container.Env, formattedEnv)...)
		}

		// add default resources to container if rules have adddefaultresources enabled
		if selectedRule, ok := containerInfo.GetSelectedRuleEnabled(types.SelectedRuleAddDefaultResources); ok {
			mutationPatch = append(mutationPatch, patch.CreateDefaultResourcesPatch(selectedRule, order, containerInfo, container.Resources)...) //nolint:lll
		}

		// add default resources to container if rules have runasnonroot enabled
		if selectedRule, ok := containerInfo.GetSelectedRuleEnabled(types.SelectedRuleRunAsNonRoot); ok {
			mutationPatch = append(mutationPatch, patch.CreateRunAsNonRootPatch(selectedRule, order, containerInfo, pod.Spec.SecurityContext, container.SecurityContext)...) //nolint:lll
		}
	}

	// if no patches found return empty response
	if len(mutationPatch) == 0 {
		return &admissionv1.AdmissionResponse{
			Allowed:  true,
			Warnings: []string{types.WarningNoPatchGenerated},
		}
	}

	podAnnotations := make(map[string]string)
	if pod.Annotations != nil {
		podAnnotations = pod.Annotations
	}

	podAnnotations[types.AnnotationInjected] = "true"

	mutationPatch = append(mutationPatch, types.PatchOperation{
		Op:    "add",
		Path:  "/metadata/annotations",
		Value: podAnnotations,
	})

	patchBytes, err := json.Marshal(mutationPatch)
	if err != nil {
		return mutateError(namespace.Name, err)
	}

	log.Infof("mutate %s/%s", req.Namespace, req.UID)

	log.Debugf("patch=%s", string(patchBytes))

	metrics.MutationsTotal.WithLabelValues(namespace.Name).Inc()

	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Result: &metav1.Status{
			Status: metav1.StatusSuccess,
		},
		Patch: patchBytes,
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch

			return &pt
		}(),
	}
}

func mutateError(namespaceName string, err error) *admissionv1.AdmissionResponse {
	log.WithError(err).Error("Error mutating")

	metrics.MutationsError.WithLabelValues(namespaceName).Inc()

	return &admissionv1.AdmissionResponse{
		Result: &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: err.Error(),
		},
	}
}

func ParseRequest(body []byte) ([]byte, error) {
	obj, gvk, err := deserializer.Decode(body, nil, &admissionv1.AdmissionReview{})
	if err != nil {
		return nil, errors.Wrap(err, "Request could not be decoded")
	}

	var responseObj runtime.Object

	requestedAdmissionReview, ok := obj.(*admissionv1.AdmissionReview)
	if !ok {
		return nil, errors.Errorf("Expected v1.AdmissionReview but got: %T", obj)
	}

	namespace, err := Clientset.CoreV1().Namespaces().Get(context.Background(), requestedAdmissionReview.Request.Namespace, metav1.GetOptions{}) //nolint:lll
	if err != nil {
		return nil, errors.Wrap(err, "Could not get namespace")
	}

	responseAdmissionReview := &admissionv1.AdmissionReview{}
	responseAdmissionReview.SetGroupVersionKind(*gvk)
	responseAdmissionReview.Response = Mutate(namespace, requestedAdmissionReview)
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
	responseObj = responseAdmissionReview

	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		return nil, errors.Wrap(err, "can not marshal response")
	}

	return respBytes, nil
}

func FormatEnv(containerInfo types.ContainerInfo, containersEnv []corev1.EnvVar) ([]corev1.EnvVar, error) {
	var err error

	formattedEnv := make([]corev1.EnvVar, 0)

	for _, containerEnv := range containersEnv {
		item := containerEnv

		item.Value, err = template.Get(containerInfo, item.Value)
		if err != nil {
			return nil, errors.Wrap(err, "error template value")
		}

		formattedEnv = append(formattedEnv, item)
	}

	return formattedEnv, nil
}
