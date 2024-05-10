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
	"net/url"
	"strings"
	"time"

	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const defaultHTTPTimeout = 30 * time.Second

var client = &http.Client{
	Jar:     nil,
	Timeout: defaultHTTPTimeout + time.Minute,
}

type KeyDsn struct {
	Public string
}

type Key struct {
	Dsn KeyDsn
}

func (k *Key) GetRelayDSN() string {
	urlDSN, err := url.Parse(k.Dsn.Public)
	if err != nil {
		return k.Dsn.Public
	}

	urlRelay, err := url.Parse(config.Get().Sentry.Relay)
	if err != nil {
		return k.Dsn.Public
	}

	urlDSN.Scheme = urlRelay.Scheme
	urlDSN.Host = urlRelay.Host

	return urlDSN.String()
}

type Organization struct {
	Slug string
}

type Project struct {
	Organization Organization
	Slug         string
}

var cache map[string]string

func getCachedDSN(namespace, projectSlug string) (string, bool) {
	prefixes := []string{""}

	if userPrefixes := config.Get().Sentry.GetPrefixes(namespace); len(userPrefixes) > 0 {
		prefixes = append(prefixes, userPrefixes...)
	}

	log.Debugf("find %s with prefixes=%+v", projectSlug, prefixes)

	for _, p := range prefixes {
		if dsn, ok := cache[p+projectSlug]; ok {
			return dsn, true
		}
	}

	return "", false
}

func GetSentryDSN(namespace, imagePath string) (string, bool) {
	// if cache is not created, return false
	if cache == nil {
		return "", false
	}

	log.Debug("Check for imagePath=", imagePath)

	for projectSlug, projectImagePath := range config.Get().Sentry.Projects {
		log.Debugf("Check for projectImagePath=%s, imagePath=%s", projectImagePath, imagePath)

		if strings.HasPrefix(projectImagePath, imagePath) || strings.HasPrefix(imagePath, projectImagePath) {
			if dsn, ok := getCachedDSN(namespace, projectSlug); ok {
				return dsn, true
			}
		}
	}

	return "", false
}

func CreateCache(ctx context.Context) error {
	if config.Get().Sentry == nil {
		return nil
	}

	if len(config.Get().Sentry.Cache) > 0 {
		log.Info("Using Sentry cache from config...")

		cache = config.Get().Sentry.Cache

		return nil
	}

	if len(config.Get().Sentry.Endpoint) == 0 || len(config.Get().Sentry.Token) == 0 {
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
			return errors.Wrapf(err, "failed to get keys, project=%+v", project)
		}

		cache[project.Slug] = keys[0].Dsn.Public
	}

	var cacheStruct strings.Builder

	cacheStruct.WriteString("cache:")

	for k, v := range cache {
		cacheStruct.WriteString("\n  " + k + ": " + v)
	}

	log.Debug(cacheStruct.String())

	return nil
}

func GetProjects(ctx context.Context) ([]Project, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
	defer cancel()

	req, err := rawRequest(ctx, http.MethodGet, "/api/0/projects/")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make request")
	}

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf("result not OK, status=%d", response.StatusCode)
	}

	defer response.Body.Close()

	projects := []Project{}

	if err := json.NewDecoder(response.Body).Decode(&projects); err != nil {
		return nil, errors.Wrap(err, "failed to decode response")
	}

	return projects, nil
}

func GetProjectKeys(ctx context.Context, project Project) ([]Key, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultHTTPTimeout)
	defer cancel()

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
		return nil, errors.Errorf("result not OK, status=%d", response.StatusCode)
	}

	defer response.Body.Close()

	keys := []Key{}

	if err := json.NewDecoder(response.Body).Decode(&keys); err != nil {
		return nil, errors.Wrap(err, "failed to decode response")
	}

	if len(config.Get().Sentry.Relay) == 0 {
		return keys, nil
	}

	// change DSN to relay DSN
	keys[0].Dsn.Public = keys[0].GetRelayDSN()

	return keys, nil
}

func rawRequest(ctx context.Context, method string, path string) (*http.Request, error) {
	endpoint := fmt.Sprintf("%s%s", strings.TrimRight(config.Get().Sentry.Endpoint, "/"), path)

	log.Debugf("Sending request to: %s", endpoint)

	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+config.Get().Sentry.Token)

	return req, nil
}
