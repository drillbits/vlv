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
	"context"
	"log"
	"net/url"
	"strings"

	"gocloud.dev/docstore"
	// in-memory driver
	"gocloud.dev/docstore/memdocstore"
)

// OpenCollection opens the collection of tasks.
func OpenCollection(ctx context.Context, config *StoreConfig) (*docstore.Collection, error) {
	urlstr := config.URL

	u, err := url.Parse(urlstr)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(u.Scheme, memdocstore.Scheme) {
		log.Printf("open collection: %s", urlstr)
		coll, err := docstore.OpenCollection(ctx, urlstr)
		if err != nil {
			return nil, err
		}
		return coll, nil
	}

	ss := strings.Split(u.Path, "/")
	keyField := ss[len(ss)-1]
	log.Printf("open in-memory collection: keyField: %s, localfile: %s", keyField, config.Localfile)
	coll, err := memdocstore.OpenCollection(keyField, &memdocstore.Options{
		Filename: config.Localfile,
	})
	if err != nil {
		return nil, err
	}
	return coll, err
}
