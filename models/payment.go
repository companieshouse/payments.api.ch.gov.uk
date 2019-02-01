package models

import "time"

// PaymentResource contains all payment details to be stored in the DB
type PaymentResource struct {
	ID                       string              `bson:"_id"`
	RedirectURI              string              `bson:"redirect_uri"`
	State                    string              `bson:"state"`
	ExternalPaymentStatusURI string              `bson:"external_payment_status_url"`
	Data                     PaymentResourceData `bson:"data"`
}

// PaymentResourceData is public facing payment details to be returned in the response
type PaymentResourceData struct {
	Amount                  string         `bson:"amount"`
	AvailablePaymentMethods []string       `bson:"available_payment_methods,omitempty"`
	CompletedAt             time.Time      `bson:"completed_at,omitempty"`
	CreatedAt               time.Time      `bson:"created_at,omitempty"`
	CreatedBy               CreatedBy      `bson:"created_by"`
	Description             string         `bson:"description"`
	Links                   PaymentLinks   `bson:"links"`
	PaymentMethod           string         `bson:"payment_method"`
	Reference               string         `bson:"reference,omitempty"`
	Status                  string         `bson:"status"`
	Costs                   []CostResource `bson:"items"`
}

// CreatedBy is the user who is creating the payment session
type CreatedBy struct {
	Email    string `bson:"email"`
	Forename string `bson:"forename"`
	ID       string `bson:"id"`
	Surname  string `bson:"surname"`
}

// PaymentLinks is a set of URLs related to the resource, including self
type PaymentLinks struct {
	Journey  string `bson:"journey"`
	Resource string `bson:"resource"`
	Self     string `bson:"self" validate:"required"`
}

// CostResource contains the details of an individual Cost Resource
type CostResource struct {
	Amount                  string            `json:"amount"                    validate:"required"`
	AvailablePaymentMethods []string          `json:"available_payment_methods" validate:"required"`
	ClassOfPayment          []string          `json:"class_of_payment"          validate:"required"`
	Description             string            `json:"description"               validate:"required"`
	DescriptionIdentifier   string            `json:"description_identifier"    validate:"required"`
	DescriptionValues       map[string]string `json:"description_values"`
	Links                   CostLinks         `json:"links"                     validate:"required"`
}

// CostLinks is a set of URLs related to the resource, including self
type CostLinks struct {
	Resource string `bson:"resource"`
	Self     string `bson:"self" validate:"required"`
}
