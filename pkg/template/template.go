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
package template

import (
	"bytes"
	"net"
	"regexp"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/maksim-paskal/pod-admission-controller/pkg/sentry"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/pkg/errors"
)

func Get(containerInfo *types.ContainerInfo, value string) (string, error) {
	tmpl, err := template.New("tmpl").Option("missingkey=zero").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		// regexp string by pattern
		"regexp": func(pattern string, value string) []string {
			return regexp.MustCompile(pattern).FindStringSubmatch(value)
		},
		// return unknown if part is out of slice range
		"indexUnknown": func(slice []string, part int) string {
			if part >= len(slice) {
				return "unknown"
			}

			return slice[part]
		},
		// return sentry DSN based on image name
		"GetSentryDSN": func(projectSlug string) string {
			if dsn, ok := sentry.GetCache()[projectSlug]; ok {
				return dsn
			}

			return ""
		},
		"Resolve": func(domain string) string {
			ip, err := net.ResolveIPAddr("ip", domain)
			if err != nil {
				return "error: " + err.Error()
			}

			return ip.String()
		},
		"ResolveFallback": func(domain, fallback string) string {
			ip, err := net.ResolveIPAddr("ip", domain)
			if err != nil {
				return fallback
			}

			return ip.String()
		},
	}).Parse(value)
	if err != nil {
		return "", errors.Wrap(err, "error parsing template")
	}

	var tpl bytes.Buffer

	err = tmpl.Execute(&tpl, containerInfo)
	if err != nil {
		return "", errors.Wrap(err, "error executing template")
	}

	return tpl.String(), nil
}
