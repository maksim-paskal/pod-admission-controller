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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maksim-paskal/pod-admission-controller/internal"
	"github.com/maksim-paskal/pod-admission-controller/pkg/api"
	"github.com/maksim-paskal/pod-admission-controller/pkg/config"
	log "github.com/sirupsen/logrus"
)

var version = flag.Bool("version", false, "print version and exit")

func main() { //nolint:funlen
	flag.Parse()

	if *version {
		fmt.Println(config.GetVersion()) //nolint:forbidigo
	}

	if err := config.Load(); err != nil {
		log.WithError(err).Fatal()
	}

	level, err := log.ParseLevel(*config.Get().LogLevel)
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.SetReportCaller(true)
	log.SetLevel(level)

	if !*config.Get().LogPretty {
		log.SetFormatter(&log.JSONFormatter{})
	}

	if err := config.Check(); err != nil {
		log.Info(config.Get())
		log.WithError(err).Fatal()
	}

	if err := api.Init(); err != nil {
		log.WithError(err).Fatal()
	}

	log.Debugf("using config: %s", config.Get())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChanInterrupt := make(chan os.Signal, 1)
	signal.Notify(signalChanInterrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.RegisterExitHandler(func() {
			cancel()
			// wait before shutdown
			time.Sleep(*config.Get().GracePeriod)
		})

		select {
		case <-signalChanInterrupt:
			log.Error("Got interruption signal...")
			cancel()
		case <-ctx.Done():
		}
		<-signalChanInterrupt
		os.Exit(1)
	}()

	if err := internal.Start(ctx); err != nil {
		log.WithError(err).Error()
	}

	log.Error("Server has stoped...")
}
