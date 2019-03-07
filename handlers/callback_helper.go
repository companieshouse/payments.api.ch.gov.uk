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

// ProducerTopic is the topic to which the payment processed kafka message is sent
const ProducerTopic = "payment-processed"

// ProducerSchemaName is the schema which will be used to send the payment processed kafka message with
const ProducerSchemaName = "payment-processed"

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
	PaymentSessionID string `avro:"payment_resource_id"`
}

func produceKafkaMessage(paymentID string) error {
	cfg, err := config.Get()
	if err != nil {
		log.Error(fmt.Errorf("error getting config for kafka message production: [%v]", err))
		return err
	}

	// Get a producer
	kafkaProducer, err := producer.New(&producer.Config{Acks: &producer.WaitForAll, BrokerAddrs: cfg.BrokerAddr})
	if err != nil {
		log.Error(fmt.Errorf("error creating kafka producer: [%v]", err))
		return err
	}
	paymentProcessedSchema, err := schema.Get(cfg.SchemaRegistryURL, ProducerSchemaName)
	producerSchema := &avro.Schema{
		Definition: paymentProcessedSchema,
	}

	// Prepare a message with the avro schema
	message, err := prepareKafkaMessage(paymentID, *producerSchema)

	// Send the message
	_, offset, err := kafkaProducer.Send(message)
	if err != nil {
		log.Error(err, log.Data{"failed to send message at offset": offset})
		return err
	}
	return nil
}

// prepareKafkaMessage is pulled out of produceKafkaMessage() to allow unit testing of non-kafka portion of code
func prepareKafkaMessage(paymentID string, schema avro.Schema) (*producer.Message, error) {
	paymentProcessedMessage := paymentProcessed{PaymentSessionID: paymentID}
	messageBytes, err := schema.Marshal(paymentProcessedMessage)
	if err != nil {
		log.Error(fmt.Errorf("error marshalling payment processed message: [%v]", err))
		return nil, err
	}

	producerMessage := &producer.Message{
		Value: messageBytes,
		Topic: ProducerTopic,
	}
	return producerMessage, nil
}
