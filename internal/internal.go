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
package internal

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/maksim-paskal/pod-admission-controller/pkg/api"
	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/maksim-paskal/pod-admission-controller/pkg/metrics"
	"github.com/maksim-paskal/pod-admission-controller/pkg/sentry"
	"github.com/maksim-paskal/pod-admission-controller/pkg/types"
	"github.com/maksim-paskal/pod-admission-controller/pkg/utils"
	"github.com/maksim-paskal/pod-admission-controller/pkg/web"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// webhook spec timeoutSeconds.
const serverTimeout = 5 * time.Second

func Start() error {
	ctx := context.Background()

	log.Infof("Starting %s...", config.GetVersion())

	if err := CheckConfigRules(); err != nil {
		return errors.Wrap(err, "error in config rules")
	}

	if err := sentry.CreateCache(); err != nil {
		return errors.Wrap(err, "failed to create sentry cache")
	}

	sCert, err := tls.LoadX509KeyPair(*config.Get().CertFile, *config.Get().KeyFile)
	if err != nil {
		return errors.Wrap(err, "can not load certificates")
	}

	go startServerTLS(sCert)
	go startMetricsServer()

	<-ctx.Done()

	return nil
}

// start webhook server.
func startServerTLS(sCert tls.Certificate) {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *config.Get().Port),
		Handler:      web.GetHandler(),
		ReadTimeout:  serverTimeout,
		WriteTimeout: serverTimeout,
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{sCert},
		},
	}

	log.Infof("Listening on address %s", server.Addr)

	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.WithError(err).Fatal("Failed to start tls server")
	}
}

// start metrics server.
func startMetricsServer() {
	mux := http.NewServeMux()

	mux.Handle("/metrics", metrics.GetHandler())

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *config.Get().MetricsPort),
		Handler:      mux,
		ReadTimeout:  serverTimeout,
		WriteTimeout: serverTimeout,
	}

	log.Infof("Listening metrics on address %s", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		log.WithError(err).Fatal("Failed to start metrics server")
	}
}

// check config for templating errors.
func CheckConfigRules() error {
	for _, containerConfig := range config.Get().Rules {
		containerInfo := types.ContainerInfo{}

		_, err := utils.CheckConditions(containerInfo, containerConfig.Conditions)
		if err != nil {
			return errors.Wrap(err, "error in conditions")
		}

		_, err = api.FormatEnv(containerInfo, containerConfig.Env)
		if err != nil {
			return errors.Wrap(err, "error in env")
		}
	}

	return nil
}
