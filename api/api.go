package api

import (
	"github.com/cpacia/ens-lite"
	"net/http"
	"path"
	"fmt"
	"encoding/hex"
)

const Addr = "127.0.0.1:31313"

var client *ens.ENSLiteClient

func ServeAPI(ensClient *ens.ENSLiteClient) error {
	client = ensClient
	http.HandleFunc("/", serve)
	err := http.ListenAndServe(Addr, nil)
	if err != nil {
		return err
	}
	return nil
}

func resolve(w http.ResponseWriter, r *http.Request) {
	_, name := path.Split(r.URL.Path)
	resp, err := client.Resolve(name)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	fmt.Fprint(w, hex.EncodeToString(resp[:]))
}

func shutdown() {
	client.Stop()
}

func serve(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		resolve(w, r)
	case "POST":
		shutdown()
	}
}