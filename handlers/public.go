package handlers

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/service"
	"github.com/gorilla/pat"
)

// Register defines the route mappings
func Register(r *pat.Router, cfg config.Config) {
	m := &dao.Mongo{
		URL: cfg.MongoDBURL,
	}
	p := &service.PaymentService{
		DAO: m,
	}

	r.Get("/healthcheck", healthCheck).Name("get-healthcheck")
	r.Post("/payments", p.CreatePaymentSession).Name("create-payment")
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Generates a string of 20 numbers made up of 8 random numbers, followed by 12 numbers derived from the current time
func generateID() (i string) {
	ranNumber := strconv.Itoa(10000000 + rand.Intn(90000000))
	millis := strconv.FormatInt((time.Now().UnixNano() / int64(time.Millisecond)), 10)
	return ranNumber + millis
}
