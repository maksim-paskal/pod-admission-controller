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
package sentry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const defaultHTTPTimeout = 5 * time.Second

var client = &http.Client{
	Timeout: defaultHTTPTimeout,
}

type KeyDsn struct {
	Public string
}

type Key struct {
	Dsn KeyDsn
}

type Organization struct {
	Slug string
}

type Project struct {
	Organization Organization
	Slug         string
}

var cache map[string]string

func GetCache() map[string]string {
	return cache
}

func CreateCache(ctx context.Context) error {
	if len(*config.Get().SentryEndpoint) == 0 || len(*config.Get().SentryToken) == 0 {
		return nil
	}

	log.Info("Creating Sentry projects cache")

	cache = make(map[string]string)

	projects, err := GetProjects(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get projects")
	}

	for _, project := range projects {
		if ctx.Err() != nil {
			return errors.Wrap(ctx.Err(), "context canceled")
		}

		keys, err := GetProjectKeys(ctx, project)
		if err != nil {
			return errors.Wrap(err, "failed to get keys")
		}

		log.Debugf("%s=%s", project.Slug, keys[0].Dsn.Public)

		cache[project.Slug] = keys[0].Dsn.Public
	}

	return nil
}

func GetProjects(ctx context.Context) ([]Project, error) {
	req, err := rawRequest(ctx, http.MethodGet, "/api/0/projects/")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make request")
	}

	if response.StatusCode != http.StatusOK {
		return nil, errors.New("result not OK")
	}

	defer response.Body.Close()

	projects := []Project{}

	err = json.NewDecoder(response.Body).Decode(&projects)

	return projects, errors.Wrap(err, "failed to decode response")
}

func GetProjectKeys(ctx context.Context, project Project) ([]Key, error) {
	url := fmt.Sprintf("/api/0/projects/%s/%s/keys/", project.Organization.Slug, project.Slug)

	req, err := rawRequest(ctx, http.MethodGet, url)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make request")
	}

	if response.StatusCode != http.StatusOK {
		return nil, errors.New("result not OK")
	}

	defer response.Body.Close()

	defer response.Body.Close()

	keys := []Key{}

	err = json.NewDecoder(response.Body).Decode(&keys)

	return keys, errors.Wrap(err, "failed to decode response")
}

func rawRequest(ctx context.Context, method string, path string) (*http.Request, error) {
	endpoint := fmt.Sprintf("%s%s", strings.TrimRight(*config.Get().SentryEndpoint, "/"), path)

	log.Debugf("Sending request to: %s", endpoint)

	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *config.Get().SentryToken))

	return req, nil
}
