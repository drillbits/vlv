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
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

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

	coll, err := vlv.OpenCollection(ctx, cmd.config.Store)
	if err != nil {
		log.Printf("failed to open collection: %s", err)
		return subcommands.ExitFailure
	}
	defer coll.Close()

	d := vlv.NewDispatcher(client, coll)
	go d.Start(ctx)

	srv := vlv.NewServer(cmd.config.Address, coll)

	go func() {
		log.Printf("starting to listen on tcp %s", srv.Addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// error starting or closing listener:
			log.Printf("server closed: %s", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, os.Interrupt)
	log.Printf("SIGNAL %d received, shutting down...", <-quit)

	if err := srv.Shutdown(ctx); err != nil {
		// error from closing listeners:
		log.Printf("Failed to gracefully shutdown: %s", err)
		return subcommands.ExitFailure
	}
	log.Println("server shutdown")

	return subcommands.ExitSuccess
}
