package main

import (
	"net/http"

	"github.com/companieshouse/chs.go/log"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/handlers/private"
	"github.com/companieshouse/payments.api.ch.gov.uk/handlers/public"

	"github.com/gorilla/pat"
	"github.com/justinas/alice"
)

// Namespace:
var namespace = "payments.api.ch.gov.uk"

func main() {
	log.Namespace = namespace

	cfg := config.Get()

	router := pat.New()
	chain := alice.New()

	public.Register(router)
	private.Register(router)

	log.Info("Starting " + namespace)
	err := http.ListenAndServe(cfg.BindAddr, chain.Then(router))

	if err != nil {
		log.Error(err)
	}
	log.Trace("Exiting " + namespace)
}
