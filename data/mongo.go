package data

import (
	"fmt"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	mgo "gopkg.in/mgo.v2"
)

var session *mgo.Session

// getMongoSession gets a MongoDB Session
func getMongoSession() (*mgo.Session, error) {
	if session == nil {
		var err error
		cfg, err := config.Get()
		if err != nil {
			return nil, fmt.Errorf("error getting config: %s", err)
		}
		session, err = mgo.Dial(cfg.MongoDBURL)
		if err != nil {
			return nil, fmt.Errorf("error dialling into mongodb: %s", err)
		}
	}
	return session.Copy(), nil
}

// CreatePaymentResourceDB writes a new payment resource to the DB
func CreatePaymentResourceDB(paymentResource *PaymentResource) error {

	paymentSession, err := getMongoSession()
	if err != nil {
		return err
	}
	defer paymentSession.Close()

	c := paymentSession.DB("transactions").C("payments")

	return c.Insert(paymentResource)

}
