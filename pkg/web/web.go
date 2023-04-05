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
package web

import (
	"fmt"
	"io"
	"net/http"

	"github.com/maksim-paskal/pod-admission-controller/pkg/api"
	log "github.com/sirupsen/logrus"
)

func GetHandler() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/mutate", mutate)
	mux.HandleFunc("/ready", healthz)
	mux.HandleFunc("/healthz", healthz)

	return mux
}

func mutate(w http.ResponseWriter, r *http.Request) {
	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Errorf("contentType=%s, expect application/json", contentType)
		http.Error(w, "Error reading body", http.StatusBadRequest)

		return
	}

	var body []byte

	if r.Body != nil {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			log.WithError(err).Error("Error reading body")
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}
		defer r.Body.Close()

		body = data
	}

	log.Debugf("Received handleMutate %s ", string(body))

	respBytes, err := api.ParseRequest(r.Context(), body)
	if err != nil {
		log.WithError(err).Error()
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	log.Debugf("Sending response: %s", string(respBytes))

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write(respBytes); err != nil {
		log.WithError(err).Error()
	}
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "ok")
}
