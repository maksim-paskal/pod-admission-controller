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
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/distribution/reference"
	"github.com/maksim-paskal/pod-admission-controller/pkg/client"
	"github.com/maksim-paskal/pod-admission-controller/pkg/conditions"
	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/maksim-paskal/pod-admission-controller/pkg/metrics"
	"github.com/maksim-paskal/pod-admission-controller/pkg/patch"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/maksim-paskal/pod-admission-controller/pkg/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

type MutateInput struct {
	Namespace       *corev1.Namespace
	AdmissionReview *admissionv1.AdmissionReview
}

func (m *MutateInput) GetObjectName() string {
	result := m.GetType()

	if m != nil && m.AdmissionReview != nil && m.AdmissionReview.Request != nil {
		req := m.AdmissionReview.Request

		if len(req.Operation) > 0 {
			result += "/" + string(req.Operation)
		}

		if len(req.Namespace) > 0 {
			result += "/" + req.Namespace
		}

		if len(req.Name) > 0 {
			result += "/" + req.Name
		}

		if len(req.UID) > 0 {
			result += "/" + string(req.UID)
		}
	}

	return result
}

func (m *MutateInput) GetNamespace(ctx context.Context) (*corev1.Namespace, error) {
	if m.Namespace != nil {
		return m.Namespace, nil
	}

	namespace, err := client.KubeClient().CoreV1().Namespaces().Get(ctx, m.AdmissionReview.Request.Namespace, metav1.GetOptions{}) //nolint:lll
	if err != nil {
		return nil, errors.Wrap(err, "namespace not found")
	}

	return namespace, nil
}

func (m *MutateInput) GetType() string {
	resourceType := m.AdmissionReview.Request.Resource.Resource

	if len(m.AdmissionReview.Request.Resource.Group) > 0 {
		resourceType += "." + m.AdmissionReview.Request.Resource.Group
	}

	if len(m.AdmissionReview.Request.Resource.Version) > 0 {
		resourceType += "." + m.AdmissionReview.Request.Resource.Version
	}

	return resourceType
}

type Mutation struct{}

func NewMutation() *Mutation {
	return &Mutation{}
}

func (m *Mutation) Mutate(ctx context.Context, input *MutateInput) *admissionv1.AdmissionResponse {
	log.Infof("mutate %s", input.GetObjectName())

	switch input.GetType() {
	case "pods.v1":
		return m.mutatePod(ctx, input)
	case "namespaces.v1":
		return m.mutateNamespace(ctx, input)
	case "ingresses.networking.k8s.io.v1":
		return m.mutateIngress(ctx, input)
	}

	return m.mutateError(string(input.AdmissionReview.Request.UID), errors.Errorf("unknown resource type %s", input.GetType())) //nolint:lll
}

func (m *Mutation) createSecret(ctx context.Context, namespace string, secret *types.CreateSecret) error {
	_, err := client.KubeClient().CoreV1().Secrets(namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	// return error if operation have some errors except not found
	if err != nil && !k8sErrors.IsNotFound(err) {
		return errors.Wrap(err, "error getting secret")
	}

	// delete secret if exists
	if !k8sErrors.IsNotFound(err) {
		err := client.KubeClient().CoreV1().Secrets(namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrap(err, "error deleting secret")
		}
	}

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secret.Name,
			Labels: map[string]string{
				"app": "pod-admission-controller",
			},
		},
		Type: corev1.SecretType(secret.Type),
		Data: secret.Data,
	}

	_, err = client.KubeClient().CoreV1().Secrets(namespace).Create(ctx, newSecret, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "error creating secret")
	}

	return nil
}

func (m *Mutation) replaceIngressHost(ingress networkingv1.Ingress, host string) (string, bool) {
	ingressSuffix := *config.Get().IngressSuffix

	if ingress.Annotations != nil {
		if suffix, ok := ingress.Annotations[types.AnnotationDefaultIngressSuffix]; ok {
			ingressSuffix = suffix
		}
	}

	if len(ingressSuffix) == 0 {
		return host, false
	}

	if strings.HasSuffix(host, ".") {
		return host + ingressSuffix, true
	}

	return host, false
}

func (m *Mutation) mutateIngressHosts(ingress networkingv1.Ingress) []types.PatchOperation {
	mutationPatch := make([]types.PatchOperation, 0)

	for ruleID, rule := range ingress.Spec.Rules {
		if host, ok := m.replaceIngressHost(ingress, rule.Host); ok {
			mutationPatch = append(mutationPatch, types.PatchOperation{
				Op:    "replace",
				Path:  fmt.Sprintf("/spec/rules/%d/host", ruleID),
				Value: host,
			})
		}
	}

	for tlsID, tls := range ingress.Spec.TLS {
		for hostID, host := range tls.Hosts {
			if newHost, ok := m.replaceIngressHost(ingress, host); ok {
				mutationPatch = append(mutationPatch, types.PatchOperation{
					Op:    "replace",
					Path:  fmt.Sprintf("/spec/tls/%d/hosts/%d", tlsID, hostID),
					Value: newHost,
				})
			}
		}
	}

	return mutationPatch
}

func (m *Mutation) mutateIngress(_ context.Context, input *MutateInput) *admissionv1.AdmissionResponse {
	req := input.AdmissionReview.Request

	ingress := networkingv1.Ingress{}

	if err := json.Unmarshal(req.Object.Raw, &ingress); err != nil {
		return m.mutateError(ingress.Name, err)
	}

	if m.checkIgnoreAnnotation(ingress.Annotations) {
		metrics.MutationsIgnored.WithLabelValues(ingress.Name).Inc()

		return &admissionv1.AdmissionResponse{
			Allowed: true,
			Warnings: []string{
				fmt.Sprintf("%s, ingress %s", types.WarningObjectDoedNotNeedMutation, ingress.Name),
			},
		}
	}

	mutationPatch := make([]types.PatchOperation, 0)

	mutationPatch = append(mutationPatch, m.mutateIngressHosts(ingress)...)
	mutationPatch = append(mutationPatch, m.injectAnnotation(ingress.Annotations))

	patchBytes, err := json.Marshal(mutationPatch)
	if err != nil {
		return m.mutateError(ingress.Name, err)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Result: &metav1.Status{
			Status: metav1.StatusSuccess,
		},
		Patch: patchBytes,
		PatchType: func() *admissionv1.PatchType {
			return utils.Pnt(admissionv1.PatchTypeJSONPatch)
		}(),
	}
}

const waitForNamespaceCreation = 10 * time.Second

func (m *Mutation) mutateNamespace(ctx context.Context, input *MutateInput) *admissionv1.AdmissionResponse { //nolint:lll,funlen
	req := input.AdmissionReview.Request

	namespace := corev1.Namespace{}

	if err := json.Unmarshal(req.Object.Raw, &namespace); err != nil {
		return m.mutateError(namespace.Name, err)
	}

	if m.checkIgnoreAnnotation(namespace.Annotations) {
		metrics.MutationsIgnored.WithLabelValues(namespace.Name).Inc()

		return &admissionv1.AdmissionResponse{
			Allowed: true,
			Warnings: []string{
				fmt.Sprintf("%s, namespace %s", types.WarningObjectDoedNotNeedMutation, namespace.Name),
			},
		}
	}

	createSecrets := func(ctx context.Context) {
		for _, secret := range config.Get().CreateSecrets {
			if err := m.createSecret(ctx, namespace.Name, secret); err != nil {
				log.WithError(err).Errorf("Error creating secret %s/%s", namespace.Name, secret.Name)
			}
		}
	}

	switch req.Operation { //nolint:exhaustive
	case admissionv1.Create:
		// for create operation we need to create secrets after namespace will be created
		// use goroutine to not block main thread
		go func() { //nolint:contextcheck
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			log.Infof("Scheduled secrets creation %s...", waitForNamespaceCreation)
			utils.SleepContext(ctx, waitForNamespaceCreation)

			createSecrets(ctx)
		}()
	case admissionv1.Update:
		createSecrets(ctx)
	}

	mutationPatch := []types.PatchOperation{
		m.injectAnnotation(namespace.Annotations),
		m.injectLabels(namespace.Labels),
	}

	patchBytes, err := json.Marshal(mutationPatch)
	if err != nil {
		return m.mutateError(namespace.Name, err)
	}

	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Result: &metav1.Status{
			Status: metav1.StatusSuccess,
		},
		Patch: patchBytes,
		PatchType: func() *admissionv1.PatchType {
			return utils.Pnt(admissionv1.PatchTypeJSONPatch)
		}(),
	}
}

// mutate pod.
func (m *Mutation) mutatePod(ctx context.Context, input *MutateInput) *admissionv1.AdmissionResponse { //nolint:funlen,cyclop,lll
	namespace, err := input.GetNamespace(ctx)
	if err != nil {
		return m.mutateError("namespace not found", err)
	}

	req := input.AdmissionReview.Request

	pod := corev1.Pod{}

	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		return m.mutateError(namespace.Name, err)
	}

	if m.checkIgnoreAnnotation(pod.Annotations) {
		metrics.MutationsIgnored.WithLabelValues(namespace.Name).Inc()

		return &admissionv1.AdmissionResponse{
			Allowed: true,
			Warnings: []string{
				fmt.Sprintf("%s, pod %s/%s", types.WarningObjectDoedNotNeedMutation, namespace.Name, pod.Name),
			},
		}
	}

	mutationPatch := make([]types.PatchOperation, 0)

	var ownerKind, ownerName string
	if len(pod.OwnerReferences) > 0 {
		ownerKind = pod.OwnerReferences[0].Kind
		ownerName = pod.OwnerReferences[0].Name
	}

	for _, podContainer := range types.PodContainersFromPod(namespace, &pod) {
		containerInfo := &types.ContainerInfo{
			OwnerKind:            ownerKind,
			OwnerName:            ownerName,
			PodContainer:         podContainer,
			ContainerName:        podContainer.Container.Name,
			ContainerType:        podContainer.Type,
			Namespace:            namespace.Name,
			NamespaceAnnotations: namespace.Annotations,
			NamespaceLabels:      namespace.Labels,
			PodAnnotations:       pod.Annotations,
			PodLabels:            pod.Labels,
			SelectedRules:        []*types.Rule{},
		}

		imageInfo, err := GetImageInfo(podContainer.Container.Image)
		if err != nil {
			return m.mutateError(namespace.Name, err)
		}

		containerInfo.Image = imageInfo

		log.Debugf("containerInfo.Image=%+v", containerInfo.Image)

		// check rule that corresponds to container
		for _, rule := range config.Get().Rules {
			match, err := conditions.Check(containerInfo, rule.Conditions)
			if err != nil {
				return m.mutateError(namespace.Name, err)
			}

			if match {
				containerInfo.SelectedRules = append(containerInfo.SelectedRules, rule)
			}
		}

		// if no rules found for container continue to next container
		if len(containerInfo.SelectedRules) == 0 {
			continue
		}

		pathOps, err := patch.NewPatch(ctx, containerInfo)
		if err != nil {
			return m.mutateError(namespace.Name, err)
		}

		for _, pathOp := range pathOps {
			if m.patchContains(mutationPatch, pathOp) {
				log.Debugf("patch already exists: %s", pathOp)
			} else {
				mutationPatch = append(mutationPatch, pathOp)
			}
		}
	}

	// if no patches found return empty response
	if len(mutationPatch) == 0 {
		return &admissionv1.AdmissionResponse{
			Allowed:  true,
			Warnings: []string{types.WarningNoPatchGenerated},
		}
	}

	mutationPatch = append(mutationPatch, m.injectAnnotation(pod.Annotations))

	patchBytes, err := json.Marshal(mutationPatch)
	if err != nil {
		return m.mutateError(namespace.Name, err)
	}

	log.Debugf("patch=%s", string(patchBytes))

	metrics.MutationsTotal.WithLabelValues(namespace.Name).Inc()

	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Result: &metav1.Status{
			Status: metav1.StatusSuccess,
		},
		Patch: patchBytes,
		PatchType: func() *admissionv1.PatchType {
			return utils.Pnt(admissionv1.PatchTypeJSONPatch)
		}(),
	}
}

func (m *Mutation) patchContains(patches []types.PatchOperation, patch types.PatchOperation) bool {
	for _, p := range patches {
		if p.Path == patch.Path && p.Op == patch.Op {
			if reflect.DeepEqual(p.Value, patch.Value) {
				return true
			}
		}
	}

	return false
}

// some objects does not need mutation
// pod-admission-controller/ignore=true.
func (m *Mutation) checkIgnoreAnnotation(annotations map[string]string) bool {
	if ignore, ok := annotations[types.AnnotationIgnore]; ok {
		if strings.EqualFold(ignore, "true") {
			return true
		}
	}

	return false
}

func (m *Mutation) injectAnnotation(annotations map[string]string) types.PatchOperation {
	objAnnotations := make(map[string]string)
	if annotations != nil {
		objAnnotations = annotations
	}

	objAnnotations[types.AnnotationInjected] = "true"

	return types.PatchOperation{
		Op:    "add",
		Path:  "/metadata/annotations",
		Value: objAnnotations,
	}
}

func (m *Mutation) injectLabels(labels map[string]string) types.PatchOperation {
	objLabels := make(map[string]string)
	if labels != nil {
		objLabels = labels
	}

	objLabels[types.LabelManaged] = "true"

	return types.PatchOperation{
		Op:    "add",
		Path:  "/metadata/labels",
		Value: objLabels,
	}
}

// throw mutaion errors.
func (m *Mutation) mutateError(namespaceName string, err error) *admissionv1.AdmissionResponse {
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

	input := &MutateInput{
		AdmissionReview: requestedAdmissionReview,
	}

	responseAdmissionReview := &admissionv1.AdmissionReview{}
	responseAdmissionReview.SetGroupVersionKind(*gvk)
	responseAdmissionReview.Response = NewMutation().Mutate(ctx, input)
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
	responseObj = responseAdmissionReview

	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		return nil, errors.Wrap(err, "can not marshal response")
	}

	return respBytes, nil
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
		Domain: reference.Domain(refName),
		Path:   reference.Path(refName),
		Name:   image,
		Slug:   strings.Trim(imageName, "-"),
		Tag:    "latest",
	}

	if tag, ok := refName.(reference.Tagged); ok {
		result.Tag = tag.Tag()
	}

	return &result, nil
}

func TestPOD(ctx context.Context, namespace, podName string) ([]byte, error) {
	pod, err := client.KubeClient().CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "Could not get namespace")
	}

	podJSON, err := json.Marshal(pod)
	if err != nil {
		return nil, errors.Wrap(err, "Could not marshal pod")
	}

	input := &MutateInput{
		Namespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		},
		AdmissionReview: &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Resource: metav1.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				Object: runtime.RawExtension{
					Raw: podJSON,
				},
			},
		},
	}

	log.Infof("req=%+v", input)

	resp := NewMutation().Mutate(ctx, input)

	log.Infof("warnings=%+v", resp.Warnings)

	return resp.Patch, nil
}
