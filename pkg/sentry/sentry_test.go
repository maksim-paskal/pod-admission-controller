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
package sentry_test

import (
	"flag"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/sentry"
)

var ts = httptest.NewServer(GetHandler())

func GetHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/0/projects/", projects)
	mux.HandleFunc("/api/0/projects/test-org/test-project/keys/", projectkey)

	return mux
}

func projects(w http.ResponseWriter, r *http.Request) {
	projectsByte, err := ioutil.ReadFile("testdata/projects.json")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	_, _ = w.Write(projectsByte)
}

func projectkey(w http.ResponseWriter, r *http.Request) {
	projectsByte, err := ioutil.ReadFile("testdata/projectkey.json")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	_, _ = w.Write(projectsByte)
}

func TestAPI(t *testing.T) {
	t.Parallel()

	if err := flag.Set("sentry.endpoint", ts.URL); err != nil {
		t.Fatal(err)
	}

	projects, err := sentry.GetProjects()
	if err != nil {
		t.Fatal(err)
	}

	if len(projects) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(projects))
	}

	project := sentry.Project{
		Organization: sentry.Organization{
			Slug: "test-org",
		},
		Slug: "test-project",
	}

	projectkey, err := sentry.GetProjectKeys(project)
	if err != nil {
		t.Fatal(err)
	}

	if len(projectkey) != 1 {
		t.Fatalf("expected 1 project key, got %d", len(projectkey))
	}
}
