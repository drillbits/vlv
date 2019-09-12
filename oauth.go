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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

// NewDriveClient creates a new client for Google Drive.
func NewDriveClient(ctx context.Context, confdir string) (*http.Client, error) {
	secrets := filepath.Join(confdir, "client_secret.json")
	b, err := ioutil.ReadFile(secrets)
	if err != nil {
		return nil, err
	}
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, err
	}

	tokenfile := filepath.Join(confdir, "token.json")
	f, err := os.OpenFile(tokenfile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	if err == nil {
		return config.Client(ctx, tok), nil
	}

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Print("Access the following URL in your browser:")
	fmt.Printf("%s\n\n", authURL)
	fmt.Print("Enter verification code: ")
	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, err
	}
	tok, err = config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, err
	}

	err = json.NewEncoder(f).Encode(tok)
	if err != nil {
		return nil, err
	}

	return config.Client(ctx, tok), nil
}
