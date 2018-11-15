package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

func returnNextURLGovPay(paymentResourceData *models.PaymentResourceData, id string, cfg *config.Config) (string, error) {
	var govPayRequest models.OutgoingGovPayRequest
	govPayRequest.Amount, _ = strconv.Atoi(paymentResourceData.Amount)
	govPayRequest.Description = "Companies House Payment" //Hardcoded for now as a payment description isn't saved anywhere
	govPayRequest.Reference = paymentResourceData.Reference
	govPayRequest.ReturnURL = cfg.PaymentsWebURL + "/payments/" + id + "/paymentStatus"

	requestBody, err := json.Marshal(govPayRequest)
	if err != nil {
		return "", fmt.Errorf("error reading GovPayRequest: [%s]", err)
	}

	request, err := http.NewRequest("POST", cfg.GovPayUrl, bytes.NewBuffer(requestBody))
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
	if resp.StatusCode != 201 {
		return "", fmt.Errorf("error status [%v] back from GovPay: [%s]", resp.StatusCode, govPayResponse.Description)
	}

	return govPayResponse.GovPayLinks.NextURL.HREF, nil
}
