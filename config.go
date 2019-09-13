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
	"github.com/BurntSushi/toml"
)

// Config represents a configuration for vlv.
type Config struct {
	Address string       `toml:"addr"`
	Store   *StoreConfig `toml:"store"`
}

// StoreConfig represents a configuration for store.
type StoreConfig struct {
	URL       string `toml:"url"`
	Localfile string `toml:"localfile"`
}

// LoadConfig loads a file as Config.
func LoadConfig(path string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}

	if config.Address == "" {
		config.Address = ":5151"
	}

	if config.Store == nil {
		config.Store = &StoreConfig{}
	}

	if config.Store.URL == "" {
		config.Store.URL = "mem://collection/Filename"
	}

	return &config, nil
}
