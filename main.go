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

func main() {
	log.Namespace = "payments.api.ch.gov.uk"

	cfg := config.Get()

	router := pat.New()
	chain := alice.New()

	public.Register(router)
	private.Register(router)

	log.Info("Starting payments.api.ch.gov.uk service")
	err := http.ListenAndServe(cfg.BindAddr, chain.Then(router))

	if err != nil {
		log.Error(err)
	}
	log.Trace("Exiting payments.api.ch.gov.uk service")
}
