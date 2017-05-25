package main

import (
	"github.com/cpacia/ens-lite"
	"github.com/cpacia/ens-lite/api"
	"github.com/mitchellh/go-homedir"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
)

var client *ens.ENSLiteClient

func run() error {
	var err error
	var dataDir string
	path, err := getRepoPath()
	if err != nil {
		return err
	}
	dataDir = path
	client, err = ens.NewENSLiteClient(dataDir)
	if err != nil {
		return err
	}
	go client.Start()
	api.ServeAPI(client)
	return nil
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			client.Stop()
			os.Exit(1)
		}
	}()
	run()
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
