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
	"fmt"
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

	current int64 `json:"-" docstore:"-"`
	size    int64 `json:"-" docstore:"-"`

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
func (t *Task) Do(ctx context.Context, client *http.Client, rate float64, capacity int64) error {
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
		t.current = current
		t.size = fi.Size()
		log.Printf("progress: %.2f%%, %d/%d bytes\n", t.progress(), t.current, t.size)
	}).Context(ctx).Do()
	if err != nil {
		return err
	}
	log.Printf("uploaded https://drive.google.com/file/d/%s/view\n", res.Id)

	return nil
}

func (t *Task) progress() float64 {
	return float64(t.current) / float64(t.size) * 100.0
}

func (t *Task) Status() *TaskStatus {
	return &TaskStatus{
		Progress: fmt.Sprintf("%.2f%%", t.progress()),
		Current:  t.current,
		Total:    t.size,
	}
}

type TaskStatus struct {
	Progress string `json:"progress"`
	Current  int64  `json:"current_bytes"`
	Total    int64  `json:"total_bytes"`
}

// Dispatcher represents a dispatcher.
type Dispatcher struct {
	client *http.Client
	coll   *docstore.Collection

	rate     float64
	capacity int64

	current *Task
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
		if d.shut {
			time.Sleep(1 * time.Minute)
			continue
		}

		// retrieve entries and upload.
		iter := d.coll.Query().OrderBy("createTime", "asc").Get(ctx)
		defer iter.Stop()

		var prev *Task
		for {
			if d.shut {
				time.Sleep(1 * time.Minute)
				continue
			}

			var t Task
			if prev != nil {
				t = *prev
			} else {
				if err := iter.Next(ctx, &t); err == io.EOF {
					break
				} else if err != nil {
					// TODO: error
					log.Printf("[ERROR] failed to iter collection: %s", err)
					return
				}
			}

			log.Printf("- %s: %#v\n", t.CreatedAt(), t)
			d.current = &t

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			ret := make(chan error)
			go func() {
				err := t.Do(ctx, d.client, d.rate, d.capacity)
				ret <- err
			}()
			go func() {
				for {
					if d.shut {
						log.Printf("vlv is shut now")
						cancel()
						break
					}
				}
			}()

			err := <-ret
			if err != nil {
				// TODO: error
				log.Printf("[ERROR] failed to execute task: %s", err)
				// TODO: retry?
				prev = &t
				continue
			}

			if err = d.coll.Delete(ctx, &t); err != nil {
				// TODO: error
				log.Printf("[ERROR] failed to delete task: %s", err)
			}
		}

		// TODO: hard-coded interval
		time.Sleep(1 * time.Second)
	}
}

func (d *Dispatcher) Status() *DispatcherStatus {
	s := new(DispatcherStatus)
	if d.current != nil {
		s.TaskStatus = d.current.Status()
	}
	s.Shut = d.shut
	return s
}

type DispatcherStatus struct {
	*TaskStatus
	Shut bool `json:"shut"`
}
