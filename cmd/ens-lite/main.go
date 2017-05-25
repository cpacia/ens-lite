package main

import (
	"fmt"
	"github.com/cpacia/ens-lite"
	"github.com/cpacia/ens-lite/api"
	"github.com/cpacia/ens-lite/cli"
	"github.com/jessevdk/go-flags"
	"github.com/mitchellh/go-homedir"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
)

const VERSION = "0.1.0"

var parser = flags.NewParser(nil, flags.Default)

type Start struct {
	DataDir string `short:"d" long:"datadir" description:"specify the data directory to be used"`
}
type Version struct{}

var start Start
var version Version
var client *ens.ENSLiteClient

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			fmt.Println("ens-lite shutting down...")
			client.Stop()
			os.Exit(1)
		}
	}()
	parser.AddCommand("start",
		"start the resolver",
		"The start command starts the resolver daemon",
		&start)
	parser.AddCommand("version",
		"print the version number",
		"Print the version number and exit",
		&version)
	cli.SetupCli(parser)
	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}

func (x *Version) Execute(args []string) error {
	fmt.Println(VERSION)
	return nil
}

func (x *Start) Execute(args []string) error {
	var err error
	var dataDir string
	if x.DataDir == "" {
		path, err := getRepoPath()
		if err != nil {
			return err
		}
		dataDir = path
	} else {
		dataDir = x.DataDir
	}
	client, err = ens.NewENSLiteClient(dataDir)
	if err != nil {
		return err
	}
	fmt.Println("Ens Resolver Running...")
	go client.Start()
	api.ServeAPI(client)
	return nil
}

/* Returns the directory to store repo data in.
   It depends on the OS and whether or not we are on testnet. */
func getRepoPath() (string, error) {
	// Set default base path and directory name
	path := "~"
	directoryName := "ens"

	// Override OS-specific names
	switch runtime.GOOS {
	case "linux":
		directoryName = ".ens"
	case "darwin":
		path = "~/Library/Application Support"
	}

	// Join the path and directory name, then expand the home path
	fullPath, err := homedir.Expand(filepath.Join(path, directoryName))
	if err != nil {
		return "", err
	}

	// Return the shortest lexical representation of the path
	return filepath.Clean(fullPath), nil
}
