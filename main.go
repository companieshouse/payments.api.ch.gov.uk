package main

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/handlers"
	"github.com/gorilla/mux"
)

func main() {
	namespace := "payments.api.ch.gov.uk"
	log.Namespace = namespace

	cfg, err := config.Get()
	if err != nil {
		log.Error(fmt.Errorf("error configuring service: %s. Exiting", err), nil)
		return
	}

	// Create router
	mainRouter := mux.NewRouter()

	payPalSvc, err := service.NewPayPalService(cfg)

	handlers.Register(mainRouter, *cfg, payPalSvc)

	log.Info("Starting " + namespace)
	err = http.ListenAndServe(cfg.BindAddr, mainRouter)
	if err != nil {
		log.Error(err)
	}

	log.Trace("Exiting " + namespace)
}
