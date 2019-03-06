package handlers

import (
	"fmt"
	"github.com/companieshouse/chs.go/avro"
	"github.com/companieshouse/chs.go/avro/schema"
	"github.com/companieshouse/chs.go/kafka/producer"
	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"net/http"

	"github.com/companieshouse/payments.api.ch.gov.uk/models"
)

// redirectUser redirects user to the provided redirect_uri with query params
func redirectUser(w http.ResponseWriter, r *http.Request, redirectURI string, params models.RedirectParams) {
	// Redirect the user to the redirect_uri, passing the state, ref and status as query params
	req, err := http.NewRequest("GET", redirectURI, nil)
	if err != nil {
		log.ErrorR(req, fmt.Errorf("error redirecting user: [%s]", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	query := req.URL.Query()
	query.Add("state", params.State)
	query.Add("ref", params.Ref)
	query.Add("status", params.Status)

	generatedURL := fmt.Sprintf("%s?%s", redirectURI, query.Encode())
	log.InfoR(r, "Redirecting to:", log.Data{"generated_url": generatedURL})

	http.Redirect(w, r, generatedURL, http.StatusSeeOther)
}

type paymentProcessed struct {
	PaymentSessionID string `avro:"payment_resource_url"`
}

func produceKafkaMessage(paymentID string) error {
	cfg, err := config.Get()
	kafkaProducer, err := producer.New(&producer.Config{Acks: &producer.WaitForAll, BrokerAddrs: cfg.BrokerAddr})
	if err != nil {
		log.Error(fmt.Errorf("error creating kafka producer: [%v]", err))
		return err
	}

	paymentProcessedSchema, err := schema.Get(cfg.SchemaRegistryURL, "payment-processed")
	producerAvro := &avro.Schema{
		Definition: paymentProcessedSchema,
	}

	paymentProcessedMessage := paymentProcessed{PaymentSessionID: paymentID}
	messageBytes, err := producerAvro.Marshal(paymentProcessedMessage)
	if err != nil {
		log.Error(fmt.Errorf("error marshalling payment processed message: [%v]", err))
		return err
	}

	producerTopic := "payment-processed"
	producerMessage := &producer.Message{
		Value: messageBytes,
		Topic: producerTopic,
	}

	kafkaProducer.Send(producerMessage)
	return nil
}
