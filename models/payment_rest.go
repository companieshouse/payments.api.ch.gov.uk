package models

import "time"

// IncomingPaymentResourceRequest is the data received in the body of the incoming request
type IncomingPaymentResourceRequest struct {
	RedirectURI string `json:"redirect_uri"`
	Reference   string `json:"reference"`
	Resource    string `json:"resource"`
	State       string `json:"state"`
}

// PaymentResourceRest is public facing payment details to be returned in the response
type PaymentResourceRest struct {
	Amount                  string             `json:"amount"`
	AvailablePaymentMethods []string           `json:"available_payment_methods,omitempty"`
	CompletedAt             time.Time          `json:"completed_at,omitempty"`
	CreatedAt               time.Time          `json:"created_at,omitempty"`
	CreatedBy               CreatedByRest      `json:"created_by"`
	Description             string             `json:"description"`
	Links                   PaymentLinksRest   `json:"links"`
	PaymentMethod           string             `json:"payment_method,omitempty"`
	Reference               string             `json:"reference,omitempty"`
	Status                  string             `json:"status"`
	Costs                   []CostResourceRest `json:"items"`
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

// CostResourceRest contains the details of an individual Cost Resource
type CostResourceRest struct {
	Amount                  string            `json:"amount"                    validate:"required"`
	AvailablePaymentMethods []string          `json:"available_payment_methods" validate:"required"`
	ClassOfPayment          []string          `json:"class_of_payment"          validate:"required"`
	Description             string            `json:"description"               validate:"required"`
	DescriptionIdentifier   string            `json:"description_identifier"    validate:"required"`
	DescriptionValues       map[string]string `json:"description_values"`
	Links                   CostLinksRest     `json:"links"                     validate:"required"`
	// IsVariablePayment       bool              `json:"is_variable_payment"` // removed from spec
}

// CostLinksRest is a set of URLs related to the resource, including self
type CostLinksRest struct {
	Resource string `json:"resource"`
	Self     string `json:"self" validate:"required"`
}
