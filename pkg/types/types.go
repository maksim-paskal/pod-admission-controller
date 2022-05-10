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
	// list of containers that should be skipped from RunAsNonRoot.
	AnnotationIgnoreRunAsNonRoot = annotationPrefix + "/ignoreRunAsNonRoot"
	// list of containers that should be skipped from AddDefaultResources.
	AnnotationIgnoreAddDefaultResources = annotationPrefix + "/ignoreAddDefaultResources"
	// warning when AnnotationIgnore is enabled.
	WarningPodDoedNotNeedMutation = annotationPrefix + ". POD does not need mutation"
	// warning when no patch is generated.
	WarningNoPatchGenerated = annotationPrefix + ". No patches found for pod"
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

type AddDefaultResources struct {
	Enabled  bool
	LimitCPU bool
}

type Rule struct {
	Name                string
	Env                 []corev1.EnvVar
	Conditions          []Conditions
	AddDefaultResources AddDefaultResources
	RunAsNonRoot        RunAsNonRoot
}

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type Conditions struct {
	Key      string
	Operator string
	Value    string
}

type ContainerInfo struct {
	ContainerName        string
	Namespace            string
	NamespaceAnnotations map[string]string
	NamespaceLabels      map[string]string
	Image                string
	PodAnnotations       map[string]string
	PodLabels            map[string]string
	SelectedRules        []Rule
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

	return "", false
}

func (c *ContainerInfo) GetSelectedRulesEnv() []corev1.EnvVar {
	containerEnv := make([]corev1.EnvVar, 0)

	for _, selectedRule := range c.SelectedRules {
		containerEnv = append(containerEnv, selectedRule.Env...)
	}

	return containerEnv
}

type SelectedRuleType string

const (
	SelectedRuleRunAsNonRoot        = SelectedRuleType("RunAsNonRoot")
	SelectedRuleAddDefaultResources = SelectedRuleType("AddDefaultResources")
)

func (c *ContainerInfo) GetSelectedRuleEnabled(ruleType SelectedRuleType) (Rule, bool) {
	for _, selectedRule := range c.SelectedRules {
		switch ruleType {
		case SelectedRuleRunAsNonRoot:
			if selectedRule.RunAsNonRoot.Enabled {
				return selectedRule, true
			}
		case SelectedRuleAddDefaultResources:
			if selectedRule.AddDefaultResources.Enabled {
				return selectedRule, true
			}
		}
	}

	return Rule{}, false
}
