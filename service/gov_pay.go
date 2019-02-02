package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// GovPayService handles the specific functionality of integrating GovPay provider into Payment Sessions
type GovPayService struct {
	PaymentService PaymentService
}

// CheckProvider checks the status of the payment with GovPay provider
func (gp GovPayService) CheckProvider(paymentResource *models.PaymentResourceRest) (*models.StatusResponse, error) {
	// Call the getGovPayPaymentState method down below to get state
	cfg, err := config.Get()
	if err != nil {
		return nil, fmt.Errorf("error getting config: [%s]", err)
	}

	state, err := gp.getGovPayPaymentState(paymentResource, cfg)
	if err != nil {
		return nil, fmt.Errorf("error getting state of GovPay payment: [%s]", err)
	}
	// Return state
	if state.Finished == true && state.Status == "success" {
		return &models.StatusResponse{"paid"}, nil
	} else {
		return &models.StatusResponse{"failed"}, nil
	}
}

// GenerateNextURLGovPay creates a goc pay session linked to the given payment session and stores the required details on the payment session
func (gp *GovPayService) GenerateNextURLGovPay(req *http.Request, paymentResource *models.PaymentResourceRest) (string, error) {
	var govPayRequest models.OutgoingGovPayRequest

	amountToPay, err := convertToPenceFromDecimal(paymentResource.Amount)
	if err != nil {
		return "", fmt.Errorf("error converting amount to pay to pence: [%s]", err)
	}

	govPayRequest.Amount = amountToPay
	govPayRequest.Description = "Companies House Payment" // TODO - Make description mandatory when creating payment-session so this doesn't have to be hardcoded
	govPayRequest.Reference = paymentResource.Reference
	govPayRequest.ReturnURL = fmt.Sprintf("%s/callback/payments/govpay/%s", gp.PaymentService.Config.PaymentsApiURL, paymentResource.MetaData.ID)

	requestBody, err := json.Marshal(govPayRequest)
	if err != nil {
		return "", fmt.Errorf("error reading GovPayRequest: [%s]", err)
	}

	request, err := http.NewRequest("POST", gp.PaymentService.Config.GovPayURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error generating request for GovPay: [%s]", err)
	}

	request.Header.Add("accept", "application/json")
	request.Header.Add("authorization", "Bearer "+gp.PaymentService.Config.GovPayBearerToken)
	request.Header.Add("content-type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("error sending request to GovPay to start payment session: [%s]", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response from GovPay: [%s]", err)
	}

	govPayResponse := &models.IncomingGovPayResponse{}
	err = json.Unmarshal(body, govPayResponse)
	if err != nil {
		return "", fmt.Errorf("error reading response from GovPay: [%s]", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("error status [%v] back from GovPay: [%s]", resp.StatusCode, govPayResponse.Description)
	}

	// cannot patch a field that is not part of Rest model so create helper within service to store this metadata field
	// var PaymentResourceUpdate models.PaymentResourceRest
	// PaymentResourceUpdate.ExternalPaymentStatusURI = govPayResponse.GovPayLinks.Self.HREF

	err = gp.PaymentService.StoreExternalPaymentStatusURI(req, paymentResource.MetaData.ID, govPayResponse.GovPayLinks.Self.HREF)
	if err != nil {
		return "", fmt.Errorf("error storing ExternalPaymentStatusURI for payment session: [%s]", err)
	}

	return govPayResponse.GovPayLinks.NextURL.HREF, nil
}

// To get the status of a GovPay payment, GET the payment resource from GovPay and return the State block
func (gp *GovPayService) getGovPayPaymentState(paymentResource *models.PaymentResourceRest, cfg *config.Config) (*models.State, error) {
	request, err := http.NewRequest("GET", paymentResource.MetaData.ExternalPaymentStatusURI, nil)
	if err != nil {
		return nil, fmt.Errorf("error generating request for GovPay: [%s]", err)
	}

	request.Header.Add("accept", "application/json")
	request.Header.Add("authorization", "Bearer "+cfg.GovPayBearerToken)
	request.Header.Add("content-type", "application/json")

	// Make call to GovPay to check state of payment
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error sending request to GovPay to check payment status: [%s]", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response from GovPay when checking payment status: [%s]", err)
	}

	govPayResponse := &models.IncomingGovPayResponse{}
	err = json.Unmarshal(body, govPayResponse)
	if err != nil {
		return nil, fmt.Errorf("error reading response from GovPay when checking payment status: [%s]", err)
	}

	// Return the status of the payment
	return &govPayResponse.State, nil
}
