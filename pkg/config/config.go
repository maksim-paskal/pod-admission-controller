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
package config

import (
	"flag"
	"io/ioutil"

	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	defaultPort                    = 8443
	defaultContainerResourceCPU    = "100m"
	defaultContainerResourceMemory = "500Mi"
)

type Params struct {
	ConfigFile           *string
	KubeConfigFile       *string
	LogLevel             *string
	LogPretty            *bool
	Port                 *int
	CertFile             *string
	KeyFile              *string
	Rules                []types.Rule
	DefaultRequestCPU    *string
	DefaultRequestMemory *string
	SentryEndpoint       *string
	SentryToken          *string
}

var param = Params{
	ConfigFile:           flag.String("config", "", "config file"),
	KubeConfigFile:       flag.String("kubeconfig", "", "kubeconfig file"),
	LogLevel:             flag.String("log.level", "INFO", "log level"),
	LogPretty:            flag.Bool("log.pretty", false, "print log in pretty format"),
	Port:                 flag.Int("port", defaultPort, "port to listen on"),
	CertFile:             flag.String("cert", "server.crt", "certificate file"),
	KeyFile:              flag.String("key", "server.key", "key file"),
	DefaultRequestCPU:    flag.String("adddefaultresources.cpu", defaultContainerResourceCPU, "default cpu resources"),
	DefaultRequestMemory: flag.String("addefaultresources.memory", defaultContainerResourceMemory, "default memory resources"), //nolint:lll
	SentryEndpoint:       flag.String("sentry.endpoint", "", "sentry endpoint"),
	SentryToken:          flag.String("sentry.token", "", "sentry token"),
}

func Get() *Params {
	return &param
}

func Load() error {
	if len(*param.ConfigFile) == 0 {
		return nil
	}

	configByte, err := ioutil.ReadFile(*param.ConfigFile)
	if err != nil {
		return errors.Wrap(err, "error in ioutil.ReadFile")
	}

	err = yaml.Unmarshal(configByte, &param)
	if err != nil {
		return errors.Wrap(err, "error in yaml.Unmarshal")
	}

	return nil
}

func Check() error {
	if _, err := resource.ParseQuantity(*param.DefaultRequestCPU); err != nil {
		return errors.Wrapf(err, "not valid resources %s", *param.DefaultRequestCPU)
	}

	if _, err := resource.ParseQuantity(*param.DefaultRequestMemory); err != nil {
		return errors.Wrapf(err, "not valid resources %s", *param.DefaultRequestMemory)
	}

	return nil
}

func (p *Params) String() string {
	out, _ := yaml.Marshal(p)

	return string(out)
}

var gitVersion = "dev"

func GetVersion() string {
	return gitVersion
}
