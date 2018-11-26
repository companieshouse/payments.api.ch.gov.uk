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

func returnNextURLGovPay(paymentResourceData *models.PaymentResourceData, id string, cfg *config.Config) (string, error) {
	var govPayRequest models.OutgoingGovPayRequest

	amountToPay, err := convertToPenceFromDecimal(paymentResourceData.Amount)
	if err != nil {
		return "", fmt.Errorf("error converting amount to pay to pence: [%s]", err)
	}

	govPayRequest.Amount = amountToPay
	govPayRequest.Description = "Companies House Payment" // TODO - Make description mandatory when creating payment-session so this doesn't have to be hardcoded
	govPayRequest.Reference = paymentResourceData.Reference
	govPayRequest.ReturnURL = cfg.PaymentsWebURL + "/payments/" + id + "/paymentStatus" // TODO - Change this URL when payment.web has been updated to contain a return page

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

	return govPayResponse.GovPayLinks.NextURL.HREF, nil
}
