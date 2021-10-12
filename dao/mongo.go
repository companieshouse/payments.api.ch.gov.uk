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
func (m *Mongo) CreatePaymentResource(paymentResource *models.PaymentResourceDB) error {
	paymentSession, err := getMongoSession()
	if err != nil {
		return err
	}
	defer paymentSession.Close()

	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("error getting config: %s", err)
	}
	c := paymentSession.DB(cfg.Database).C(cfg.Collection)

	return c.Insert(paymentResource)
}

// GetPaymentResource gets a payment resource from the DB
// If payment not found in DB, return nil
func (m *Mongo) GetPaymentResource(id string) (*models.PaymentResourceDB, error) {
	var resource models.PaymentResourceDB
	paymentSession, err := getMongoSession()
	if err != nil {
		return &resource, err
	}
	defer paymentSession.Close()

	cfg, err := config.Get()
	if err != nil {
		return nil, fmt.Errorf("error getting config: %s", err)
	}

	c := paymentSession.DB(cfg.Database).C(cfg.Collection)
	err = c.FindId(id).One(&resource)

	// If Payment not found in DB, return empty resource
	if err != nil && err == mgo.ErrNotFound {
		return nil, nil
	}

	return &resource, err
}

// PatchPaymentResource patches a payment resource from the DB
func (m *Mongo) PatchPaymentResource(id string, paymentUpdate *models.PaymentResourceDB) error {
	paymentSession, err := getMongoSession()
	if err != nil {
		return err
	}
	defer paymentSession.Close()

	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("error getting config: %s", err)
	}
	c := paymentSession.DB(cfg.Database).C(cfg.Collection)

	patchUpdate := make(bson.M)

	// Patch only these fields
	if paymentUpdate.Data.PaymentMethod != "" {
		patchUpdate["data.payment_method"] = paymentUpdate.Data.PaymentMethod
	}
	if paymentUpdate.Data.Status != "" {
		patchUpdate["data.status"] = paymentUpdate.Data.Status
	}
	if !paymentUpdate.Data.CompletedAt.IsZero() {
		patchUpdate["data.completed_at"] = paymentUpdate.Data.CompletedAt
	}
	if paymentUpdate.ExternalPaymentStatusURI != "" {
		patchUpdate["external_payment_status_url"] = paymentUpdate.ExternalPaymentStatusURI
	}
	if paymentUpdate.ExternalPaymentStatusID != "" {
		patchUpdate["external_payment_status_id"] = paymentUpdate.ExternalPaymentStatusID
	}
	if paymentUpdate.Refunds != nil {
		patchUpdate["refunds"] = paymentUpdate.Refunds
	}

	updateCall := bson.M{"$set": patchUpdate}

	err = c.UpdateId(id, updateCall)
	if err != nil {
		return err
	}

	return nil
}
