package service

import (
	"fmt"
	"net/http"

	"github.com/companieshouse/chs.go/log"
)

// GetExternalPaymentJourney gets an external payment session status from a payment provider, e.g: GovPay
func (service *PaymentService) FinishGovPayJourney(w http.ResponseWriter, req *http.Request) {
	// Get the payment session
	id := req.URL.Query().Get(":payment_id")
	if id == "" {
		log.ErrorR(req, fmt.Errorf("payment id not supplied"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// The PaymentResource must be retrieved directly to enable access to metadata outside the data block
	paymentResource, err := service.DAO.GetPaymentResource(id)
	if paymentResource == nil {
		log.ErrorR(req, fmt.Errorf("payment session not found. id: %s", id))
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error getting payment resource from db: [%v]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Ensure payment method matches endpoint
	if paymentResource.Data.PaymentMethod == "GovPay" {
		// Get the state of a GovPay payment
		GovpayResponse.checkProvider(GovpayResponse{}, id)
	} else {
		log.ErrorR(req, fmt.Errorf("payment method, [%s], for resource [%s] not recognised", paymentResource.Data.PaymentMethod, id))
		w.WriteHeader(http.StatusBadRequest)
	}
}
