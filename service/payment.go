package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/shopspring/decimal"
	"gopkg.in/go-playground/validator.v9"
)

// PaymentService contains the DAO for db access
type PaymentService struct {
	DAO    dao.DAO
	Config config.Config
}

// PaymentStatus Enum Type
type PaymentStatus int

// Enumeration containing all possible payment statuses
const (
	Pending PaymentStatus = 1 + iota
	InProgress
	Paid
	NoFunds
	Failed
)

// String representation of payment statuses
var paymentStatuses = [...]string{
	"pending",
	"in-progress",
	"paid",
	"no-funds ",
	"failed",
}

func (paymentStatus PaymentStatus) String() string {
	return paymentStatuses[paymentStatus-1]
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

	costs, httpStatus, err := getCosts(incomingPaymentResourceRequest.Resource, &service.Config)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting payment resource: [%v]", err))
		w.WriteHeader(httpStatus)
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
	paymentResource.Data.CreatedBy = models.CreatedBy{
		ID:       req.Header.Get("Eric-Identity"),
		Email:    email,
		Forename: forename,
		Surname:  surname,
	}
	paymentResource.Data.Amount = totalAmount
	// To match the format time is saved to mongo, e.g. "2018-11-22T08:39:16.782Z", truncate the time
	paymentResource.Data.CreatedAt = time.Now().Truncate(time.Millisecond)

	paymentResource.Data.Reference = incomingPaymentResourceRequest.Reference
	paymentResource.State = incomingPaymentResourceRequest.State
	paymentResource.RedirectURI = incomingPaymentResourceRequest.RedirectURI
	paymentResource.Data.Status = Pending.String()
	paymentResource.ID = generateID()

	journeyURL := service.Config.PaymentsWebURL + "/payments/" + paymentResource.ID + "/pay"
	paymentResource.Data.Links = models.Links{
		Journey:  journeyURL,
		Resource: incomingPaymentResourceRequest.Resource,
		Self:     fmt.Sprintf("payments/%s", paymentResource.ID),
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

	err = json.NewEncoder(w).Encode(paymentResource.Data)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		return
	}

	log.InfoR(req, "Successful POST request for new payment resource", log.Data{"payment_id": paymentResource.ID, "status": http.StatusCreated})
}

// GetPaymentSession retrieves the payment session
func (service *PaymentService) GetPaymentSession(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["payment_id"]
	if id == "" {
		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	paymentSession, httpStatus, err := (*PaymentService).getPaymentSession(service, id)
	if err != nil {
		w.WriteHeader(httpStatus)
		log.ErrorR(req, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(paymentSession)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
		return
	}

	log.InfoR(req, "Successfully GET request for payment resource: ", log.Data{"payment_id": id, "status": http.StatusCreated})
}

// PatchPaymentSession patches and updates the payment session
func (service *PaymentService) PatchPaymentSession(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["payment_id"]
	if id == "" {
		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Body == nil {
		log.ErrorR(req, fmt.Errorf("request body empty"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	requestDecoder := json.NewDecoder(req.Body)
	var PaymentResourceUpdateData models.PaymentResourceData
	err := requestDecoder.Decode(&PaymentResourceUpdateData)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("request body invalid: [%v]", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var PaymentResourceUpdate models.PaymentResource
	PaymentResourceUpdate.Data = PaymentResourceUpdateData

	httpStatus, err := service.patchPaymentSession(id, PaymentResourceUpdate)
	if err != nil {
		w.WriteHeader(httpStatus)
		log.ErrorR(req, err)
		return
	}

	log.InfoR(req, "Successful PATCH request for payment resource", log.Data{"payment_id": id, "status": http.StatusOK})
}

func (service *PaymentService) patchPaymentSession(id string, PaymentResourceUpdate models.PaymentResource) (int, error) {

	if PaymentResourceUpdate.Data.PaymentMethod == "" && PaymentResourceUpdate.Data.Status == "" && PaymentResourceUpdate.ExternalPaymentStatusURI == "" {
		return http.StatusBadRequest, fmt.Errorf("no valid fields for the patch request has been supplied for resource [%s]", id)
	}

	err := service.DAO.PatchPaymentResource(id, &PaymentResourceUpdate)
	if err != nil {
		if err.Error() == "not found" {
			return http.StatusForbidden, fmt.Errorf("could not find payment resource to patch")
		}
		return http.StatusInternalServerError, fmt.Errorf("error patching payment session on database: [%v]", err)
	}

	return http.StatusOK, nil
}

func (service *PaymentService) UpdatePaymentStatus(s models.StatusResponse, p models.PaymentResource) error {
	p.Data.Status = s.Status
	_, err := service.patchPaymentSession(p.ID, p)

	if err != nil {
		return fmt.Errorf("error updating payment status: [%s]", err)
	}
	return nil
}

func (service *PaymentService) getPaymentSession(id string) (*models.PaymentResourceData, int, error) {
	paymentResource, err := service.DAO.GetPaymentResource(id)
	if paymentResource == nil {
		return nil, http.StatusForbidden, fmt.Errorf("payment session not found. id: %s", id)
	}
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error getting payment resource from db: [%v]", err)
	}

	costs, httpStatus, err := getCosts(paymentResource.Data.Links.Resource, &service.Config)
	if err != nil {
		return nil, httpStatus, fmt.Errorf("error getting payment resource: [%v]", err)
	}

	totalAmount, err := getTotalAmount(costs)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error getting amount from costs: [%v]", err)
	}

	if totalAmount != paymentResource.Data.Amount {
		// TODO Expire payment session
		return nil, http.StatusForbidden, fmt.Errorf("amount in payment resource [%s] different from db [%s] for id [%s]", totalAmount, paymentResource.Data.Amount, paymentResource.ID)
	}

	paymentResource.Data.Costs = *costs

	return &paymentResource.Data, http.StatusOK, nil
}

func getTotalAmount(costs *[]models.CostResource) (string, error) {
	r, err := regexp.Compile(`^\d+(\.\d{2})?$`)
	if err != nil {
		return "", err
	}
	var totalAmount decimal.Decimal
	for _, cost := range *costs {
		matched := r.MatchString(cost.Amount)
		if !matched {
			return "", fmt.Errorf("amount [%s] format incorrect", cost.Amount)
		}

		amount, _ := decimal.NewFromString(cost.Amount)
		totalAmount = totalAmount.Add(amount)
	}
	return totalAmount.StringFixed(2), nil
}

func getCosts(resource string, cfg *config.Config) (*[]models.CostResource, int, error) {
	err := validateResource(resource, cfg)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	resourceReq, err := http.NewRequest("GET", resource, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create Resource Request: [%v]", err)
	}

	var client http.Client
	resp, err := client.Do(resourceReq)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error getting Cost Resource: [%v]", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = errors.New("error getting Cost Resource")
		log.ErrorR(resourceReq, err)
		return nil, http.StatusBadRequest, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error reading Cost Resource: [%v]", err)
	}

	costs := &[]models.CostResource{}
	err = json.Unmarshal(body, costs)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("error reading Cost Resource: [%v]", err)
	}

	if err = validateCosts(costs); err != nil {
		log.ErrorR(resourceReq, fmt.Errorf("invalid Cost Resource: [%v]", err))
		return nil, http.StatusBadRequest, err
	}

	return costs, http.StatusOK, nil
}

// Generates a string of 20 numbers made up of 7 random numbers, followed by 13 numbers derived from the current time
func generateID() (i string) {
	rand.Seed(time.Now().UTC().UnixNano())
	ranNumber := fmt.Sprintf("%07d", rand.Intn(9999999))
	millis := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	return ranNumber + millis
}

func validateResource(resource string, cfg *config.Config) error {
	parsedURL, err := url.Parse(resource)
	if err != nil {
		return err
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
		return err
	}
	return err
}

func validateCosts(costs *[]models.CostResource) error {
	validate := validator.New()
	for _, cost := range *costs {
		err := validate.Struct(cost)
		if err != nil {
			return err
		}
	}
	return nil
}
