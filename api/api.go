package api

import (
	"encoding/hex"
	"fmt"
	"github.com/cpacia/ens-lite"
	"net/http"
	"path"
)

const Addr = "127.0.0.1:31313"

var client *ens.ENSLiteClient

func ServeAPI(ensClient *ens.ENSLiteClient) error {
	topMux := http.NewServeMux()
	handler := new(resolverHandler)
	topMux.Handle("/resolver/", handler)
	topMux.Handle("/ws",newWSAPIHandler(ensClient))
	srv := &http.Server{Addr: Addr, Handler: topMux}
	handler.server = srv

	client = ensClient
	err := srv.ListenAndServe()
	if err != nil {
		return err
	}
	return nil
}

type resolverHandler struct{
	server *http.Server
}

func resolve(w http.ResponseWriter, r *http.Request) {
	_, name := path.Split(r.URL.Path)
	resp, err := client.Resolve(name)
	if err != nil && err == ens.ErrorBlockchainSyncing {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	} else if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	fmt.Fprint(w, hex.EncodeToString(resp[:]))
}

func (rh resolverHandler) shutdown() {
	client.Stop()
	rh.server.Close()
}

func (rh resolverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		resolve(w, r)
	case "POST":
		rh.shutdown()
	}
}
