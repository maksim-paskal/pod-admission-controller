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
	"crypto/x509"
	"flag"
	"net/http"
	"os"
	"time"

	logrushooksentry "github.com/maksim-paskal/logrus-hook-sentry"
	"github.com/maksim-paskal/pod-admission-controller/pkg/api"
	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	"github.com/maksim-paskal/pod-admission-controller/pkg/metrics"
	"github.com/maksim-paskal/pod-admission-controller/pkg/sentry"
	"github.com/maksim-paskal/pod-admission-controller/pkg/web"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// webhook spec timeoutSeconds.
const serverTimeout = 5 * time.Second

var (
	testPod       = flag.String("test.pod", "", "test pod")
	testNamespace = flag.String("test.namespace", "", "test namespace")
)

func Start(ctx context.Context) error {
	hook, err := logrushooksentry.NewHook(ctx, logrushooksentry.Options{
		SentryDSN: *config.Get().SentryDSN,
		Release:   config.GetVersion(),
	})
	if err != nil {
		log.WithError(err).Error()
	}

	log.AddHook(hook)

	log.Infof("Starting %s...", config.GetVersion())

	if err := sentry.CreateCache(ctx); err != nil {
		return errors.Wrap(err, "failed to create sentry cache")
	}

	if len(*testPod)+len(*testNamespace) > 0 {
		patchBytes, err := api.TestPOD(ctx, *testNamespace, *testPod)
		if err != nil {
			log.WithError(err).Error()
		}

		log.Info("Creating patch.json...")

		if err := os.WriteFile("patch.json", patchBytes, 0o600); err != nil { //nolint:gomnd,mnd
			log.WithError(err).Error()
		}

		os.Exit(0)

		return nil
	}

	sCert, err := tls.LoadX509KeyPair(*config.Get().CertFile, *config.Get().KeyFile)
	if err != nil {
		return errors.Wrap(err, "can not load certificates")
	}

	if err := printCertInfo(sCert); err != nil {
		return errors.Wrap(err, "can not print certificate info")
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

	if err := server.ListenAndServeTLS("", ""); err != nil && ctx.Err() == nil {
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

	if err := server.ListenAndServe(); err != nil && ctx.Err() == nil {
		log.WithError(err).Fatal()
	}
}

func printCertInfo(sCert tls.Certificate) error {
	for _, cert := range sCert.Certificate {
		x509Cert, err := x509.ParseCertificate(cert)
		if err != nil {
			return errors.Wrap(err, "can not parse certificate")
		}

		log.Infof("Certificate valid for %s till %s",
			x509Cert.Subject.CommonName,
			x509Cert.NotAfter.String(),
		)
	}

	return nil
}
