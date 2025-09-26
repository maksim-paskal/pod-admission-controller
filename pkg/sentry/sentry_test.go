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
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/maksim-paskal/pod-admission-controller/pkg/sentry"
	log "github.com/sirupsen/logrus"
)

var ts = httptest.NewServer(GetHandler())

func GetHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/0/projects/", projects)
	mux.HandleFunc("/api/0/projects/test-org/test-project/keys/", projectkey)
	mux.HandleFunc("/api/0/projects/the-interstellar-jurisdiction/the-spoiled-yoghurt/keys/", projectkey)
	mux.HandleFunc("/api/0/projects/the-interstellar-jurisdiction/prime-mover/keys/", projectkey)
	mux.HandleFunc("/api/0/projects/the-interstellar-jurisdiction/pump-station/keys/", projectkey)

	return mux
}

func projects(w http.ResponseWriter, _ *http.Request) {
	projectsByte, err := os.ReadFile("testdata/projects.json")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	_, _ = w.Write(projectsByte)
}

func projectkey(w http.ResponseWriter, r *http.Request) {
	projectsByte, err := os.ReadFile("testdata/projectkey.json")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	projectsByte = bytes.ReplaceAll(projectsByte,
		[]byte("https://cec9dfceb0b74c1c9a5e3c135585f364@sentry.io/2"),
		[]byte("https://"+strings.Split(r.URL.Path, "/")[5]+"@sentry.io/2"),
	)

	_, _ = w.Write(projectsByte)
}

func Test(t *testing.T) { //nolint:funlen,cyclop
	t.Parallel()

	ctx := t.Context()

	log.SetLevel(log.DebugLevel)

	params := config.Params{
		Sentry: &config.Sentry{
			Endpoint:     ts.URL,
			Relay:        "http://localhost:3000",
			Token:        "test-token",
			Organization: "test-org",
			Prefixes: []*config.SentryPrefix{
				{
					Pattern: "^prime-.+$",
					Name:    "prime-",
				},
			},
			Projects: map[string]string{
				"the-spoiled-yoghurt": "some/test/image",
				"mover":               "prime/mover/image",
				"pump-station":        "pump/station/image",
			},
		},
	}

	config.Set(params)

	if err := sentry.CreateCache(ctx); err != nil {
		t.Fatal(err)
	}

	t.Run("TestGetProjects", func(t *testing.T) {
		t.Parallel()

		projects, err := sentry.GetProjects(ctx)
		if err != nil {
			t.Fatal(err)
		}

		if expected := 3; len(projects) != expected {
			t.Fatalf("expected %d projects, got %d", expected, len(projects))
		}
	})

	t.Run("TestGetProjectKeys", func(t *testing.T) {
		t.Parallel()

		project := sentry.Project{
			Organization: sentry.Organization{
				Slug: "test-org",
			},
			Slug: "test-project",
		}

		projectkey, err := sentry.GetProjectKeys(ctx, project)
		if err != nil {
			t.Fatal(err)
		}

		if expected := 1; len(projectkey) != expected {
			t.Fatalf("expected %d project key, got %d", expected, len(projectkey))
		}

		if expected := "http://test-project@localhost:3000/2"; projectkey[0].Dsn.Public != expected {
			t.Fatalf("expected %s project key, got %s", expected, projectkey[0].Dsn.Public)
		}
	})

	t.Run("GetSentryDSN", func(t *testing.T) {
		t.Parallel()

		type test struct {
			imagePath string
			dsn       string
			namespace string
		}

		tests := []test{
			{
				imagePath: "docker.io/some/test/image:latest",
				dsn:       "",
			},
			{
				imagePath: "some/test/image",
				dsn:       "http://the-spoiled-yoghurt@localhost:3000/2",
			},
			{
				imagePath: "some/test/image/test",
				dsn:       "http://the-spoiled-yoghurt@localhost:3000/2",
			},
			{
				imagePath: "pump/station/image",
				dsn:       "http://pump-station@localhost:3000/2",
			},
			{
				imagePath: "prime/mover/image",
				dsn:       "http://prime-mover@localhost:3000/2",
				namespace: "prime-test",
			},
			{
				imagePath: "/some/test/fimage",
				dsn:       "",
			},
			{
				imagePath: "/fake",
				dsn:       "",
			},
		}

		for _, test := range tests {
			t.Run(test.imagePath, func(t *testing.T) {
				dsn, ok := sentry.GetSentryDSN(test.namespace, test.imagePath)
				t.Log(dsn, ok)
				// if dsn is empty, ok should be true
				if ok != (len(test.dsn) != 0) {
					t.Fatalf("expected %t, got %t", len(test.dsn) != 0, ok)
				}

				if len(test.dsn) != 0 && dsn != test.dsn {
					t.Fatalf("expected %s dsn, got %s", test.dsn, dsn)
				}
			})
		}
	})

	t.Run("TestGetRelayDSN", func(t *testing.T) {
		t.Parallel()

		key := sentry.Key{
			Dsn: sentry.KeyDsn{
				Public: "https://qwerty@sentry.io/2",
			},
		}

		if expected := "http://qwerty@localhost:3000/2"; key.GetRelayDSN() != expected {
			t.Fatalf("expected %s dsn, got %s", expected, key.GetRelayDSN())
		}
	})
}
