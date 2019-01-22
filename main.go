package main

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"

	eric "github.com/companieshouse/eric/chain" // Identity bridge
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/handlers"
	"github.com/companieshouse/payments.api.ch.gov.uk/interceptors"

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
	router.Use(interceptors.AuthenticationInterceptor)
	handlers.Register(router, *cfg)

	log.Info("Starting " + namespace)
	err = http.ListenAndServe(cfg.BindAddr, chain.Then(router))
	if err != nil {
		log.Error(err)
	}

	log.Trace("Exiting " + namespace)
}
