package models

import "time"

// PaymentResourceDB contains all payment details to be stored in the DB
type PaymentResourceDB struct {
	ID                       string                `bson:"_id"`
	RedirectURI              string                `bson:"redirect_uri"`
	State                    string                `bson:"state"`
	ExternalPaymentStatusURI string                `bson:"external_payment_status_url"`
	Data                     PaymentResourceDataDB `bson:"data"`
}

// PaymentResourceDataDB is public facing payment details to be returned in the response
type PaymentResourceDataDB struct {
	Amount        string         `bson:"amount"`
	CompletedAt   time.Time      `bson:"completed_at,omitempty"`
	CreatedAt     time.Time      `bson:"created_at,omitempty"`
	CreatedBy     CreatedByDB    `bson:"created_by"`
	Description   string         `bson:"description"`
	Links         PaymentLinksDB `bson:"links"`
	PaymentMethod string         `bson:"payment_method"`
	Reference     string         `bson:"reference,omitempty"`
	Status        string         `bson:"status"`
}

// CreatedByDB is the user who is creating the payment session
type CreatedByDB struct {
	Email    string `bson:"email"`
	Forename string `bson:"forename"`
	ID       string `bson:"id"`
	Surname  string `bson:"surname"`
}

// PaymentLinksDB is a set of URLs related to the resource, including self
type PaymentLinksDB struct {
	Journey  string `bson:"journey"`
	Resource string `bson:"resource"`
	Self     string `bson:"self" validate:"required"`
}

// CostLinksDB is a set of URLs related to the resource, including self
type CostLinksDB struct {
	Resource string `bson:"resource"`
	Self     string `bson:"self" validate:"required"`
}