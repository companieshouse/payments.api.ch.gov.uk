package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/data"

	"github.com/gorilla/pat"
)

// Register defines the route mappings
func Register(r *pat.Router) {
	r.Get("/healthcheck", getHealthCheck).Name("get-healthcheck")
	r.Post("/payments", createPaymentSession).Name("create-payment")
}

func getHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

var client http.Client

func createPaymentSession(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		log.ErrorR(req, fmt.Errorf("Request Body Empty"))
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}

	requestDecoder := json.NewDecoder(req.Body)
	var incomingPaymentResourceRequest data.IncomingPaymentResourceRequest
	err := requestDecoder.Decode(&incomingPaymentResourceRequest)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("Request Body Invalid"))
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}

	resource := incomingPaymentResourceRequest.Resource

	cfg, err := config.Get()
	if err != nil {
		log.ErrorR(req, fmt.Errorf("Error getting config: [%s]", err))
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	url, err := url.Parse(resource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("Error parsing resource: [%s]", err))
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}

	whitelist := strings.Split(cfg.DomainWhitelist, ",")
	matched := false
	for _, domain := range whitelist {
		if url.Host == domain {
			matched = true
			break
		}
	}
	if !matched {
		log.ErrorR(req, fmt.Errorf("Invalid resource domain: %s", url.Host))
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}

	resourceReq, err := http.NewRequest("GET", resource, nil)
	if err != nil {
		log.ErrorR(resourceReq, fmt.Errorf("Failed to create Resource Request: [%s]", err))
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	resp, err := client.Do(resourceReq)
	if err != nil {
		log.ErrorR(resourceReq, fmt.Errorf("Error getting Cost Resource: [%s]", err))
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.ErrorR(resourceReq, fmt.Errorf("Error reading Cost Resource: [%s]", err))
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}

	// TODO save cost resource and ensure all mandatory fields are present:

	paymentResource := &data.PaymentResource{}
	err = json.Unmarshal(body, paymentResource)
	if err != nil {
		log.ErrorR(resourceReq, fmt.Errorf("Error reading Cost Resource: [%s]", err))
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}

	user := strings.Split(req.Header.Get("Eric-Authorised-User"), ";")
	email := user[0]
	var forename string
	var surname string

	for i := 1; i < len(user); i++ {
		v := strings.Split(user[i], "=")
		if v[0] == " forename" {
			forename = v[1]
		} else if v[0] == " surname" {
			surname = v[1]
		} else {
			log.ErrorR(req, fmt.Errorf("Unexpected format in Eric-Authorised-User: %s", user))
			w.WriteHeader(http.StatusInternalServerError) // 500
			return
		}
	}
	paymentResource.CreatedBy = data.CreatedBy{
		ID:       req.Header.Get("Eric-Identity"),
		Email:    email,
		Forename: forename,
		Surname:  surname,
	}

	paymentResource.CreatedAt = time.Now()
	paymentResource.Reference = incomingPaymentResourceRequest.Reference

	if data.CreatePaymentResourceDB(paymentResource) != nil {
		log.ErrorR(req, fmt.Errorf("error writing to MongoDB: %s", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Add data to response
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("Error writing response: %s", err))
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}

	// log.Trace("TODO log successful creation with details") // TODO
}
