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
package types

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const (
	annotationPrefix = "pod-admission-controller"
	// annotation that will added to pod if mutation executes.
	AnnotationInjected = annotationPrefix + "/injected"
	// skip mutation.
	AnnotationIgnore = annotationPrefix + "/ignore"
	// list of containers that should be skipped from RunAsNonRoot.
	AnnotationIgnoreEnv = annotationPrefix + "/ignoreEnv"
	// Deprecated: list of containers that should be skipped from RunAsNonRoot.
	AnnotationIgnoreRunAsNonRoot = annotationPrefix + "/ignoreRunAsNonRoot"
	// Deprecated: list of containers that should be skipped from AddDefaultResources.
	AnnotationIgnoreAddDefaultResources = annotationPrefix + "/ignoreAddDefaultResources"
	// Default CPU requests.
	AnnotationDefaultResourcesCPU = annotationPrefix + "/defaultResourcesCPU"
	// Default Memory requests.
	AnnotationDefaultResourcesMemory = annotationPrefix + "/defaultResourcesMemory"
	// warning when AnnotationIgnore is enabled.
	WarningObjectDoedNotNeedMutation = annotationPrefix + ": ignore mutation by annotation " + AnnotationIgnore
	// warning when no patch is generated.
	WarningNoPatchGenerated = annotationPrefix + ". No patches found"
)

type RunAsNonRootReplaceUser struct {
	Enabled  bool
	FromUser int64
	ToUser   int64
}

type RunAsNonRoot struct {
	Enabled bool
	// replace RunAsUser in container
	ReplaceUser RunAsNonRootReplaceUser
}

type ReplaceContainerImageHost struct {
	Enabled bool
	From    string
	To      string
}

type AddDefaultResources struct {
	Enabled  bool
	LimitCPU bool
	// Deprecated: use custompatch instead
	RemoveResources bool
}

type Rule struct {
	Debug                     bool
	Name                      string
	EnvReplace                []corev1.EnvVar
	Env                       []corev1.EnvVar
	Conditions                []Condition
	AddDefaultResources       AddDefaultResources
	RunAsNonRoot              RunAsNonRoot
	ReplaceContainerImageHost ReplaceContainerImageHost
	Tolerations               []corev1.Toleration
	ImagePullSecrets          []corev1.LocalObjectReference
	CustomPatches             []PatchOperation
}

func (r *Rule) Logf(format string, args ...interface{}) {
	if r.Debug || log.IsLevelEnabled(log.DebugLevel) {
		log.WithFields(log.Fields{
			"name": r.Name,
		}).Infof(format, args...)
	}
}

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (p *PatchOperation) String() string {
	out, err := json.Marshal(p)
	if err != nil {
		return err.Error()
	}

	return string(out)
}

// must be lowercase.
type ConditionOperator string

func (op ConditionOperator) Value() ConditionOperator {
	return ConditionOperator(strings.ToLower(string(op)))
}

func (op ConditionOperator) Validate() error {
	if slices.Contains(validOperators, op) {
		return nil
	}

	return errors.Errorf("unknown operator %s", op)
}

func (op ConditionOperator) IsNegate() bool {
	return slices.Contains(negateOperators, op)
}

const (
	OperatorEqual     ConditionOperator = "equal"
	OperatorNotEqual  ConditionOperator = "notequal"
	OperatorRegexp    ConditionOperator = "regexp"
	OperatorNotRegexp ConditionOperator = "notregexp"
	OperatorIn        ConditionOperator = "in"
	OperatorNotIn     ConditionOperator = "notin"
	OperatorEmpty     ConditionOperator = "empty"
	OperatorNotEmpty  ConditionOperator = "notempty"
)

var negateOperators = []ConditionOperator{
	OperatorNotEqual,
	OperatorNotRegexp,
	OperatorNotIn,
	OperatorNotEmpty,
}

var validOperators = []ConditionOperator{
	OperatorEqual,
	OperatorNotEqual,
	OperatorRegexp,
	OperatorNotRegexp,
	OperatorIn,
	OperatorNotIn,
	OperatorEmpty,
	OperatorNotEmpty,
}

type Condition struct {
	Key      string
	Operator ConditionOperator
	Value    string
	Values   []string
}

type ContainerImage struct {
	Domain string
	Name   string
	Slug   string
	Tag    string
}

type ContainerInfo struct {
	PodContainer         *PodContainer
	ContainerName        string
	ContainerType        PodContainerType
	Namespace            string
	NamespaceAnnotations map[string]string
	NamespaceLabels      map[string]string
	Image                *ContainerImage
	PodAnnotations       map[string]string
	PodLabels            map[string]string
	SelectedRules        []*Rule
}

// return JSON representation of the container info.
func (c *ContainerInfo) String() string {
	out, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}

	return string(out)
}

// return namespaced pod annotation value.
func (c *ContainerInfo) GetPodAnnotation(key string) (string, bool) {
	if val, ok := c.PodAnnotations[key]; ok {
		return val, true
	}

	if val, ok := c.NamespaceAnnotations[key]; ok {
		return val, true
	}

	return "", false
}

func (c *ContainerInfo) GetSelectedRulesEnv() []corev1.EnvVar {
	containerEnv := make([]corev1.EnvVar, 0)

	for _, selectedRule := range c.SelectedRules {
		containerEnv = append(containerEnv, selectedRule.Env...)
	}

	return containerEnv
}

func (c *ContainerInfo) GetSelectedRulesEnvReplace() []corev1.EnvVar {
	containerEnv := make([]corev1.EnvVar, 0)

	for _, selectedRule := range c.SelectedRules {
		containerEnv = append(containerEnv, selectedRule.EnvReplace...)
	}

	return containerEnv
}

type PodContainerType string

const (
	PodContainerTypeInitContainer PodContainerType = "initContainer"
	PodContainerTypeContainer     PodContainerType = "container"
)

type PodContainer struct {
	Pod       *corev1.Pod
	Namespace *corev1.Namespace
	Order     int
	Type      PodContainerType
	Container *corev1.Container
}

func (c *PodContainer) String() string {
	out, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}

	return string(out)
}

// return owner kind of the pod.
func (c *PodContainer) OwnerKind() string {
	if len(c.Pod.OwnerReferences) > 0 {
		return c.Pod.OwnerReferences[0].Kind
	}

	return ""
}

func (c *PodContainer) ContainerPath() string {
	return fmt.Sprintf("/spec/%ss/%d", c.Type, c.Order)
}

func PodContainersFromPod(namespace *corev1.Namespace, pod *corev1.Pod) []*PodContainer {
	podContainers := make([]*PodContainer, 0)

	for order := range pod.Spec.InitContainers {
		podContainers = append(podContainers, &PodContainer{
			Pod:       pod,
			Namespace: namespace,
			Order:     order,
			Type:      PodContainerTypeInitContainer,
			Container: &pod.Spec.InitContainers[order],
		})
	}

	for order := range pod.Spec.Containers {
		podContainers = append(podContainers, &PodContainer{
			Pod:       pod,
			Namespace: namespace,
			Order:     order,
			Type:      PodContainerTypeContainer,
			Container: &pod.Spec.Containers[order],
		})
	}

	return podContainers
}

type CreateSecret struct {
	Name string
	Type string
	Data map[string][]byte
}
