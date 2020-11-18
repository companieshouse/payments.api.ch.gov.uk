package models

import "time"

// IncomingPaymentResourceRequest is the data received in the body of the incoming request
type IncomingPaymentResourceRequest struct {
	RedirectURI string `json:"redirect_uri" validate:"required,url"`
	Reference   string `json:"reference"`
	Resource    string `json:"resource"     validate:"required,url"`
	State       string `json:"state"        validate:"required"`
}

// PaymentResourceRest is public facing payment details to be returned in the response
type PaymentResourceRest struct {
	Amount                  string                      `json:"amount"`
	AvailablePaymentMethods []string                    `json:"available_payment_methods,omitempty"`
	CompletedAt             time.Time                   `json:"completed_at,omitempty"`
	CreatedAt               time.Time                   `json:"created_at,omitempty"`
	CreatedBy               CreatedByRest               `json:"created_by"`
	IpAddress               string                      `json:"ip_address"`
	Description             string                      `json:"description"`
	Links                   PaymentLinksRest            `json:"links"`
	PaymentMethod           string                      `json:"payment_method,omitempty"`
	Reference               string                      `json:"reference,omitempty"`
	CompanyNumber           string                      `json:"company_number,omitempty"`
	Status                  string                      `json:"status"`
	Costs                   []CostResourceRest          `json:"costs"`
	Etag                    string                      `json:"etag"`
	Kind                    string                      `json:"kind"`
	MetaData                PaymentResourceMetaDataRest `json:"-"`
	Refunds                 []RefundResourceRest        `json:"refunds"`
}

// PaymentResourceMetaDataRest contains all metadata fields that are relevant to the payment resource but not part of the Rest resource
type PaymentResourceMetaDataRest struct {
	ID                       string
	RedirectURI              string
	State                    string
	ExternalPaymentStatusURI string
}

// CreatedByRest is the user who is creating the payment session
type CreatedByRest struct {
	Email    string `json:"email"`
	Forename string `json:"forename"`
	ID       string `json:"id"`
	Surname  string `json:"surname"`
}

// PaymentLinksRest is a set of URLs related to the resource, including self
type PaymentLinksRest struct {
	Journey  string `json:"journey"`
	Resource string `json:"resource"`
	Self     string `json:"self" validate:"required"`
}

// Costs contains details of all the Cost Resources
type CostsRest struct {
	Description      string             `json:"description"`
	Etag             string             `json:"etag"`
	Costs            []CostResourceRest `json:"items"`
	Kind             string             `json:"kind"`
	Links            PaymentLinksRest   `json:"links"`
	PaidAt           time.Time          `json:"paid_at"`
	PaymentReference string             `json:"payment_reference"`
	Status           string             `json:"status"`
	CompanyNumber    string             `json:"company_number"`
}

// CostResourceRest contains the details of an individual Cost Resource
type CostResourceRest struct {
	Amount                  string            `json:"amount"                    validate:"required"`
	AvailablePaymentMethods []string          `json:"available_payment_methods" validate:"required"`
	ClassOfPayment          []string          `json:"class_of_payment"          validate:"required"`
	Description             string            `json:"description"               validate:"required"`
	DescriptionIdentifier   string            `json:"description_identifier"    validate:"required"`
	ProductType             string            `json:"product_type"              validate:"required"`
	DescriptionValues       map[string]string `json:"description_values"`
}
