package main

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
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

	if cfg.GovPayBearerTokenChAccount == "" {
		log.Info("WARNING: You need to set GovPayBearerTokenChAccount")
	}
	if cfg.GovPayBearerTokenLegacy == "" {
		log.Info("WARNING: You need to set GovPayBearerTokenLegacy")
	}
	if cfg.GovPayBearerTokenTreasury == "" {
		log.Info("WARNING: You need to set GovPayBearerTokenTreasury")
	}

	paymentsDAO := dao.NewDAO(cfg)

	// Create router
	mainRouter := mux.NewRouter()

	handlers.Register(mainRouter, *cfg, paymentsDAO)

	log.Info("Starting " + namespace)
	err = http.ListenAndServe(cfg.BindAddr, mainRouter)
	if err != nil {
		log.Error(err)
	}

	log.Trace("Exiting " + namespace)
}
