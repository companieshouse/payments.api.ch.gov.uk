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

type GovpayResponse struct{}

func (g GovpayResponse) checkProvider(paymentResource *models.PaymentResource) (*models.StatusResponse, error) {
	// Call the getGovPayPaymentState method down below to get state
	cfg, err := config.Get()
	if err != nil {
		return nil, fmt.Errorf("error getting config: [%s]", err)
	}

	state, err := getGovPayPaymentState(paymentResource, cfg)
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

func (service *PaymentService) returnNextURLGovPay(paymentResourceData *models.PaymentResourceData, id string, cfg *config.Config) (string, error) {
	var govPayRequest models.OutgoingGovPayRequest

	amountToPay, err := convertToPenceFromDecimal(paymentResourceData.Amount)
	if err != nil {
		return "", fmt.Errorf("error converting amount to pay to pence: [%s]", err)
	}

	govPayRequest.Amount = amountToPay
	govPayRequest.Description = "Companies House Payment" // TODO - Make description mandatory when creating payment-session so this doesn't have to be hardcoded
	govPayRequest.Reference = paymentResourceData.Reference
	govPayRequest.ReturnURL = fmt.Sprintf("%s/callback/payments/govpay/%s", cfg.PaymentsApiURL, id) // TODO - Change this URL when payment.web has been updated to contain a return page

	requestBody, err := json.Marshal(govPayRequest)
	if err != nil {
		return "", fmt.Errorf("error reading GovPayRequest: [%s]", err)
	}

	request, err := http.NewRequest("POST", cfg.GovPayURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error generating request for GovPay: [%s]", err)
	}

	request.Header.Add("accept", "application/json")
	request.Header.Add("authorization", "Bearer "+cfg.GovPayBearerToken)
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

	var PaymentResourceUpdate models.PaymentResource
	PaymentResourceUpdate.ExternalPaymentStatusURI = govPayResponse.GovPayLinks.Self.HREF

	_, err = service.patchPaymentSession(id, PaymentResourceUpdate)
	if err != nil {
		return "", fmt.Errorf("error patching payment session with PaymentStatusUrl: [%s]", err)
	}

	return govPayResponse.GovPayLinks.NextURL.HREF, nil
}

// To get the status of a GovPay payment, GET the payment resource from GovPay and return the State block
func getGovPayPaymentState(paymentResource *models.PaymentResource, cfg *config.Config) (*models.State, error) {
	request, err := http.NewRequest("GET", paymentResource.ExternalPaymentStatusURI, nil)
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
