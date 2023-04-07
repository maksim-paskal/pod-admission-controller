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
	"regexp"
	"strings"

	"github.com/distribution/distribution/v3/reference"
	"github.com/maksim-paskal/pod-admission-controller/pkg/client"
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
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

// mutate pod.
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
			Namespace:            namespace.Name,
			NamespaceAnnotations: namespace.Annotations,
			NamespaceLabels:      namespace.Labels,
			PodAnnotations:       pod.Annotations,
			PodLabels:            pod.Labels,
			SelectedRules:        []*types.Rule{},
		}

		imageInfo, err := GetImageInfo(container.Image)
		if err != nil {
			return mutateError(namespace.Name, err)
		}

		containerInfo.Image = imageInfo

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

// throw mutaion errors.
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

// parse http request.
func ParseRequest(ctx context.Context, body []byte) ([]byte, error) {
	obj, gvk, err := deserializer.Decode(body, nil, &admissionv1.AdmissionReview{})
	if err != nil {
		return nil, errors.Wrap(err, "Request could not be decoded")
	}

	var responseObj runtime.Object

	requestedAdmissionReview, ok := obj.(*admissionv1.AdmissionReview)
	if !ok {
		return nil, errors.Errorf("Expected v1.AdmissionReview but got: %T", obj)
	}

	namespace, err := client.KubeClient().CoreV1().Namespaces().Get(ctx, requestedAdmissionReview.Request.Namespace, metav1.GetOptions{}) //nolint:lll
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

// template values in container environment variables.
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

// parse image to repo, slug path and tag.
func GetImageInfo(image string) (*types.ContainerImage, error) {
	// check if image is has fully qualified name
	refName, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing image name %s", image)
	}

	// get only lowered the image name
	imageName := strings.ToLower(reference.Path(refName))

	// replace all non alphanumeric characters with a dash
	imageName = regexp.MustCompile("[^A-Za-z0-9]+").ReplaceAllString(imageName, "-")

	result := types.ContainerImage{
		Name: image,
		Slug: strings.Trim(imageName, "-"),
		Tag:  "latest",
	}

	if tag, ok := refName.(reference.Tagged); ok {
		result.Tag = tag.Tag()
	}

	return &result, nil
}
