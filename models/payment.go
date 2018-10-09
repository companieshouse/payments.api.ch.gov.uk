package models

import "time"

// IncomingPaymentResourceRequest is the data received in the body of the incoming request
type IncomingPaymentResourceRequest struct {
	RedirectURI string `json:"redirect_uri"`
	Reference   string `json:"reference"`
	Resource    string `json:"resource"`
	State       string `json:"state"`
}

// PaymentResource is the payment details to be stored in the DB and returned in the response
type PaymentResource struct {
	Amount                  string    `json:"amount"                              bson:"amount"`
	AvailablePaymentMethods []string  `json:"available_payment_methods,omitempty" bson:"available_payment_methods,omitempty"`
	CompletedAt             time.Time `json:"completed_at,omitempty"              bson:"completed_at,omitempty"`
	CreatedAt               time.Time `json:"created_at,omitempty"                bson:"created_at,omitempty"`
	CreatedBy               CreatedBy `json:"created_by"                          bson:"created_by"`
	Description             string    `json:"description"                         bson:"description"`
	Links                   Links     `json:"links"                               bson:"links"`
	PaymentMethod           string    `json:"payment_method,omitempty"            bson:"payment_method,omitempty"`
	Reference               string    `json:"reference,omitempty"                 bson:"reference,omitempty"`
	Status                  string    `json:"status"                              bson:"status"`
}

// CreatedBy is the user who is creating the payment session
type CreatedBy struct {
	Email    string `bson:"email"`
	Forename string `bson:"forename"`
	ID       string `bson:"id"`
	Surname  string `bson:"surname"`
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
