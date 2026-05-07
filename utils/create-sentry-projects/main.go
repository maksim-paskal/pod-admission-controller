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
package main

import (
	"context"
	_ "embed"
	"flag"
	"slices"
	"strings"

	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/maksim-paskal/pod-admission-controller/pkg/sentry"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

//go:embed prod-images.txt
var prodImages string

type Application struct{}

func (a *Application) checkConfig() error {
	needProjects := []string{}

	for _, image := range strings.Split(prodImages, "\n") {
		if image == "" {
			continue
		}

		parts := strings.Split(image, "/")
		sentryProject := strings.Join(parts[1:len(parts)-1], "/")

		if !slices.Contains(needProjects, sentryProject) {
			needProjects = append(needProjects, sentryProject)
		}
	}

	notFoundProjects := []string{}

	for _, project := range needProjects {
		if !config.Get().Sentry.HasProjectPrefix(project) {
			split := strings.Split(project, "/")
			notFoundProjects = append(notFoundProjects, split[len(split)-1]+": "+project)
		}
	}

	if len(notFoundProjects) > 0 {
		return errors.Errorf("please add to your config to your sentry project config:\n\n%s", strings.Join(notFoundProjects, "\n")) //nolint:lll
	}

	return nil
}

func (a *Application) createProjects() error {
	projectsToCreate := []string{}

	for project := range config.Get().Sentry.Projects {
		if sentry.HasProjectInCache(project) {
			continue
		}

		for _, prefix := range []string{"prod", "stage", "branch"} {
			projectsToCreate = append(projectsToCreate, prefix+"-"+project)
		}
	}

	log.Infof("Project to be created: %d", len(projectsToCreate))

	for _, project := range projectsToCreate {
		log.Infof("Creating Sentry project %s", project)

		if err := sentry.CreateProject(project); err != nil {
			return errors.Wrapf(err, "creating Sentry project %s", project)
		}
	}

	return nil
}

func (a *Application) Run(ctx context.Context) error {
	if err := a.checkConfig(); err != nil {
		return errors.Wrap(err, "checking config")
	}

	if err := sentry.CreateCache(ctx); err != nil {
		return errors.Wrap(err, "creating Sentry cache")
	}

	if err := a.createProjects(); err != nil {
		return errors.Wrap(err, "creating projects")
	}

	log.Info("Add to your cache in config: \n" + sentry.GetKVCache())

	return nil
}

func main() {
	flag.Parse()

	level, err := log.ParseLevel(*config.Get().LogLevel)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(level)

	if err := config.Load(); err != nil {
		log.Fatal(err)
	}

	app := &Application{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.Run(ctx); err != nil {
		log.Error(err)
	}
}
