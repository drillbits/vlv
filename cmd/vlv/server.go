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

package main

import (
	"context"
	"flag"
	"log"
	"path/filepath"

	"github.com/drillbits/vlv"
	"github.com/google/subcommands"
	"github.com/mitchellh/go-homedir"
)

type runCmd struct {
	confdir string
	config  vlv.Config
}

func (*runCmd) Name() string {
	return "run"
}

func (*runCmd) Synopsis() string {
	return "run vlv server."
}

func (*runCmd) Usage() string {
	return `run [-config] <config dir>:
  Run vlv server.
`
}

func (cmd *runCmd) SetFlags(f *flag.FlagSet) {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	f.StringVar(&cmd.confdir, "config", filepath.Join(home, ".config", "vlv"), "config directory")

	path := filepath.Join(cmd.confdir, "config.toml")
	config, err := vlv.LoadConfig(path)
	if err != nil {
		panic(err)
	}
	cmd.config = *config
}

func (cmd *runCmd) Execute(ctx context.Context, flagset *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	client, err := vlv.NewDriveClient(ctx, cmd.confdir)
	if err != nil {
		log.Printf("failed to create client: %s", err)
		return subcommands.ExitFailure
	}

	d := vlv.NewDispatcher(client)
	go d.Start(ctx)

	addr := cmd.config.Address
	srv := vlv.NewServer(addr, &cmd.config)
	log.Printf("starting to listen on tcp %s", addr)
	err = srv.ListenAndServe()
	if err != nil {
		log.Printf("failed to listen and serve: %s", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
