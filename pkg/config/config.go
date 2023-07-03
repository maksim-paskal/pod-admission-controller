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
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	defaultGracePeriod             = 5
	defaultAddr                    = ":8443"
	defaultMetricsAddr             = ":31080"
	defaultContainerResourceCPU    = "100m"
	defaultContainerResourceMemory = "500Mi"
)

type Params struct {
	GracePeriodSeconds   *int
	ConfigFile           *string
	KubeConfigFile       *string
	LogLevel             *string
	LogPretty            *bool
	Addr                 *string
	MetricsAddr          *string
	CertFile             *string
	KeyFile              *string
	Rules                []*types.Rule
	DefaultRequestCPU    *string
	DefaultRequestMemory *string
	SentryEndpoint       *string
	SentryToken          *string
	SentryDSN            *string
}

var param = Params{
	GracePeriodSeconds:   flag.Int("graceperiod", defaultGracePeriod, "grace period"),
	ConfigFile:           flag.String("config", "", "config file"),
	KubeConfigFile:       flag.String("kubeconfig", "", "kubeconfig file"),
	LogLevel:             flag.String("log.level", "INFO", "log level"),
	LogPretty:            flag.Bool("log.pretty", false, "print log in pretty format"),
	Addr:                 flag.String("listen", defaultAddr, "address to listen on"),
	MetricsAddr:          flag.String("metrics.listen", defaultMetricsAddr, "address to listen on metrics"),
	CertFile:             flag.String("cert", "server.crt", "certificate file"),
	KeyFile:              flag.String("key", "server.key", "key file"),
	DefaultRequestCPU:    flag.String("adddefaultresources.cpu", defaultContainerResourceCPU, "default cpu resources"),
	DefaultRequestMemory: flag.String("addefaultresources.memory", defaultContainerResourceMemory, "default memory resources"), //nolint:lll
	SentryEndpoint:       flag.String("sentry.endpoint", "", "sentry endpoint"),
	SentryToken:          flag.String("sentry.token", "", "sentry token"),
	SentryDSN:            flag.String("sentry.dsn", os.Getenv("SENTRY_DSN"), "sentry DSN for error reporting"),
}

func (p *Params) GetGracePeriod() time.Duration {
	return time.Duration(*p.GracePeriodSeconds) * time.Second
}

func Get() *Params {
	return &param
}

func Load() error {
	if len(*param.ConfigFile) == 0 {
		return nil
	}

	configByte, err := os.ReadFile(*param.ConfigFile)
	if err != nil {
		return errors.Wrap(err, "error in os.ReadFile")
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
	out, err := json.Marshal(p)
	if err != nil {
		return err.Error()
	}

	return string(out)
}

var gitVersion = "dev"

func GetVersion() string {
	return gitVersion
}
