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

type createPaymentResource struct {
	RedirectURI string `json:"redirect_uri"`
	Reference   string `json:"reference"`
	Resource    string `json:"resource"`
	State       string `json:"state"`
}

// PaymentResource is the payment details to be stored in the DB and returned in the response
type PaymentResource struct {
	//	ID                      string    `json:"id"                                  bson:"_id"`                                 // TODO implement
	Amount                  string    `json:"amount"                              bson:"amount"`                              // TODO implement
	AvailablePaymentMethods []string  `json:"available_payment_methods,omitempty" bson:"available_payment_methods,omitempty"` // TODO implement
	CompletedAt             time.Time `json:"completed_at,omitempty"              bson:"completed_at,omitempty"`              // TODO implement
	CreatedAt               time.Time `json:"created_at,omitempty"                bson:"created_at,omitempty"`
	CreatedBy               CreatedBy `json:"created_by"                          bson:"created_by"`
	Description             string    `json:"description"                         bson:"description"`              // TODO implement
	Links                   Links     `json:"links"                               bson:"links"`                    // TODO implement
	PaymentMethod           string    `json:"payment_method,omitempty"            bson:"payment_method,omitempty"` // TODO implement
	Reference               string    `json:"reference,omitempty"                 bson:"reference,omitempty"`      // TODO implement
	Status                  string    `json:"status"                              bson:"status"`                   // TODO implement
}

// CreatedBy is the user who is creating the payment session
type CreatedBy struct {
	Email    string `bson:"email"`
	Forename string `bson:"forename"`
	ID       string `bson:"id"`
	Surname  string `bson:"surname"`
}

// Links is a set of URLs related to the resource, including self
type Links struct {
	Journey  string `json:"journey"`
	Resource string `json:"resource"`
	Self     string `json:"self"`
}

//Data is a representation of the top level data retrieved from the Transaction API
type Data struct {
	CompanyName string            `json:"company_name"`
	Filings     map[string]Filing `json:"filings"`
}

//Filing is a representation of the Filing subsection of data retrieved from the Transaction API
type Filing struct {
	Description string `json:"description"`
}

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
	var b createPaymentResource
	err := requestDecoder.Decode(&b)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("Request Body Invalid"))
		w.WriteHeader(http.StatusBadRequest) // 400
		return
	}

	resource := b.Resource

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

	paymentResource := &PaymentResource{}
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
		}
	}
	paymentResource.CreatedBy = CreatedBy{
		ID:       req.Header.Get("Eric-Identity"),
		Email:    email,
		Forename: forename,
		Surname:  surname,
	}

	paymentResource.CreatedAt = time.Now()
	paymentResource.Reference = b.Reference

	// Write to DB
	session, err := data.GetMongoSession()
	if err != nil {
		log.ErrorR(req, fmt.Errorf("Error connecting to MongoDB: %s", err))
		w.WriteHeader(http.StatusInternalServerError) // 500
		return
	}
	defer session.Close()

	c := session.DB("transactions").C("payments")

	if err := c.Insert(paymentResource); err != nil {
		log.ErrorR(req, fmt.Errorf("Error writing to MongoDB: %s", err))
		w.WriteHeader(http.StatusInternalServerError) // 500
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

	// TODO Add tests
	// log.Trace("TODO log successful creation with details") // TODO
}
