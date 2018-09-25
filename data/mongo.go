package data

import (
	"fmt"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	mgo "gopkg.in/mgo.v2"
)

var session *mgo.Session

// GetMongoSession gets a MongoDB Session
func GetMongoSession() (*mgo.Session, error) {
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
