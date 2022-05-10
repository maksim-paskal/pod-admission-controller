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
	"testing"

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

	if *config.Get().SentryEndpoint != "1" {
		t.Fatal("not valid SentryEndpoint")
	}

	if *config.Get().SentryToken != "2" {
		t.Fatal("not valid SentryToken")
	}

	if err := config.Check(); err != nil {
		t.Fatal("not valid config")
	}

	if err := flag.Set("adddefaultresources.cpu", "fake"); err != nil {
		t.Fatal(err)
	}

	// config check must fail
	if err := config.Check(); err == nil {
		t.Fatal("must fail config check adddefaultresources.cpu not valid")
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	if config.GetVersion() != "dev" {
		t.Fatal("version is not dev")
	}
}
