package models

import "time"

// IncomingPaymentResourceRequest is the data received in the body of the incoming request
type IncomingPaymentResourceRequest struct {
	RedirectURI string `json:"redirect_uri"        validate:"required"`
	Reference   string `json:"reference"`
	Resource    string `json:"resource"            validate:"required"`
	State       string `json:"state"               validate:"required"`
}

// PaymentResource contains all payment details to be stored in the DB
type PaymentResource struct {
	ID                       string              `json:"_id"                           bson:"_id"`
	RedirectURI              string              `json:"redirect_uri"                  bson:"redirect_uri"`
	State                    string              `json:"state"                         bson:"state"`
	ExternalPaymentStatusURI string              `json:"external_payment_status_url"   bson:"external_payment_status_url"`
	Data                     PaymentResourceData `json:"data"                          bson:"data"`
}

// PaymentResourceData is public facing payment details to be returned in the response
type PaymentResourceData struct {
	Amount                  string         `json:"amount"                              bson:"amount"`
	AvailablePaymentMethods []string       `json:"available_payment_methods,omitempty" bson:"available_payment_methods,omitempty"`
	CompletedAt             time.Time      `json:"completed_at,omitempty"              bson:"completed_at,omitempty"`
	CreatedAt               time.Time      `json:"created_at,omitempty"                bson:"created_at,omitempty"`
	CreatedBy               CreatedBy      `json:"created_by"                          bson:"created_by"`
	Description             string         `json:"description"                         bson:"description"`
	Links                   Links          `json:"links"                               bson:"links"`
	PaymentMethod           string         `json:"payment_method,omitempty"            bson:"payment_method"`
	Reference               string         `json:"reference,omitempty"                 bson:"reference,omitempty"`
	Status                  string         `json:"status"                              bson:"status"`
	Costs                   []CostResource `json:"items"`
	Etag                    string         `bson:"etag"`
	Kind                    string         `bson:"kind"`
}

// CreatedBy is the user who is creating the payment session
type CreatedBy struct {
	Email    string `json:"email"    bson:"email"`
	Forename string `json:"forename" bson:"forename"`
	ID       string `json:"id"       bson:"id"`
	Surname  string `json:"surname"  bson:"surname"`
}

// Links is a set of URLs related to the resource, including self
type Links struct {
	Journey  string `json:"journey"`
	Resource string `json:"resource"`
	Self     string `json:"self" validate:"required"`
}

// Data is a representation of the top level data retrieved from the Transaction API
type Data struct {
	CompanyName string            `json:"company_name"`
	Filings     map[string]Filing `json:"filings"`
}

// Filing is a representation of the Filing subsection of data retrieved from the Transaction API
type Filing struct {
	Description string `json:"description"`
}

// CostResource contains the details of an individual Cost Resource
type CostResource struct {
	Amount                  string            `json:"amount"                    validate:"required"`
	AvailablePaymentMethods []string          `json:"available_payment_methods" validate:"required"`
	ClassOfPayment          []string          `json:"class_of_payment"          validate:"required"`
	Description             string            `json:"description"               validate:"required"`
	DescriptionIdentifier   string            `json:"description_identifier"    validate:"required"`
	DescriptionValues       DescriptionValues `json:"description_values"`
	IsVariablePayment       bool              `json:"is_variable_payment"`
	Links                   Links             `json:"links"                     validate:"required"`
}

// DescriptionValues contains a description of the cost
type DescriptionValues struct {
}
