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
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/atlassian/go-sentry-api"
	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

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

var cache map[string]string

func GetKVCache() string {
	var cacheStruct strings.Builder

	keys := slices.SortedFunc(maps.Keys(cache), strings.Compare)

	for _, k := range keys {
		cacheStruct.WriteString("\n  " + k + ": " + cache[k])
	}

	return cacheStruct.String()
}

func HasProjectInCache(project string) bool {
	for k := range cache {
		if strings.HasSuffix(k, project) {
			return true
		}
	}

	return false
}

func CreateProject(projectSlug string) error {
	team, err := sentryClient.GetTeam(sentryOrganization, projectSlug)
	if err != nil {
		team, err = sentryClient.CreateTeam(sentryOrganization, projectSlug, &projectSlug)
		if err != nil {
			return errors.Wrapf(err, "creating team %s", projectSlug)
		}
	}

	project, err := sentryClient.GetProject(sentryOrganization, projectSlug)
	if err != nil {
		project, err = sentryClient.CreateProject(sentryOrganization, team, projectSlug, &projectSlug)
		if err != nil {
			return errors.Wrapf(err, "creating project %s", projectSlug)
		}
	}

	log.Debugf("Project created: %s", project.ID)

	dsn, err := getProjectDSN(project)
	if err != nil {
		return errors.Wrapf(err, "getting DSN for project %s", projectSlug)
	}

	cache[projectSlug] = dsn

	return nil
}

func getCachedDSN(namespace, projectSlug string) (string, bool) {
	// default will be search project without prefix
	prefixes := []string{""}

	if userPrefixes := config.Get().Sentry.GetPrefixes(namespace); len(userPrefixes) > 0 {
		for _, p := range userPrefixes {
			if slices.Contains(prefixes, p) {
				continue
			}

			prefixes = append(prefixes, p)
		}
	}

	log.Debugf("find %s with prefixes=%+v", projectSlug, prefixes)

	for _, p := range prefixes {
		name := p + projectSlug

		log.Debugf("Check for project %s in cache", name)

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

var (
	sentryClient       *sentry.Client
	sentryOrganization sentry.Organization
)

func clientInit() error {
	var err error

	sentryClient, err = sentry.NewClient(
		config.Get().Sentry.Token,
		&config.Get().Sentry.Endpoint,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "error creating Sentry client")
	}

	sentryOrganization, err = sentryClient.GetOrganization(config.Get().Sentry.Organization)
	if err != nil {
		return errors.Wrap(err, "failed to get organization")
	}

	return nil
}

func getProjectDSN(project sentry.Project) (string, error) {
	log.Debugf("get keys for %s/%s", sentryOrganization.Name, project.Name)

	keys, err := sentryClient.GetClientKeys(sentryOrganization, project)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get keys, project=%+v", project)
	}

	key := Key{
		Dsn: KeyDsn{
			Public: keys[0].DSN.Public,
		},
	}

	return key.GetRelayDSN(), nil
}

func CreateCache(ctx context.Context) error { //nolint:cyclop,funlen
	if config.Get().Sentry == nil {
		log.Warn("No Sentry config")

		return nil
	}

	if err := clientInit(); err != nil {
		return errors.Wrap(err, "initializing Sentry client")
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

	page, link, err := sentryClient.GetProjects()
	if err != nil {
		return errors.Wrap(err, "failed to get projects")
	}

	projects := make([]sentry.Project, 0)

	for ctx.Err() == nil {
		projects = append(projects, page...)

		if !link.Next.Results {
			break
		}

		link, err = sentryClient.GetPage(link.Next, &page)
		if err != nil {
			return errors.Wrap(err, "failed to get projects")
		}
	}

	log.Infof("Found %d projects in %s", len(projects), sentryOrganization.Name)
	log.Info("Getting DSN for each project...")

	for _, project := range projects {
		if ctx.Err() != nil {
			return errors.Wrap(ctx.Err(), "context canceled")
		}

		dsn, err := getProjectDSN(project)
		if err != nil {
			return errors.Wrapf(err, "getting DSN for project %s", project.Name)
		}

		cache[*project.Slug] = dsn
	}

	return nil
}
