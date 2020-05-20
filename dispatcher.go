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
	"encoding/gob"
	"encoding/json"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/juju/ratelimit"
	"gocloud.dev/docstore"
	"google.golang.org/api/drive/v3"
)

func init() {
	// for array field of docstore struct
	gob.Register([]interface{}{})
}

// Task represents a task to upload.
type Task struct {
	Filename    string   `json:"filename" docstore:"filename"`
	Description string   `json:"description" docstore:"description"`
	Parents     []string `json:"parents" docstore:"parents"`
	MimeType    string   `json:"mimeType" docstore:"mimeType"`
	CreateTime  int64    `json:"createTime" docstore:"createTime"`

	DocstoreRevision interface{}
}

// TaskFromRequest creates a new Task from http.Request.
func TaskFromRequest(r *http.Request) (*Task, error) {
	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		return nil, err
	}
	defer r.Body.Close()

	t.CreateTime = time.Now().UnixNano()
	return &t, nil
}

// CreatedAt returns when the task was created.
func (t *Task) CreatedAt() time.Time {
	return time.Unix(0, t.CreateTime)
}

// Do uploads a file of the task.
func (t *Task) Do(client *http.Client, rate float64, capacity int64) error {
	service, err := drive.New(client)
	if err != nil {
		return err
	}

	f, err := os.Open(t.Filename)
	if err != nil {
		return err
	}
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	filename := filepath.Base(t.Filename)
	if t.MimeType == "" {
		t.MimeType = mime.TypeByExtension(filepath.Ext(filename))
	}
	dst := &drive.File{
		Name:        filename,
		Description: t.Description,
		Parents:     t.Parents,
		MimeType:    t.MimeType,
	}

	log.Printf("uploading %s\n", filename)

	b := ratelimit.NewBucketWithRate(rate, capacity)
	res, err := service.Files.Create(dst).Media(ratelimit.Reader(f, b)).ProgressUpdater(func(current, total int64) {
		log.Printf("progress: %.2f%%, %d/%d bytes\n", float64(current)/float64(fi.Size())*100.0, current, fi.Size())
	}).Do()
	if err != nil {
		return err
	}
	log.Printf("uploaded https://drive.google.com/file/d/%s/view\n", res.Id)

	return nil
}

// Dispatcher represents a dispatcher.
type Dispatcher struct {
	client *http.Client
	coll   *docstore.Collection

	rate     float64
	capacity int64

	shut    bool
}

// NewDispatcher creates a new dispatcher.
func NewDispatcher(client *http.Client, coll *docstore.Collection, rate float64, capacity int64) *Dispatcher {
	d := &Dispatcher{
		client:   client,
		coll:     coll,
		rate:     rate,
		capacity: capacity,
		shut:     false,
	}
	return d
}

// Start starts to dispatch.
func (d *Dispatcher) Start(ctx context.Context) {
	for {
		// retrieve entries and upload.
		iter := d.coll.Query().OrderBy("createTime", "asc").Get(ctx)
		defer iter.Stop()

		for {
			var t Task
			if err := iter.Next(ctx, &t); err == io.EOF {
				break
			} else if err != nil {
				// TODO: error
				log.Printf("[ERROR] failed to iter collection: %s", err)
				return
			} else {
				log.Printf("- %s: %#v\n", t.CreatedAt(), t)
				err = t.Do(d.client, d.rate, d.capacity)
				if err != nil {
					// TODO: error
					log.Printf("[ERROR] failed to execute task: %s", err)
					// TODO: retry?
				}
				// TODO: delete if task was failure
				if err = d.coll.Delete(ctx, &t); err != nil {
					// TODO: error
					log.Printf("[ERROR] failed to delete task: %s", err)
				}
			}
		}
		// TODO: hard-coded interval
		time.Sleep(1 * time.Second)
	}
}

func (d *Dispatcher) Status() *DispatcherStatus {
	s := new(DispatcherStatus)
	s.Shut = d.shut
	return s
}

type DispatcherStatus struct {
	Shut bool `json:"shut"`
}
