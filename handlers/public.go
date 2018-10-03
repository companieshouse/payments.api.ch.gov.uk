package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/gorilla/pat"
)

type paymentService struct {
	DAO dao.DAO
}

// Register defines the route mappings
func Register(r *pat.Router, cfg config.Config) {
	m := &dao.Mongo{
		URL: cfg.MongoDBURL,
	}
	p := &paymentService{
		DAO: m,
	}
	r.Get("/healthcheck", getHealthCheck).Name("get-healthcheck")
	r.Post("/payments", p.createPaymentSession).Name("create-payment")
}

// Return a 200 response if service is running
func getHealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Create a payment session and return a journey URL for the calling app to redirect to
func (service *paymentService) createPaymentSession(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		log.ErrorR(req, fmt.Errorf("request body empty"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	requestDecoder := json.NewDecoder(req.Body)
	var incomingPaymentResourceRequest models.IncomingPaymentResourceRequest
	if requestDecoder.Decode(&incomingPaymentResourceRequest) != nil {
		log.ErrorR(req, fmt.Errorf("request body invalid"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	paymentResource, err := getPaymentResource(w, req, incomingPaymentResourceRequest.Resource)
	if err != nil {
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
			log.ErrorR(req, fmt.Errorf("unexpected format in Eric-Authorised-User: %s", user))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	paymentResource.CreatedBy = models.CreatedBy{
		ID:       req.Header.Get("Eric-Identity"),
		Email:    email,
		Forename: forename,
		Surname:  surname,
	}

	paymentResource.CreatedAt = time.Now()
	paymentResource.Reference = incomingPaymentResourceRequest.Reference
	paymentResource.PaymentResourceID = generateRandomId()

	err = service.DAO.CreatePaymentResourceDB(paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing to MongoDB: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Add data to response
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// log.Trace("TODO log successful creation with details") // TODO
}

func getPaymentResource(w http.ResponseWriter, req *http.Request, resource string) (*models.PaymentResource, error) {

	cfg, err := config.Get()
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting config: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}
	parsedURL, err := url.Parse(resource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error parsing resource: [%v]", err))
		w.WriteHeader(http.StatusBadRequest)
		return nil, err
	}

	whitelist := strings.Split(cfg.DomainWhitelist, ",")
	matched := false
	for _, domain := range whitelist {
		if parsedURL.Host == domain {
			matched = true
			break
		}
	}
	if !matched {
		err = fmt.Errorf("invalid resource domain: %s", parsedURL.Host)
		log.ErrorR(req, err)
		w.WriteHeader(http.StatusBadRequest)
		return nil, err
	}

	resourceReq, err := http.NewRequest("GET", resource, nil)
	if err != nil {
		log.ErrorR(resourceReq, fmt.Errorf("failed to create Resource Request: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}

	var client http.Client
	resp, err := client.Do(resourceReq)
	if err != nil {
		log.ErrorR(resourceReq, fmt.Errorf("error getting Cost Resource: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.ErrorR(resourceReq, fmt.Errorf("error reading Cost Resource: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}

	// TODO save cost resource and ensure all mandatory fields are present:

	paymentResource := &models.PaymentResource{}
	err = json.Unmarshal(body, paymentResource)
	if err != nil {
		log.ErrorR(resourceReq, fmt.Errorf("error reading Cost Resource: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}
	return paymentResource, nil
}

func generateRandomId() (i string) {
	ranNumber := strconv.Itoa(1000000 + rand.Intn(9000000))
	now := time.Now()
	nanos := now.UnixNano()
	millis := strconv.FormatInt((nanos / 1000000), 10)
	return ranNumber + millis
}
