package dao

import (
	"fmt"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

var session *mgo.Session

// Mongo represents a simplistic MongoDB configuration.
type Mongo struct {
	URL string
}

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

// CreatePaymentResource writes a new payment resource to the DB
func (m *Mongo) CreatePaymentResource(paymentResource *models.PaymentResource) error {
	paymentSession, err := getMongoSession()
	if err != nil {
		return err
	}
	defer paymentSession.Close()

	c := paymentSession.DB("payments").C("payments")

	return c.Insert(paymentResource)
}

// GetPaymentResource gets a payment resource from the DB
// If payment not found in DB, return nil
func (m *Mongo) GetPaymentResource(id string) (*models.PaymentResource, error) {
	var resource models.PaymentResource
	paymentSession, err := getMongoSession()
	if err != nil {
		return &resource, err
	}
	defer paymentSession.Close()

	c := paymentSession.DB("payments").C("payments")
	err = c.FindId(id).One(&resource)

	// If Payment not found in DB, return empty resource
	if err != nil && err == mgo.ErrNotFound {
		return nil, nil
	}

	return &resource, err
}

// PatchPaymentResource patches a payment resource from the DB
func (m *Mongo) PatchPaymentResource(id string, paymentUpdate *models.PaymentResourceData) error {
	paymentSession, err := getMongoSession()
	if err != nil {
		return err
	}
	defer paymentSession.Close()

	c := paymentSession.DB("payments").C("payments")

	patchUpdate := make(bson.M)

	if paymentUpdate.PaymentMethod != "" || paymentUpdate.Status != "" {
		// Patch only these fields
		if paymentUpdate.PaymentMethod != "" {
			patchUpdate["data.payment_method"] = paymentUpdate.PaymentMethod
		}
		if paymentUpdate.Status != "" {
			patchUpdate["data.status"] = paymentUpdate.Status
		}
	} else {
		return fmt.Errorf("no valid fields for the patch request has been supplied for resource [%s]", id)
	}

	updateCall := bson.M{"$set": patchUpdate}

	return c.UpdateId(id, updateCall)
}
