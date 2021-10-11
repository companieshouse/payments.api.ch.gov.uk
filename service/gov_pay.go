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

	"github.com/plutov/paypal/v4"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// GovPayService handles the specific functionality of integrating GovPay provider into Payment Sessions
type GovPayService struct {
	PaymentService PaymentService
}

// CheckPaymentProviderStatus checks the status of the payment with GovPay
func (gp GovPayService) CheckPaymentProviderStatus(paymentResource *models.PaymentResourceRest) (*models.StatusResponse, ResponseType, error) {
	// Call the getGovPayPaymentState method down below to get state
	cfg, err := config.Get()
	if err != nil {
		return nil, Error, fmt.Errorf("error getting config: [%s]", err)
	}

	state, responseType, err := gp.getGovPayPaymentState(paymentResource, cfg)
	if err != nil {
		return nil, responseType, fmt.Errorf("error getting state of GovPay payment: [%s]", err)
	}
	// Return state
	if state.Finished && state.Status == "success" {
		return &models.StatusResponse{Status: "paid"}, Success, nil
	} else if state.Finished && state.Code == "P0030" {
		return &models.StatusResponse{Status: "cancelled"}, Success, nil
	}
	return &models.StatusResponse{Status: "failed"}, Error, nil
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
		return "", Error, fmt.Errorf("error generating request for GovPay: [%s]", err)
	}

	err = addGovPayHeaders(request, paymentResource, gp)
	if err != nil {
		return "", InvalidData, fmt.Errorf("error adding GovPay headers: [%s]", err)
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
		return "", Error, fmt.Errorf("error status [%v] back from GovPay: [%s]", resp.StatusCode, govPayResponse.Description)
	}

	err = gp.PaymentService.StoreExternalPaymentStatusDetails(paymentResource.MetaData.ID, govPayResponse.GovPayLinks.Self.HREF, govPayResponse.PaymentID)
	if err != nil {
		return "", Error, fmt.Errorf("error storing GovPay external payment details for payment session: [%s]", err)
	}

	return govPayResponse.GovPayLinks.NextURL.HREF, Success, nil
}

// To get the status of a GovPay payment, GET the payment resource from GovPay and return the State block
func (gp *GovPayService) getGovPayPaymentState(paymentResource *models.PaymentResourceRest, cfg *config.Config) (*models.State, ResponseType, error) {

	govPayResponse, err := callGovPay(gp, paymentResource)
	if err != nil {
		return nil, Error, err
	}

	// Return the status of the payment
	return &govPayResponse.State, Success, nil
}

// GetPaymentDetails gets the details of a GovPay payment
func (gp *GovPayService) GetPaymentDetails(paymentResource *models.PaymentResourceRest) (*models.PaymentDetails, ResponseType, error) {

	govPayResponse, err := callGovPay(gp, paymentResource)
	if err != nil {
		return nil, Error, err
	}

	paymentDetails := &models.PaymentDetails{CardType: govPayResponse.CardBrand, ExternalPaymentID: govPayResponse.PaymentID, TransactionDate: govPayResponse.CreatedDate}

	if govPayResponse.State.Finished && govPayResponse.State.Status == "success" {
		paymentDetails.PaymentStatus = "accepted"
	} else if govPayResponse.State.Finished && govPayResponse.State.Code == "P0010" {
		paymentDetails.PaymentStatus = "rejected"
	}

	return paymentDetails, Success, nil
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
		return nil, Error, fmt.Errorf("error generating request for GovPay: [%s]", err)
	}

	err = addGovPayHeaders(request, paymentResource, gp)
	if err != nil {
		return nil, Error, fmt.Errorf("error adding GovPay headers: [%s]", err)
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
		return nil, Error, fmt.Errorf("error status [%v] back from GovPay: [%s]", resp.StatusCode, govPayResponse.Status)
	}

	return govPayResponse, Success, nil
}

// GetRefundStatus gets refund status from GovPay
func (gp *GovPayService) GetRefundStatus(paymentResource *models.PaymentResourceRest, refundId string) (*models.GetRefundStatusGovPayResponse, ResponseType, error) {
	request, err := http.NewRequest("GET", paymentResource.MetaData.ExternalPaymentStatusURI+"/refunds/"+refundId, nil)
	if err != nil {
		return nil, Error, fmt.Errorf("error generating request for GovPay: [%s]", err)
	}

	err = addGovPayHeaders(request, paymentResource, gp)
	if err != nil {
		return nil, Error, fmt.Errorf("error adding GovPay headers: [%s]", err)
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

	govPayResponse := &models.GetRefundStatusGovPayResponse{}
	err = json.Unmarshal(body, govPayResponse)
	if err != nil {
		return nil, Error, fmt.Errorf("error reading response from GovPay: [%s]", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, Error, fmt.Errorf("error status [%v] back from GovPay: [%s]", resp.StatusCode, govPayResponse.Status)
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
		return nil, fmt.Errorf("error generating request for GovPay: [%s]", err)
	}

	err = addGovPayHeaders(request, paymentResource, gp)
	if err != nil {
		return nil, fmt.Errorf("error adding GovPay headers: [%s]", err)
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
	treasuryBearer := "Bearer " + gp.PaymentService.Config.GovPayBearerTokenTreasury
	chBearer := "Bearer " + gp.PaymentService.Config.GovPayBearerTokenChAccount
	legacyBearer := "Bearer " + gp.PaymentService.Config.GovPayBearerTokenLegacy

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
	return nil, nil
}
