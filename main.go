package main

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/handlers/private" //Private API handling
	"github.com/companieshouse/payments.api.ch.gov.uk/handlers/public"  //Public API handling

	eric "github.com/companieshouse/eric/chain" // Identity bridge

	"github.com/gorilla/pat"
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

	router := pat.New()
	chain := alice.New()

	chain = eric.Register(chain)

	private.Register(router)
	public.Register(router)

	log.Info("Starting " + namespace)
	err = http.ListenAndServe(cfg.BindAddr, chain.Then(router))
	if err != nil {
		log.Error(err)
	}

	log.Trace("Exiting " + namespace)
}
