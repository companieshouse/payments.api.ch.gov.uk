package service

import (
	"fmt"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/davecgh/go-spew/spew"
	"log"
	"net/http"
	"os"
)

// redirectUser redirects user to the provided redirect_uri with query params
func redirectUser(w http.ResponseWriter, r *http.Request, redirectURI string, state string, ref string, status string) {
	// Redirect the user to the redirect_uri, passing the state, ref and status as query params
	req, err := http.NewRequest("GET", redirectURI, nil)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	query := req.URL.Query()
	query.Add("state", state)
	query.Add("ref", ref)
	query.Add("status", status)
	//req.URL.RawQuery = query.Encode()

	generatedURL := fmt.Sprintf("%s?%s", redirectURI, query.Encode())
	spew.Dump(generatedURL)
	http.Redirect(w, r, generatedURL, http.StatusSeeOther)
}

func produceKafkaMessage() {
	// TODO: Produce message to payment-processed topic
}

func (service *PaymentService) UpdatePaymentStatus(s models.StatusResponse, p models.PaymentResource) {
	//requestDecoder := json.NewDecoder(p)
	//var PaymentResourceUpdate models.PaymentResourceData
	//err := requestDecoder.Decode(&PaymentResourceUpdate)
	//if err != nil {
	//	log.ErrorR(req, fmt.Errorf("request body invalid: [%v]", err))
	//	w.WriteHeader(http.StatusBadRequest)
	//	return
	//}
	//
	//if PaymentResourceUpdate.PaymentMethod == "" && PaymentResourceUpdate.Status == "" {
	//	log.ErrorR(req, fmt.Errorf("no valid fields for the patch request has been supplied for resource [%s]", id))
	//	w.WriteHeader(http.StatusBadRequest)
	//	return
	//}

	err := service.DAO.PatchPaymentResource(p.ID, &p.Data)
	if err != nil {
		//if err.Error() == "not found" {
		//	log.ErrorR(req, fmt.Errorf("could not find payment resource to patch"))
		//	w.WriteHeader(http.StatusForbidden)
		//	return
		//}
		//log.ErrorR(req, fmt.Errorf("error patching payment session on database: [%v]", err))
		//w.WriteHeader(http.StatusInternalServerError)
		return
	}
	return
	//log.InfoR(req, "Successful PATCH request for payment resource", log.Data{"payment_id": id, "status": http.StatusOK})
}
