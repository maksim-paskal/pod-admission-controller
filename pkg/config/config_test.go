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
package config_test

import (
	"flag"
	"reflect"
	"testing"
	"time"

	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	if err := flag.Set("config", "testdata/test-config.yaml"); err != nil {
		t.Fatal(err)
	}

	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	t.Log("config:", config.Get().String())

	if *config.Get().CertFile != "1" {
		t.Fatal("not valid SentryEndpoint")
	}

	if *config.Get().KeyFile != "2" {
		t.Fatal("not valid SentryToken")
	}

	if err := config.Validate(); err != nil {
		t.Fatal("not valid config")
	}

	if period := config.Get().GetGracePeriod(); period != 5*time.Second {
		t.Fatalf("not valid grace period %s", period)
	}

	// not valid
	if err := flag.Set("config", "testdata/not-valid-config.yaml"); err != nil {
		t.Fatal(err)
	}

	if err := config.Load(); err != nil {
		t.Fatal("config not loaded")
	}

	if err := config.Validate(); err == nil {
		t.Fatal("must be not valid config")
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	if config.GetVersion() != "dev" {
		t.Fatal("version is not dev")
	}
}

func TestSentryPrefix(t *testing.T) {
	param := &config.Params{
		Sentry: &config.Sentry{
			Prefixes: []*config.SentryPrefix{
				{
					Pattern: "^test$",
					Name:    "testname",
				},
			},
		},
	}

	if prefix := param.Sentry.GetPrefixes("aa"); len(prefix) != 0 {
		t.Fatal("not valid prefix aa")
	}

	if prefix := param.Sentry.GetPrefixes("test"); len(prefix) != 1 || prefix[0] != "testname" {
		t.Fatal("not valid prefix test")
	}

	t.Setenv("SENTRY_PROJECTS_PREFIX", "test2,test3")

	if prefix := param.Sentry.GetPrefixes("test"); !reflect.DeepEqual(prefix, []string{"testname", "test2", "test3"}) {
		t.Fatal("not valid prefix test2, not valid: ", prefix)
	}
}
