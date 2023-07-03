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
	"net/http"
	"time"

	logrushooksentry "github.com/maksim-paskal/logrus-hook-sentry"
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

func Start(ctx context.Context) error {
	hook, err := logrushooksentry.NewHook(logrushooksentry.Options{
		SentryDSN: *config.Get().SentryDSN,
		Release:   config.GetVersion(),
	})
	if err != nil {
		log.WithError(err).Error()
	}

	log.AddHook(hook)
	defer hook.Stop()

	log.Infof("Starting %s...", config.GetVersion())

	if err := CheckConfigRules(); err != nil {
		return errors.Wrap(err, "error in config rules")
	}

	if err := sentry.CreateCache(ctx); err != nil {
		return errors.Wrap(err, "failed to create sentry cache")
	}

	sCert, err := tls.LoadX509KeyPair(*config.Get().CertFile, *config.Get().KeyFile)
	if err != nil {
		return errors.Wrap(err, "can not load certificates")
	}

	go startServerTLS(ctx, sCert)
	go startMetricsServer(ctx)

	return nil
}

// start webhook server.
func startServerTLS(ctx context.Context, sCert tls.Certificate) {
	server := &http.Server{
		Addr:         *config.Get().Addr,
		Handler:      web.GetHandler(),
		ReadTimeout:  serverTimeout,
		WriteTimeout: serverTimeout,
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{sCert},
		},
	}

	go func() {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), config.Get().GetGracePeriod())
		defer cancel()

		_ = server.Shutdown(ctx) //nolint:contextcheck
	}()

	log.Infof("Listening on address %s", server.Addr)

	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.WithError(err).Fatal()
	}
}

// start metrics server.
func startMetricsServer(ctx context.Context) {
	mux := http.NewServeMux()

	mux.Handle("/metrics", metrics.GetHandler())

	server := &http.Server{
		Addr:         *config.Get().MetricsAddr,
		Handler:      mux,
		ReadTimeout:  serverTimeout,
		WriteTimeout: serverTimeout,
	}

	go func() {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), config.Get().GetGracePeriod())
		defer cancel()

		_ = server.Shutdown(ctx) //nolint:contextcheck
	}()

	log.Infof("Listening metrics on address %s", server.Addr)

	if err := server.ListenAndServe(); err != nil {
		log.WithError(err).Fatal()
	}
}

// check config for templating errors.
func CheckConfigRules() error {
	for _, containerConfig := range config.Get().Rules {
		containerInfo := types.ContainerInfo{
			Image: &types.ContainerImage{},
		}

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
