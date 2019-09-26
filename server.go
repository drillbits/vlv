// Copyright 2019 drillbits
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vlv

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gocloud.dev/docstore"
)

// NewServer creates a new server.
func NewServer(addr string, coll *docstore.Collection) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello, vlv."))
	})

	mux.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		t, err := TaskFromRequest(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "failed to decode: %s", err)
			return
		}

		if err := coll.Create(ctx, t); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "failed to create: %s", err)
			return
		}

		b, err := json.MarshalIndent(&t, "", "  ")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "failed to marshal: %s", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	})

	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		iter := coll.Query().Get(ctx)
		defer iter.Stop()

		for {
			var t Task
			if err := iter.Next(ctx, &t); err == io.EOF {
				break
			} else if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "failed to store the next: %s", err)
				return
			} else {
				fmt.Fprintf(w, "- %s: %#v\n", t.CreatedAt(), t)
			}
		}
	})

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
