package cli

import (
	"github.com/jessevdk/go-flags"
	"net/http"
	"time"
	"github.com/cpacia/ens-lite/api"
	"fmt"
	"io/ioutil"
)



func SetupCli(parser *flags.Parser) {
	// Add commands to parser
	parser.AddCommand("stop",
		"stop the resover",
		"The stop command disconnects from peers and shuts down the resolver",
		&stop)
	parser.AddCommand("resolve",
		"resolve a name",
		"Resolve a name. The merkle proofs will be validated automatically.",
		&resolve)
}

type Stop struct{}

var stop Stop

func (x *Stop) Execute(args []string) error {
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	_, err := client.Post(api.Addr, "text/plain", nil)
	if err != nil {
		return err
	}
	fmt.Println("Ens Resolver Stopping...")
	return nil
}

type Resolve struct{}

var resolve Resolve

func (x *Resolve) Execute(args []string) error {
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := client.Get("http://" + api.Addr + "/" + args[0])
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Println("Not found")
		return err
	}
	h, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(h))
	return nil
}