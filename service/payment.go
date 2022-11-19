package service

import (
	"crypto/sha512"
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

	"github.com/companieshouse/chs.go/authentication"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/companieshouse/payments.api.ch.gov.uk/transformers"
	"github.com/shopspring/decimal"
	"gopkg.in/go-playground/validator.v9"
)

// PaymentService contains the DAO for db access
type PaymentService struct {
	DAO              dao.DAO
	Config           config.Config
	SecureCostsRegex *regexp.Regexp
	Client           PayPalSDK
}

// PaymentStatus Enum Type
type PaymentStatus int

// PaymentSessionKind is the value stored in the payment resource kind field
const PaymentSessionKind = "payment-session#payment-session"

// Enumeration containing all possible payment statuses
const (
	Pending PaymentStatus = 1 + iota
	InProgress
	Paid
	NoFunds
	Failed
	Expired
	PendingRefund
	RefundRequested
)

// String representation of payment statuses
var paymentStatuses = [...]string{
	"pending",
	"in-progress",
	"paid",
	"no-funds",
	"failed",
	"expired",
	"refund-pending",
	"refund-requested",
}

func (paymentStatus PaymentStatus) String() string {
	return paymentStatuses[paymentStatus-1]
}

// CreatePaymentSession creates a payment session and returns a journey URL for the calling app to redirect to
func (service *PaymentService) CreatePaymentSession(req *http.Request, createResource models.IncomingPaymentResourceRequest) (*models.PaymentResourceRest, ResponseType, error) {
	log.TraceR(req, "create payment session", log.Data{"create_resource": createResource})
	err := validateIncomingPayment(createResource, &service.Config)
	if err != nil {
		err = fmt.Errorf("invalid incoming payment: [%v]", err)
		log.ErrorR(req, err)
		return nil, InvalidData, err
	}

	// Get user details from context, put there by UserAuthenticationInterceptor
	userDetails, ok := req.Context().Value(authentication.ContextKeyUserDetails).(authentication.AuthUserDetails)
	if !ok {
		err = fmt.Errorf("invalid AuthUserDetails in request context")
		log.ErrorR(req, err)
		return nil, Error, err
	}

	costs, costsResponseType, err := getCosts(createResource.Resource, &service.Config, service.SecureCostsRegex)
	if err != nil {
		err = fmt.Errorf("error getting payment resource: [%v]", err)
		log.ErrorR(req, err)
		return nil, costsResponseType, err
	}

	totalAmount, err := getTotalAmount(&costs.Costs)
	if err != nil {
		err = fmt.Errorf("error getting amount from costs: [%v]", err)
		log.ErrorR(req, err)
		return nil, Error, err
	}

	//  Create payment session REST data from writable input fields and decorating with read only fields
	paymentResourceRest := models.PaymentResourceRest{}
	paymentResourceRest.CreatedBy = models.CreatedByRest{
		ID:       userDetails.ID,
		Email:    userDetails.Email,
		Forename: userDetails.Forename,
		Surname:  userDetails.Surname,
	}
	paymentResourceRest.Costs = costs.Costs
	paymentResourceRest.Description = costs.Description
	paymentResourceRest.CompanyNumber = costs.CompanyNumber
	paymentResourceRest.Amount = totalAmount
	// To match the format time is saved to mongo, e.g. "2018-11-22T08:39:16.782Z", truncate the time
	paymentResourceRest.CreatedAt = time.Now().Truncate(time.Millisecond)

	paymentMethods := make(map[string]bool)
	for _, c := range costs.Costs {
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

	// If auth is API Key, add suffix to journey URL
	if authentication.GetAuthorisedIdentityType(req) == authentication.APIKeyIdentityType {
		journeyURL += "/api-key"
	}

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
		err = fmt.Errorf("error writing to DB: %v", err)
		log.ErrorR(req, err)
		return nil, Error, err
	}

	return &paymentResourceRest, Success, nil
}

// PatchPaymentSession updates an existing payment session with the data provided from the Rest model
func (service *PaymentService) PatchPaymentSession(req *http.Request, id string, paymentResourceUpdateRest models.PaymentResourceRest) (ResponseType, error) {
	PaymentResourceUpdate := transformers.PaymentTransformer{}.TransformToDB(paymentResourceUpdateRest)
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

// StoreExternalPaymentStatusDetails stores the URI and the ID of the external payment session in the metadata
func (service *PaymentService) StoreExternalPaymentStatusDetails(id, externalPaymentStatusURI, externalPaymentStatusID string) error {
	PaymentResourceUpdate := models.PaymentResourceDB{
		ExternalPaymentStatusURI: externalPaymentStatusURI,
		ExternalPaymentStatusID:  externalPaymentStatusID,
	}
	err := service.DAO.PatchPaymentResource(id, &PaymentResourceUpdate)
	if err != nil {
		err = fmt.Errorf("error storing the External Payment Status Details against the payment session: [%v]", err)
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

	costs, costsResponseType, err := getCosts(paymentResource.Data.Links.Resource, &service.Config, service.SecureCostsRegex)
	if err != nil {
		err = fmt.Errorf("error getting payment resource: [%v]", err)
		log.ErrorR(req, err)
		return nil, costsResponseType, err
	}

	totalAmount, err := getTotalAmount(&costs.Costs)
	if err != nil {
		err = fmt.Errorf("error getting amount from costs: [%v]", err)
		log.ErrorR(req, err)
		return nil, Error, err
	}

	if totalAmount != paymentResource.Data.Amount {
		err = fmt.Errorf("amount in payment resource [%s] different from db [%s] for id [%s]", totalAmount, paymentResource.Data.Amount, paymentResource.ID)
		log.ErrorR(req, err)
		return nil, Forbidden, err
	}

	paymentResourceRest := transformers.PaymentTransformer{}.TransformToRest(*paymentResource)
	paymentResourceRest.Costs = costs.Costs
	paymentResourceRest.Description = costs.Description

	return &paymentResourceRest, Success, nil
}

func getTotalAmount(costs *[]models.CostResourceRest) (string, error) {
	r := regexp.MustCompile(`^\d+(\.\d{2})?$`)
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

func getCosts(resource string, cfg *config.Config, secAppCostsRegex *regexp.Regexp) (*models.CostsRest, ResponseType, error) {

	resourceReq, err := http.NewRequest("GET", resource, nil)
	if err != nil {
		return nil, Error, fmt.Errorf("failed to create Resource Request: [%v]", err)
	}

	resourceReq.SetBasicAuth(cfg.ChsAPIKey, "")

	var client http.Client
	resp, err := client.Do(resourceReq)
	if err != nil {
		return nil, Error, fmt.Errorf("error getting Cost Resource: [%v]", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		if secAppCostsRegex.MatchString(resource) {
			err = errors.New("error getting Cost Resource - Gone: [410]")
			log.ErrorR(resourceReq, err)
			return nil, CostsGone, err
		}

		err = errors.New("error getting Cost Resource - Not Found: [404]")
		log.ErrorR(resourceReq, err)
		return nil, CostsNotFound, err
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error getting Cost Resource - status code: [%v]", resp.StatusCode)
		log.ErrorR(resourceReq, err)
		return nil, InvalidData, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, Error, fmt.Errorf("error reading Cost Resource: [%v]", err)
	}

	costs := &models.CostsRest{}
	err = json.Unmarshal(body, costs)
	if err != nil {
		return nil, InvalidData, fmt.Errorf("error reading Cost Resource: [%v]", err)
	}

	if err = validateCosts(&costs.Costs); err != nil {
		log.ErrorR(resourceReq, fmt.Errorf("invalid Cost Resource: [%v]", err))
		return nil, InvalidData, err
	}

	return costs, Success, nil
}

// Generates a string of 15 alpha numeric characters. this needs to be less than 16 characters as these id's are also
// being sent to E5 (our finance system) when paying for late filing penalties. Although this restriction is currently
// being imposed on us by an external system, there is enough entropy here that makes collisions highly unlikely, with
// a total range being 15**62.
//
// **If you change the implementation of this test, you must run the utility test `go test ./... -run 'Util'**
func generateID() string {
	idLength := 15
	rand.Seed(time.Now().UTC().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz1234567890")
	id := make([]rune, idLength)
	for i := 0; i < idLength; i++ {
		id[i] = chars[rand.Intn(len(chars))]
	}
	return string(id)
}

// generateEtag generates a random etag which is generated on every write action on the payment session
func generateEtag() string {
	// Get a random number and the time in seconds and milliseconds
	rand.Seed(time.Now().UTC().UnixNano())
	randomNumber := fmt.Sprintf("%07d", rand.Intn(9999999))
	timeInMillis := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	timeInSeconds := strconv.FormatInt(time.Now().UnixNano()/int64(time.Second), 10)
	// Calculate a SHA-512 truncated digest
	shaDigest := sha512.New512_224()
	shaDigest.Write([]byte(randomNumber + timeInMillis + timeInSeconds))
	sha1_hash := hex.EncodeToString(shaDigest.Sum(nil))
	return sha1_hash
}

func validateIncomingPayment(incomingPaymentResourceRequest models.IncomingPaymentResourceRequest, cfg *config.Config) error {
	validate := validator.New()
	err := validate.Struct(incomingPaymentResourceRequest)
	if err != nil {
		return err
	}

	parsedURL, err := url.Parse(incomingPaymentResourceRequest.Resource)
	if err != nil {
		return err
	}
	resourceDomain := strings.Join([]string{parsedURL.Scheme, parsedURL.Host}, "://")

	allowList := strings.Split(cfg.DomainAllowList, ",")
	matched := false
	for _, domain := range allowList {
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

func IsExpired(paymentSession models.PaymentResourceRest, cfg *config.Config) (bool, error) {
	expiryTimeInMinutes, err := strconv.Atoi(cfg.ExpiryTimeInMinutes)
	if err != nil {
		return false, err
	}
	return paymentSession.CreatedAt.Add(time.Minute * time.Duration(expiryTimeInMinutes)).Before(time.Now()), nil
}
