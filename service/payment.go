package service

import (
	"crypto/sha1"
	"encoding/hex"
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

// PaymentSessionKind constant is the value stored in the payment resource kind field
const PaymentSessionKind = "payment-session#payment-session"
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

// CreatePaymentSession creates a payment session and returns a journey URL for the calling app to redirect to
func (service *PaymentService) CreatePaymentSession(req *http.Request, createResource models.IncomingPaymentResourceRequest) (*models.PaymentResourceRest, ResponseType, error) {

	// Get user details from context, put there by UserAuthenticationInterceptor
	userDetails, ok := req.Context().Value(helpers.ContextKeyUserDetails).(models.AuthUserDetails)
	if !ok {
		err := fmt.Errorf("invalid AuthUserDetails in request context")
		log.ErrorR(req, err)
		return nil, InvalidData, err
	}

	costs, costsResponseType, err := getCosts(createResource.Resource, &service.Config)
	if err != nil {
		err = fmt.Errorf("error getting payment resource: [%v]", err)
		log.ErrorR(req, err)
		return nil, costsResponseType, err
	}

	totalAmount, err := getTotalAmount(costs)
	if err != nil {
		err = fmt.Errorf("error getting amount from costs: [%v]", err)
		log.ErrorR(req, err)
		return nil, Error, err
	}

	//  Create payment session REST data from writable input fields and decorating with read only fields
	paymentResourceRest := models.PaymentResourceRest{}
	paymentResourceRest.CreatedBy = models.CreatedByRest{
		ID:       userDetails.Id,
		Email:    userDetails.Email,
		Forename: userDetails.Forename,
		Surname:  userDetails.Surname,
	}
	paymentResourceRest.Costs = *costs
	paymentResourceRest.Amount = totalAmount
	// To match the format time is saved to mongo, e.g. "2018-11-22T08:39:16.782Z", truncate the time
	paymentResourceRest.CreatedAt = time.Now().Truncate(time.Millisecond)

	paymentMethods := make(map[string]bool)
	for _, c := range *costs {
		for _, cc := range c.AvailablePaymentMethods {
			paymentMethods[cc] = true
		}
	}
	for k := range paymentMethods {
		paymentResourceRest.AvailablePaymentMethods = append(paymentResourceRest.AvailablePaymentMethods, k)
	}

	paymentResourceRest.Reference = createResource.Reference
	paymentResourceRest.Status = Pending.String()
	paymentResourceRest.Kind = PaymentSessionKind
	paymentResourceRest.Etag = generateEtag()
	paymentResourceID := generateID()

	journeyURL := service.Config.PaymentsWebURL + "/payments/" + paymentResourceID + "/pay"
	paymentResourceRest.Links = models.PaymentLinksRest{
		Journey:  journeyURL,
		Resource: createResource.Resource,
		Self:     fmt.Sprintf("payments/%s", paymentResourceID),
	}

	// transform the complete REST model to a DB model before writing to the DB
	paymentResourceEntity := transformers.PaymentTransformer{}.TransformToDB(paymentResourceRest)

	// set metadata fields on the DB model before writing
	paymentResourceEntity.ID = paymentResourceID
	paymentResourceEntity.State = createResource.State
	paymentResourceEntity.RedirectURI = createResource.RedirectURI

	err = service.DAO.CreatePaymentResource(&paymentResourceEntity)

	if err != nil {
		err = fmt.Errorf("error writing to MongoDB: %v", err)
		log.ErrorR(req, err)
		return nil, Error, err
	}

	return &paymentResourceRest, Success, nil
}

// PatchPaymentSession updates an existing payment session with the data provided from the Rest model
func (service *PaymentService) PatchPaymentSession(req *http.Request, id string, PaymentResourceUpdateRest models.PaymentResourceRest) (ResponseType, error) {
	PaymentResourceUpdate := transformers.PaymentTransformer{}.TransformToDB(PaymentResourceUpdateRest)
	PaymentResourceUpdate.Data.Etag = generateEtag()

	paymentSession, response, err := service.GetPaymentSession(req, id)
	if err != nil {
		err = fmt.Errorf("error getting payment resource to patch: [%v]", err)
		log.ErrorR(req, err)
		return response, err
	}
	if paymentSession.Status == Pending.String() {
		PaymentResourceUpdate.Data.Status = InProgress.String()
	}

	err = service.DAO.PatchPaymentResource(id, &PaymentResourceUpdate)
	if err != nil {
		err = fmt.Errorf("error patching payment session on database: [%v]", err)
		log.Error(err)
		return Error, err
	}
	return Success, nil
}

// StoreExternalPaymentStatusURI stores a new value in the payment resource metadata for the ExternalPaymentStatusURI
func (service *PaymentService) StoreExternalPaymentStatusURI(req *http.Request, id string, externalPaymentStatusURI string) error {
	PaymentResourceUpdate := models.PaymentResourceDB{
		ExternalPaymentStatusURI: externalPaymentStatusURI,
	}
	err := service.DAO.PatchPaymentResource(id, &PaymentResourceUpdate)
	if err != nil {
		err = fmt.Errorf("error storing ExternalPaymentStatusURI on payment session: [%v]", err)
		log.ErrorR(req, err)
		return err
	}
	return nil
}

// GetPaymentSession retrieves the payment session with the given ID from the database
func (service *PaymentService) GetPaymentSession(req *http.Request, id string) (*models.PaymentResourceRest, ResponseType, error) {
	paymentResource, err := service.DAO.GetPaymentResource(id)
	if err != nil {
		err = fmt.Errorf("error getting payment resource from db: [%v]", err)
		log.ErrorR(req, err)
		return nil, Error, err
	}
	if paymentResource == nil {
		log.TraceR(req, "payment session not found", log.Data{"payment_id": id})
		return nil, NotFound, nil
	}

	costs, costsResponseType, err := getCosts(paymentResource.Data.Links.Resource, &service.Config)
	if err != nil {
		err = fmt.Errorf("error getting payment resource: [%v]", err)
		log.ErrorR(req, err)
		return nil, costsResponseType, err
	}

	totalAmount, err := getTotalAmount(costs)
	if err != nil {
		err = fmt.Errorf("error getting amount from costs: [%v]", err)
		log.ErrorR(req, err)
		return nil, Error, err
	}

	if totalAmount != paymentResource.Data.Amount {
		// TODO Expire payment session
		err = fmt.Errorf("amount in payment resource [%s] different from db [%s] for id [%s]", totalAmount, paymentResource.Data.Amount, paymentResource.ID)
		log.ErrorR(req, err)
		return nil, Forbidden, err
	}

	paymentResourceRest := transformers.PaymentTransformer{}.TransformToRest(*paymentResource)
	paymentResourceRest.Costs = *costs

	return &paymentResourceRest, Success, nil
}

func getTotalAmount(costs *[]models.CostResourceRest) (string, error) {
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

func getCosts(resource string, cfg *config.Config) (*[]models.CostResourceRest, ResponseType, error) {
	err := validateResource(resource, cfg)
	if err != nil {
		return nil, InvalidData, err
	}

	resourceReq, err := http.NewRequest("GET", resource, nil)
	if err != nil {
		return nil, Error, fmt.Errorf("failed to create Resource Request: [%v]", err)
	}

	var client http.Client
	resp, err := client.Do(resourceReq)
	if err != nil {
		return nil, Error, fmt.Errorf("error getting Cost Resource: [%v]", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = errors.New("error getting Cost Resource")
		log.ErrorR(resourceReq, err)
		return nil, InvalidData, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, Error, fmt.Errorf("error reading Cost Resource: [%v]", err)
	}

	costs := &[]models.CostResourceRest{}
	err = json.Unmarshal(body, costs)
	if err != nil {
		return nil, InvalidData, fmt.Errorf("error reading Cost Resource: [%v]", err)
	}

	if err = validateCosts(costs); err != nil {
		log.ErrorR(resourceReq, fmt.Errorf("invalid Cost Resource: [%v]", err))
		return nil, InvalidData, err
	}

	return costs, Success, nil
}

// Generates a string of 20 numbers made up of 7 random numbers, followed by 13 numbers derived from the current time
func generateID() (i string) {
	rand.Seed(time.Now().UTC().UnixNano())
	ranNumber := fmt.Sprintf("%07d", rand.Intn(9999999))
	millis := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	return ranNumber + millis
}

// generateEtag generates a random etag which is generated on every write action on the payment session
func generateEtag() string {
	// Get a random number and the time in seconds and milliseconds
	rand.Seed(time.Now().UTC().UnixNano())
	randomNumber := fmt.Sprintf("%07d", rand.Intn(9999999))
	timeInMillis := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	timeInSeconds := strconv.FormatInt(time.Now().UnixNano()/int64(time.Second), 10)
	// Calculate a SHA-1 digest
	shaDigest := sha1.New()
	shaDigest.Write([]byte(randomNumber + timeInMillis + timeInSeconds))
	sha1_hash := hex.EncodeToString(shaDigest.Sum(nil))
	return sha1_hash
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
	}
	return err
}

func validateCosts(costs *[]models.CostResourceRest) error {
	validate := validator.New()
	for _, cost := range *costs {
		err := validate.Struct(cost)
		if err != nil {
			return err
		}
	}
	return nil
}
