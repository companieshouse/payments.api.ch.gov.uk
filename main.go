package main

import (
	"fmt"
	"github.com/companieshouse/payments.api.ch.gov.uk/wrappers"
	"github.com/davecgh/go-spew/spew"
	"net/http"

	"github.com/companieshouse/chs.go/log"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/handlers"

	eric "github.com/companieshouse/eric/chain" // Identity bridge

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

func main() {
	namespace := "payments.api.ch.gov.uk"
	log.Namespace = namespace

	cfg, err := config.Get()
	if err != nil {
		log.Error(fmt.Errorf("error configuring service: %s. Exiting", err), nil)
		return
	}

	router := mux.NewRouter()
	chain := alice.New()

	chain = eric.Register(chain)
	router.Use(wrappers.IsAuthorized)
	handlers.Register(router, *cfg)

	log.Info("Starting " + namespace)
	err = http.ListenAndServe(cfg.BindAddr, chain.Then(router))
	if err != nil {
		log.Error(err)
	}

	log.Trace("Exiting " + namespace)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do stuff here
		spew.Dump(r.RequestURI)
		spew.Dump("DALLLLELLLELELEL")
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
