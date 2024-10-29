package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/plutov/paypal/v4"
)

var govPayRequestError = "error generating request for GovPay: [%s]"
var govPayHeaderError = "error adding GovPay headers: [%s]"
var govPayStatusError = "error status [%v] back from GovPay: [%s]"

// GovPayService handles the specific functionality of integrating GovPay provider into Payment Sessions
type GovPayService struct {
	PaymentService PaymentService
}

// CheckPaymentProviderStatus checks the status of the payment with GovPay
func (gp *GovPayService) CheckPaymentProviderStatus(paymentResource *models.PaymentResourceRest) (*models.StatusResponse, string, ResponseType, error) {
	govPayResponse, err := callGovPay(gp, paymentResource)
	if err != nil {
		return nil, "", Error, err
	}
	state := govPayResponse.State

	if state.Finished && state.Status == "success" {
		return &models.StatusResponse{Status: "paid"}, govPayResponse.ProviderID, Success, nil
	} else if state.Finished && state.Code == "P0030" {
		return &models.StatusResponse{Status: "cancelled"}, "", Success, nil
	} else if !state.Finished && state.Status == "created" {
		/*
			handle payment 'not yet finished' response from GovPay:
				"state": {
					"status": "created",
					"finished": false
				}
		*/
		return &models.StatusResponse{Status: "paid"}, govPayResponse.ProviderID, Created, nil
	}
	return &models.StatusResponse{Status: "failed"}, "", Error, nil
}

// CreatePaymentAndGenerateNextURL creates a gov pay session linked to the given payment session and stores the required details on the payment session
func (gp *GovPayService) CreatePaymentAndGenerateNextURL(req *http.Request, paymentResource *models.PaymentResourceRest) (string, ResponseType, error) {
	var govPayRequest models.OutgoingGovPayRequest

	amountToPay, err := convertToPenceFromDecimal(paymentResource.Amount)
	if err != nil {
		return "", Error, fmt.Errorf("error converting amount to pay to pence: [%s]", err)
	}

	govPayRequest.Amount = amountToPay
	if paymentResource.CreatedBy.Email != "" {
		govPayRequest.Email = paymentResource.CreatedBy.Email
	}
	govPayRequest.Description = "Companies House Payment" // Hard-coded value for payment screens
	govPayRequest.Reference = paymentResource.MetaData.ID
	govPayRequest.ReturnURL = fmt.Sprintf("%s/callback/payments/govpay/%s", gp.PaymentService.Config.PaymentsAPIURL, paymentResource.MetaData.ID)

	// Add metadata fields to send to Gov.UK Pay
	// https://docs.payments.service.gov.uk/custom_metadata/#add-metadata-to-a-payment
	govPayRequest.Metadata.CompanyNumber = paymentResource.CompanyNumber

	// Product Information is a comma separated string, truncated to 100 characters
	var productTypes []string
	for _, cost := range paymentResource.Costs {
		productTypes = append(productTypes, cost.ProductType)
	}
	productInformation := strings.Join(productTypes, ",")
	productInformation = fmt.Sprintf("%.100s", productInformation)
	govPayRequest.Metadata.ProductInformation = productInformation

	log.TraceR(req, "performing gov pay request", log.Data{"gov_pay_request_data": govPayRequest})

	requestBody, err := json.Marshal(govPayRequest)
	if err != nil {
		return "", Error, fmt.Errorf("error reading GovPayRequest: [%s]", err)
	}

	request, err := http.NewRequest("POST", gp.PaymentService.Config.GovPayURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", Error, fmt.Errorf(govPayRequestError, err)
	}

	err = addGovPayHeaders(request, paymentResource, gp)
	if err != nil {
		return "", InvalidData, fmt.Errorf(govPayHeaderError, err)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", Error, fmt.Errorf("error sending request to GovPay to start payment session: [%s]", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", Error, fmt.Errorf("error reading response from GovPay: [%s]", err)
	}

	govPayResponse := &models.IncomingGovPayResponse{}
	err = json.Unmarshal(body, govPayResponse)
	if err != nil {
		return "", Error, fmt.Errorf("error reading response from GovPay: [%s]", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return "", Error, fmt.Errorf(govPayStatusError, resp.StatusCode, govPayResponse.Description)
	}

	err = gp.PaymentService.StoreExternalPaymentStatusDetails(paymentResource.MetaData.ID, govPayResponse.GovPayLinks.Self.HREF, govPayResponse.PaymentID)
	if err != nil {
		return "", Error, fmt.Errorf("error storing GovPay external payment details for payment session: [%s]", err)
	}

	return govPayResponse.GovPayLinks.NextURL.HREF, Success, nil
}

// GetPaymentDetails gets the details of a GovPay payment
func (gp *GovPayService) GetPaymentDetails(paymentResource *models.PaymentResourceRest) (*models.PaymentDetails, ResponseType, error) {

	govPayResponse, err := callGovPay(gp, paymentResource)
	if err != nil {
		return nil, Error, err
	}

	paymentDetails := &models.PaymentDetails{
		CardType:          govPayResponse.CardBrand,
		ExternalPaymentID: govPayResponse.PaymentID,
		TransactionDate:   govPayResponse.CreatedDate,
		ProviderID:        govPayResponse.ProviderID,
	}

	if govPayResponse.State.Finished && govPayResponse.State.Status == "success" {
		paymentDetails.PaymentStatus = "accepted"
	} else if govPayResponse.State.Finished && govPayResponse.State.Code == "P0010" {
		paymentDetails.PaymentStatus = "rejected"
	}

	return paymentDetails, Success, nil
}

// GetPaymentStatus gets the status of a GovPay payment
// https://docs.payments.service.gov.uk/api_reference/#payment-status-lifecycle
func (gp *GovPayService) GetPaymentStatus(paymentResource *models.PaymentResourceRest) (finished bool, status string, providerID string, err error) {

	govPayResponse, err := callGovPay(gp, paymentResource)
	if err != nil {
		return false, "", "", err
	}

	if !govPayResponse.State.Finished {
		return govPayResponse.State.Finished, govPayResponse.State.Status, "", nil
	}

	if govPayResponse.State.Status == "failed" {
		errorMappings := map[string]string{
			"P0010": "payment-method-rejected",
			"P0020": "payment-expired",
			"P0030": "payment-cancelled-by-user",
			"P0040": "payment-cancelled-by-service",
			"P0050": "payment-provider-error",
		}
		return govPayResponse.State.Finished, govPayResponse.State.Status + "_" + errorMappings[govPayResponse.State.Code], "", nil
	}

	if govPayResponse.State.Status == "success" {
		return govPayResponse.State.Finished, "paid", govPayResponse.ProviderID, nil
	}

	return govPayResponse.State.Finished, govPayResponse.State.Status, "", nil
}

// GetRefundSummary gets refund summary of a GovPay payment
func (gp *GovPayService) GetRefundSummary(req *http.Request, id string) (*models.PaymentResourceRest, *models.RefundSummary, ResponseType, error) {
	// Get PaymentSession for the GovPay call
	paymentSession, response, err := gp.PaymentService.GetPaymentSession(req, id)
	if err != nil {
		err = fmt.Errorf("error getting payment resource: [%v]", err)
		log.ErrorR(req, err)
		return nil, nil, response, err
	}

	if response == NotFound {
		err = fmt.Errorf("error getting payment resource")
		log.ErrorR(req, err)

		return nil, nil, NotFound, err
	}

	govPayResponse, err := callGovPay(gp, paymentSession)
	if err != nil {
		err = fmt.Errorf("error getting payment information from gov pay: [%v]", err)
		log.ErrorR(req, err)

		return nil, nil, Error, err
	}

	switch govPayResponse.RefundSummary.Status {
	case RefundUnavailable:
		err = errors.New("cannot refund the payment - check if the payment failed")
		return nil, nil, InvalidData, err
	case RefundFull:
		err = errors.New("cannot refund the payment - the full amount has already been refunded")
		return nil, nil, InvalidData, err
	case RefundPending:
		err = errors.New("cannot refund the payment - the user has not completed the payment")
		return nil, nil, InvalidData, err
	case RefundAvailable:
		return paymentSession, &govPayResponse.RefundSummary, Success, nil
	default:
		err = errors.New("cannot refund the payment - payment information not found")
		return nil, nil, NotFound, err
	}
}

// CreateRefund creates a refund in GovPay
func (gp *GovPayService) CreateRefund(paymentResource *models.PaymentResourceRest, refundRequest *models.CreateRefundGovPayRequest) (*models.CreateRefundGovPayResponse, ResponseType, error) {
	requestBody, err := json.Marshal(refundRequest)
	if err != nil {
		return nil, Error, fmt.Errorf("error reading refund GovPayRequest: [%s]", err)
	}

	request, err := http.NewRequest("POST", paymentResource.MetaData.ExternalPaymentStatusURI+"/refunds", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, Error, fmt.Errorf(govPayRequestError, err)
	}

	err = addGovPayHeaders(request, paymentResource, gp)
	if err != nil {
		return nil, Error, fmt.Errorf(govPayHeaderError, err)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, Error, fmt.Errorf("error sending request to GovPay to create a refund: [%s]", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, Error, fmt.Errorf("error reading response from GovPay: [%s]", err)
	}

	govPayResponse := &models.CreateRefundGovPayResponse{}
	err = json.Unmarshal(body, govPayResponse)
	if err != nil {
		return nil, Error, fmt.Errorf("error reading response from GovPay: [%s]", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, Error, fmt.Errorf(govPayStatusError, resp.StatusCode, govPayResponse.Status)
	}

	return govPayResponse, Success, nil
}

// GetRefundStatus gets refund status from GovPay
func (gp *GovPayService) GetRefundStatus(paymentResource *models.PaymentResourceRest, refundId string) (*models.CreateRefundGovPayResponse, ResponseType, error) {
	request, err := http.NewRequest("GET", paymentResource.MetaData.ExternalPaymentStatusURI+"/refunds/"+refundId, nil)
	if err != nil {
		return nil, Error, fmt.Errorf(govPayRequestError, err)
	}

	err = addGovPayHeaders(request, paymentResource, gp)
	if err != nil {
		return nil, Error, fmt.Errorf(govPayHeaderError, err)
	}

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, Error, fmt.Errorf("error sending request to GovPay to get status of a refund: [%s]", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, Error, fmt.Errorf("error reading response from GovPay: [%s]", err)
	}

	govPayResponse := &models.CreateRefundGovPayResponse{}

	err = json.Unmarshal(body, govPayResponse)
	if err != nil {
		return nil, Error, fmt.Errorf("error reading response from GovPay: [%s]", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, Error, fmt.Errorf(govPayStatusError, resp.StatusCode, govPayResponse.Status)
	}

	return govPayResponse, Success, nil
}

// decimalPayment will always be in the form XX.XX (e.g: 12.00) due to getTotalAmount converting to decimal with 2 fixed places right of decimal point.
func convertToPenceFromDecimal(decimalPayment string) (int, error) {
	pencePayment := strings.Replace(decimalPayment, ".", "", 1)
	return strconv.Atoi(pencePayment)
}

func callGovPay(gp *GovPayService, paymentResource *models.PaymentResourceRest) (*models.IncomingGovPayResponse, error) {

	if paymentResource.MetaData.ExternalPaymentStatusURI == "" {
		return nil, fmt.Errorf("gov pay URL not defined")
	}

	request, err := http.NewRequest("GET", paymentResource.MetaData.ExternalPaymentStatusURI, nil)
	if err != nil {
		return nil, fmt.Errorf(govPayRequestError, err)
	}

	err = addGovPayHeaders(request, paymentResource, gp)
	if err != nil {
		return nil, fmt.Errorf(govPayHeaderError, err)
	}

	// Make call to GovPay
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request to GovPay: [%s]", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response from GovPay: [%s]", err)
	}

	govPayResponse := &models.IncomingGovPayResponse{}
	err = json.Unmarshal(body, govPayResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response from GovPay: [%s]", err)
	}
	return govPayResponse, nil
}

func addGovPayHeaders(request *http.Request, paymentResource *models.PaymentResourceRest, gp *GovPayService) error {
	BearerToken := "Bearer "
	treasuryBearer := BearerToken + gp.PaymentService.Config.GovPayBearerTokenTreasury
	chBearer := BearerToken + gp.PaymentService.Config.GovPayBearerTokenChAccount
	legacyBearer := BearerToken + gp.PaymentService.Config.GovPayBearerTokenLegacy

	govPayTokens := map[string]string{
		"penalty":          treasuryBearer,
		"data-maintenance": chBearer,
		"orderable-item":   chBearer,
		"legacy":           legacyBearer,
	}

	token := govPayTokens[paymentResource.Costs[0].ClassOfPayment[0]]
	if token == "" {
		return fmt.Errorf("payment class [%s] not recognised", paymentResource.Costs[0].ClassOfPayment[0])
	}

	request.Header.Add("authorization", token)
	request.Header.Add("accept", "application/json")
	request.Header.Add("content-type", "application/json")

	return nil
}

// CapturePayment is a paypal specific implementation
// so it does not need to be implemented by the govpay svc
func (gp GovPayService) CapturePayment(_ string) (*paypal.CaptureOrderResponse, error) {
	// not implemented
	return nil, nil
}

// GetCapturedPaymentDetails is a PayPal specific implementation
// so it does not need to be implemented by the GOV.UK Pay svc
func (gp GovPayService) GetCapturedPaymentDetails(id string) (*paypal.CaptureDetailsResponse, error) {
	// not implemented
	return nil, nil
}

// RefundCapture is a PayPal specific implementation
// so it does not need to be implemented by the GOV.UK Pay svc
func (gp GovPayService) RefundCapture(captureID string) (*paypal.RefundResponse, error) {
	// not implemented
	return nil, nil
}
