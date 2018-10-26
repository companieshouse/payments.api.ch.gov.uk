package service

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
)

// PaymentService contains the DAO for db access
type PaymentService struct {
	DAO    dao.DAO
	Config config.Config
}

// CreatePaymentSession creates a payment session and returns a journey URL for the calling app to redirect to
func (service *PaymentService) CreatePaymentSession(w http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		log.ErrorR(req, fmt.Errorf("request body empty"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	requestDecoder := json.NewDecoder(req.Body)
	var incomingPaymentResourceRequest models.IncomingPaymentResourceRequest
	err := requestDecoder.Decode(&incomingPaymentResourceRequest)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("request body invalid: [%v]", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	costs, err := getCosts(w, req, incomingPaymentResourceRequest.Resource, &service.Config)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting payment resource: [%v]", err))
		return
	}

	totalAmount, err := getTotalAmount(costs)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting amount from costs: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
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

	var paymentResource models.PaymentResource
	paymentResource.CreatedBy = models.CreatedBy{
		ID:       req.Header.Get("Eric-Identity"),
		Email:    email,
		Forename: forename,
		Surname:  surname,
	}
	paymentResource.Amount = totalAmount
	paymentResource.CreatedAt = time.Now()
	paymentResource.Reference = incomingPaymentResourceRequest.Reference
	paymentResource.ID = generateID()

	journeyURL := service.Config.PaymentsWebURL + "/payments/" + paymentResource.ID + "/pay"
	paymentResource.Links = models.Links{
		Journey:  journeyURL,
		Resource: incomingPaymentResourceRequest.Resource,
	}

	err = service.DAO.CreatePaymentResource(&paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing to MongoDB: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Add data to response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", journeyURL)
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		return
	}

	// log.Trace("TODO log successful creation with details") // TODO
}

// GetPaymentSession retrieves the payment session
func (service *PaymentService) GetPaymentSession(w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get(":payment_id")
	if id == "" {
		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	paymentResource, err := service.DAO.GetPaymentResource(id)
	if err != nil {
		if paymentResource != nil {
			log.Info(fmt.Sprintf("payment session not found. id: %s", id))
			w.WriteHeader(http.StatusForbidden)
			return
		}
		log.ErrorR(req, fmt.Errorf("error getting payment resource from db: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	costs, err := getCosts(w, req, paymentResource.Links.Resource, &service.Config)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting payment resource: [%v]", err))
		return
	}

	totalAmount, err := getTotalAmount(costs)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting amount from costs: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if totalAmount != paymentResource.Amount {
		log.Info(fmt.Sprintf("amount in payment resource [%s] different from db [%s] for id [%s].", totalAmount, paymentResource.Amount, paymentResource.ID))
		// TODO Expire payment session
		w.WriteHeader(http.StatusForbidden)
		return
	}

	paymentResource.Costs = *costs
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(paymentResource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		return
	}
}

func getTotalAmount(costs *[]models.CostResource) (string, error) {
	totalAmount := 0
	for _, cost := range *costs {
		amount, err := strconv.Atoi(cost.Amount)
		if err != nil {
			return "", err
		}
		totalAmount += amount
	}
	return strconv.Itoa(totalAmount), nil
}

func getCosts(w http.ResponseWriter, req *http.Request, resource string, cfg *config.Config) (*[]models.CostResource, error) {
	parsedURL, err := url.Parse(resource)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error parsing resource: [%v]", err))
		w.WriteHeader(http.StatusBadRequest)
		return nil, err
	}
	resourceDomain := strings.Join([]string{parsedURL.Scheme, parsedURL.Host}, "://")

	whitelist := strings.Split(cfg.DomainWhitelist, ",")
	matched := false
	for _, domain := range whitelist {
		if resourceDomain == domain {
			matched = true
			break
		}
	}
	if !matched {
		err = fmt.Errorf("invalid resource domain: %s", resourceDomain)
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

	costs := &[]models.CostResource{}
	err = json.Unmarshal(body, costs)
	if err != nil {
		log.ErrorR(resourceReq, fmt.Errorf("error reading Cost Resource: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return nil, err
	}
	return costs, nil
}

// Generates a string of 20 numbers made up of 7 random numbers, followed by 13 numbers derived from the current time
func generateID() (i string) {
	rand.Seed(time.Now().UTC().UnixNano())
	ranNumber := fmt.Sprintf("%07d", rand.Intn(9999999))
	millis := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	return ranNumber + millis
}
