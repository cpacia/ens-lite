package api

import (
	"fmt"
	"github.com/cpacia/ens-lite"
	"github.com/miekg/dns"
	"net/http"
	"path"
	"time"
	"strconv"
	"strings"
)

const Addr = "127.0.0.1:31313"

var client *apiClient

type apiClient struct {
	ensClient *ens.ENSLiteClient
	cache     map[string]cachedRecord
}

type cachedRecord struct {
	rr         dns.RR
	expiration time.Time
}

func ServeAPI(ensClient *ens.ENSLiteClient) error {
	topMux := http.NewServeMux()
	handler := new(resolverHandler)

	topMux.Handle("/resolver/", handler)
	topMux.Handle("/ws", newWSAPIHandler(ensClient))
	srv := &http.Server{Addr: Addr, Handler: topMux}
	handler.server = srv

	client = &apiClient{ensClient, make(map[string]cachedRecord)}
	err := srv.ListenAndServe()
	if err != nil {
		return err
	}
	return nil
}

type resolverHandler struct {
	server *http.Server
}

func resolve(w http.ResponseWriter, r *http.Request) {
	urlPath, name := path.Split(r.URL.Path)
	_, queryType := path.Split(urlPath[:len(urlPath)-1])

	if strings.ToLower(queryType) == "dns" {
		lookup := r.URL.Query().Get("lookup")
		l, _ := strconv.ParseBool(lookup)
		record, ok := client.cache[name]
		if ok {
			if time.Now().Before(record.expiration) {
				fmt.Fprint(w, record.rr.(*dns.A).A.String())
				return
			} else {
				delete(client.cache, name)
			}
		}

		resp, err := client.ensClient.ResolveDNS(name)
		if err != nil && err == ens.ErrorBlockchainSyncing {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		} else if err != nil || len(resp) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		client.cache[name] = cachedRecord{resp[0], time.Now().Add(time.Duration(resp[0].Header().Ttl))}
		if l {
			var ret string
			for i, rec := range resp {
				ret += rec.String()
				if i != len(resp) - 1 {
					ret += "\n"
				}
			}
			fmt.Fprint(w, ret)
		} else {
			fmt.Fprint(w, resp[0].(*dns.A).A.String())
		}
	} else if strings.ToLower(queryType) == "address" {
		resp, err := client.ensClient.ResolveAddress(name)
		if err != nil && err == ens.ErrorBlockchainSyncing {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		} else if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprint(w, resp.Hex())
	}
}

func (rh resolverHandler) shutdown() {
	client.ensClient.Stop()
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
