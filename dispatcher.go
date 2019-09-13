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
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"google.golang.org/api/drive/v3"
)

// Task represents a task to upload.
type Task struct {
	Filename    string   `json:"filename" docstore:"filename"`
	Description string   `json:"description" docstore:"description"`
	Parents     []string `json:"parents" docstore:"parents"`
	MimeType    string   `json:"mimeType" docstore:"mimeType"`

	DocstoreRevision interface{}
}

// Do uploads a file of the task.
func (t *Task) Do(client *http.Client) error {
	service, err := drive.New(client)
	if err != nil {
		return err
	}

	f, err := os.Open(t.Filename)
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
	res, err := service.Files.Create(dst).Media(f).Do()
	if err != nil {
		return err
	}
	log.Printf("uploaded https://drive.google.com/file/d/%s/view\n", res.Id)

	return nil
}

// Dispatcher represents a dispatcher.
type Dispatcher struct {
	client *http.Client
}

// NewDispatcher creates a new dispatcher.
func NewDispatcher(client *http.Client) *Dispatcher {
	d := &Dispatcher{
		client: client,
	}

	return d
}

// Start starts to dispatch.
func (d *Dispatcher) Start(ctx context.Context) {
	for {
		// TODO: retrieve entries and upload.
	}
}
