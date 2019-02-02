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

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/transformers"
	"github.com/gorilla/mux"
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

const PaymentSessionKey = "payment_session"

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

// the start of this func is the responsibility of a payment create handler so split out
// // CreatePaymentSession creates a payment session and returns a journey URL for the calling app to redirect to
// func (service *PaymentService) CreatePaymentSession(w http.ResponseWriter, req *http.Request) {
// 	if req.Body == nil {
// 		log.ErrorR(req, fmt.Errorf("request body empty"))
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}

// 	requestDecoder := json.NewDecoder(req.Body)
// 	var incomingPaymentResourceRequest models.IncomingPaymentResourceRequest
// 	err := requestDecoder.Decode(&incomingPaymentResourceRequest)
// 	if err != nil {
// 		log.ErrorR(req, fmt.Errorf("request body invalid: [%v]", err))
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}

// if err = validatePaymentCreate(incomingPaymentResourceRequest); err != nil {
// 	log.ErrorR(req, fmt.Errorf("invalid POST request to create payment session: [%v]", err))
// 	w.WriteHeader(http.StatusBadRequest)
// 	return nil, err
// }

// this service func now just handles the incoming model. request is still passed in for other purposes e.g. log tracing, using models from request context - user auth
// however the response writer should not be available in any service level call so it has been removed from this func args.
// instead this func returns errors which can be converted into status codes by the controller - the error types and conversion will need more work going forwards
// but this split should be put in now

// CreatePaymentSession creates a payment session and returns a journey URL for the calling app to redirect to
func (service *PaymentService) CreatePaymentSession(req *http.Request, createResource models.IncomingPaymentResourceRequest) (*models.PaymentResourceRest, error) {

	// Get user details from context, put there by UserAuthenticationInterceptor
	userDetails, ok := req.Context().Value(helpers.UserDetailsKey).(models.AuthUserDetails)
	if !ok {
		err := fmt.Errorf("invalid AuthUserDetails in request context")
		log.ErrorR(req, err)
		return nil, err
	}

	// http status code may not matter here - if there is any error we can treat as internal server error for now
	costs, _, err := getCosts(createResource.Resource, &service.Config)
	if err != nil {
		err = fmt.Errorf("error getting payment resource: [%v]", err)
		log.ErrorR(req, err)
		return nil, err
	}

	totalAmount, err := getTotalAmount(costs)
	if err != nil {
		err = fmt.Errorf("error getting amount from costs: [%v]", err)
		log.ErrorR(req, err)
		return nil, err
	}

	// user details can now be retrieved from request context as their are handled by UserAuthenticationInterceptor
	// user := strings.Split(req.Header.Get("Eric-Authorised-User"), ";")
	// email := user[0]
	// var forename string
	// var surname string

	// for i := 1; i < len(user); i++ {
	// 	v := strings.Split(user[i], "=")
	// 	if v[0] == " forename" {
	// 		forename = v[1]
	// 	} else if v[0] == " surname" {
	// 		surname = v[1]
	// 	} else {
	// 		log.ErrorR(req, fmt.Errorf("unexpected format in Eric-Authorised-User: %s", user))
	// 		w.WriteHeader(http.StatusInternalServerError)
	// 		return nil, err
	// 	}
	// }

	//  Create payment session REST data from writable input fields and decorating with read only fields
	paymentResourceRest := &models.PaymentResourceRest{}
	paymentResourceRest.CreatedBy = models.CreatedByRest{
		ID:       userDetails.Id,
		Email:    userDetails.Email,
		Forename: userDetails.Forename,
		Surname:  userDetails.Surname,
	}
	paymentResourceRest.Amount = totalAmount
	// To match the format time is saved to mongo, e.g. "2018-11-22T08:39:16.782Z", truncate the time
	paymentResourceRest.CreatedAt = time.Now().Truncate(time.Millisecond)

	paymentResourceRest.Reference = createResource.Reference
	paymentResourceRest.Status = Pending.String()
	paymentResourceID := generateID()

	journeyURL := service.Config.PaymentsWebURL + "/payments/" + paymentResourceID + "/pay"
	paymentResourceRest.Links = models.PaymentLinksRest{
		Journey:  journeyURL,
		Resource: createResource.Resource,
		Self:     fmt.Sprintf("payments/%s", paymentResourceID),
	}

	// transform the complete REST model to a DB model before writing to the DB
	paymentResourceEntity := transformers.PaymentTransformer{}.TransformToDB(*paymentResourceRest)

	// set metadata fields on the DB model before writing
	paymentResourceEntity.ID = paymentResourceID
	paymentResourceEntity.State = createResource.State

	err = service.DAO.CreatePaymentResource(&paymentResourceEntity)

	if err != nil {
		err = fmt.Errorf("error writing to MongoDB: %v", err)
		log.ErrorR(req, err)
		return nil, err
	}

	return paymentResourceRest, nil
}

// the end of this func is the responsibility of a payment create handler so split out
// 	// Add data to response
// 	w.Header().Set("Content-Type", "application/json")
// 	w.Header().Set("Location", journeyURL)
// 	w.WriteHeader(http.StatusCreated)

// 	// response body contains fully decorated REST model
// 	err = json.NewEncoder(w).Encode(paymentResourceRest)
// 	if err != nil {
// 		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
// 		return
// 	}

// 	log.InfoR(req, "Successful POST request for new payment resource", log.Data{"payment_id": paymentResourceEntity.ID, "status": http.StatusCreated})
// }

// This func has been moved to the payments handler
// // GetPaymentSessionFromRequest retrieves the payment session
// func (service *PaymentService) GetPaymentSessionFromRequest(w http.ResponseWriter, req *http.Request) {
// 	vars := mux.Vars(req)
// 	id := vars["payment_id"]
// 	if id == "" {
// 		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
// 		w.WriteHeader(http.StatusBadRequest)
// 		return
// 	}

// 	paymentSession, httpStatus, err := (*PaymentService).GetPaymentSession(service, id)
// 	if err != nil {
// 		w.WriteHeader(httpStatus)
// 		log.ErrorR(req, err)
// 		return
// 	}

// 	paymentSessionResponse := transformers.PaymentTransformer{}.TransformToRest(*paymentSession)

// 	w.Header().Set("Content-Type", "application/json")

// 	err = json.NewEncoder(w).Encode(paymentSessionResponse)
// 	if err != nil {
// 		log.ErrorR(req, fmt.Errorf("error writing response: %v", err))
// 		return
// 	}

// 	log.InfoR(req, "Successfully GET request for payment resource: ", log.Data{"payment_id": id, "status": http.StatusCreated})
// }

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
	var PaymentResourceUpdateData models.PaymentResourceRest
	err := requestDecoder.Decode(&PaymentResourceUpdateData)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("request body invalid: [%v]", err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var PaymentResourceUpdate models.PaymentResourceDB
	PaymentResourceUpdate = transformers.PaymentTransformer{}.TransformToDB(PaymentResourceUpdateData)

	httpStatus, err := service.patchPaymentSession(id, PaymentResourceUpdate)
	if err != nil {
		w.WriteHeader(httpStatus)
		log.ErrorR(req, err)
		return
	}

	log.InfoR(req, "Successful PATCH request for payment resource", log.Data{"payment_id": id, "status": http.StatusOK})
}

func (service *PaymentService) patchPaymentSession(id string, PaymentResourceUpdate models.PaymentResourceDB) (int, error) {

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

// UpdatePaymentStatus updates the Status in the Payment Session.
func (service *PaymentService) UpdatePaymentStatus(s models.StatusResponse, p models.PaymentResourceDB) error {
	p.Data.Status = s.Status
	_, err := service.patchPaymentSession(p.ID, p)

	if err != nil {
		return fmt.Errorf("error updating payment status: [%s]", err)
	}
	return nil
}

// GetPaymentSession retrieves the payment session witht he given id from the database
func (service *PaymentService) GetPaymentSession(req *http.Request, id string) (*models.PaymentResourceRest, int, error) {
	paymentResource, err := service.DAO.GetPaymentResource(id)
	if err != nil {
		err := fmt.Errorf("error getting payment resource from db: [%v]", err)
		log.ErrorR(req, err)
		return nil, http.StatusInternalServerError, err
	}

	if paymentResource == nil {
		log.TraceR(req, "payment session not found", log.Data{"payment_id": id})
		return nil, http.StatusNotFound, nil
	}

	costs, httpStatus, err := getCosts(paymentResource.Data.Links.Resource, &service.Config)
	if err != nil {
		err := fmt.Errorf("error getting payment costs: [%v]", err)
		log.ErrorR(req, err)
		return nil, httpStatus, err
	}

	totalAmount, err := getTotalAmount(costs)
	if err != nil {
		err := fmt.Errorf("error getting amount from costs: [%v]", err)
		log.ErrorR(req, err)
		return nil, http.StatusInternalServerError, err
	}

	if totalAmount != paymentResource.Data.Amount {
		// TODO Expire payment session
		err := fmt.Errorf("amount in payment resource [%s] different from db [%s] for id [%s]", totalAmount, paymentResource.Data.Amount, paymentResource.ID)
		log.ErrorR(req, err)
		return nil, http.StatusForbidden, err
	}

	paymentResource.Data.Costs = *costs

	// transform the complete DB model to a Rest model before returning to the caller
	// this is because the DB model is internal to the service layer, nothing dealing with the service ayer should ever know about the internal storage model
	paymentResourceRest := transformers.PaymentTransformer{}.TransformToRest(paymentResource.Data)

	return &paymentResourceRest, http.StatusOK, nil
}

func getTotalAmount(costs *[]models.CostResourceDB) (string, error) {
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

func getCosts(resource string, cfg *config.Config) (*[]models.CostResourceDB, int, error) {
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

	costs := &[]models.CostResourceDB{}
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

func validateCosts(costs *[]models.CostResourceDB) error {
	validate := validator.New()
	for _, cost := range *costs {
		err := validate.Struct(cost)
		if err != nil {
			return err
		}
	}
	return nil
}

// func validatePaymentCreate(incomingPaymentResourceRequest models.IncomingPaymentResourceRequest) error {
// 	validate := validator.New()
// 	err := validate.Struct(incomingPaymentResourceRequest)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
