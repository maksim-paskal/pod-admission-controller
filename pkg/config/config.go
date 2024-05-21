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
	"regexp"
	"strings"
	"time"

	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	defaultGracePeriod = 5
	defaultAddr        = ":8443"
	defaultMetricsAddr = ":31080"
)

type SentryPrefix struct {
	Pattern string
	Name    string
}

type Sentry struct {
	Endpoint     string
	Token        string
	Organization string
	Prefixes     []*SentryPrefix
	// replace sentry DSN with replay
	Relay string
	// project slug -> image path
	Projects map[string]string
	Cache    map[string]string
}

func (s *Sentry) GetPrefixes(name string) []string {
	result := make([]string, 0)

	for _, prefix := range s.Prefixes {
		if prefix.Pattern == "" {
			continue
		}

		re2, err := regexp.Compile(prefix.Pattern)
		if err != nil {
			log.Warnf("error in %s regexp.Compile: %v", prefix.Pattern, err)

			continue
		}

		if re2.MatchString(name) {
			result = append(result, prefix.Name)
		}
	}

	if prefix := os.Getenv("SENTRY_PROJECTS_PREFIX"); len(prefix) > 0 {
		result = append(result, strings.Split(prefix, ",")...)
	}

	return result
}

type Params struct {
	GracePeriodSeconds *int
	ConfigFile         *string
	KubeConfigFile     *string
	LogLevel           *string
	LogPretty          *bool
	Addr               *string
	MetricsAddr        *string
	CertFile           *string
	KeyFile            *string
	Rules              []*types.Rule
	Sentry             *Sentry
	SentryDSN          *string
	CreateSecrets      []*types.CreateSecret
	IngressSuffix      *string
}

var param = Params{
	GracePeriodSeconds: flag.Int("graceperiod", defaultGracePeriod, "grace period"),
	ConfigFile:         flag.String("config", "", "config file"),
	KubeConfigFile:     flag.String("kubeconfig", "", "kubeconfig file"),
	LogLevel:           flag.String("log.level", "INFO", "log level"),
	LogPretty:          flag.Bool("log.pretty", false, "print log in pretty format"),
	Addr:               flag.String("listen", defaultAddr, "address to listen on"),
	MetricsAddr:        flag.String("metrics.listen", defaultMetricsAddr, "address to listen on metrics"),
	CertFile:           flag.String("cert", "server.crt", "certificate file"),
	KeyFile:            flag.String("key", "server.key", "key file"),
	SentryDSN:          flag.String("sentry.dsn", os.Getenv("SENTRY_DSN"), "sentry DSN for error reporting"),
	IngressSuffix:      flag.String("ingress.suffix", os.Getenv("INGRESS_SUFFIX"), "default ingress suffix"),
}

func (p *Params) GetGracePeriod() time.Duration {
	return time.Duration(*p.GracePeriodSeconds) * time.Second
}

func Get() *Params {
	return &param
}

func Set(config Params) {
	param = config
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

	for ruleID, rule := range param.Rules {
		for conditionID, condition := range rule.Conditions {
			// normalize operator
			param.Rules[ruleID].Conditions[conditionID].Operator = condition.Operator.Value()
		}
	}

	return nil
}

func Validate() error {
	for _, rule := range param.Rules {
		for _, condition := range rule.Conditions {
			if err := condition.Validate(); err != nil {
				return errors.Wrap(err, "error in validating condition")
			}

			if err := condition.Operator.Validate(); err != nil {
				return errors.Wrap(err, "error in validating operator")
			}
		}
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
