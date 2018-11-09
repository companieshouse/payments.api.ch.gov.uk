package models

import "time"

// IncomingPaymentResourceRequest is the data received in the body of the incoming request
type IncomingPaymentResourceRequest struct {
	RedirectURI string `json:"redirect_uri"`
	Reference   string `json:"reference"`
	Resource    string `json:"resource"`
	State       string `json:"state"`
}

// PaymentResource contains all payment details to be stored in the DB
type PaymentResource struct {
	ID   string              `json:"_id"   bson:"_id"`
	Data PaymentResourceData `json:"data"  bson:"data"`
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
	RedirectURI             string         `json:"redirect_uri"                        bson:"redirect_uri"`
	Reference               string         `json:"reference,omitempty"                 bson:"reference,omitempty"`
	Status                  string         `json:"status"                              bson:"status"`
	Costs                   []CostResource `json:"items"`
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
	Self     string `json:"self"`
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
	Amount                  string            `json:"amount"`
	AvailablePaymentMethods []string          `json:"available_payment_methods"`
	ClassOfPayment          []string          `json:"class_of_payment"`
	Description             string            `json:"description"`
	DescriptionIdentifier   string            `json:"description_identifier"`
	DescriptionValues       DescriptionValues `json:"description_values"`
	IsVariablePayment       bool              `json:"is_variable_payment"`
	Links                   Links             `json:"links"`
}

// DescriptionValues contains a description of the cost
type DescriptionValues struct {
}
