package models

import "encoding/xml"

// BatchService is the overall model that consists of all of the refunds
type BatchService struct {
	XMLName      xml.Name `xml:"batchService"`
	Version      string   `xml:"version,attr"`
	MerchantCode string   `xml:"merchantCode,attr" validate:"required"`
	BatchCode    string   `xml:"batchCode,attr" validate:"required"`
	Refunds      []Refund `xml:"refund" validate:"required,dive,required"`
}

// Refund is an individual worldpay refund
type Refund struct {
	XMLName   xml.Name `xml:"refund"`
	Reference string   `xml:"reference,attr" validate:"required"`
	OrderCode string   `xml:"orderCode,attr" validate:"required"`
	Amount    Amount   `xml:"amount" validate:"required,dive,required"`
}

// Amount is the worldpay refund amount
type Amount struct {
	XMLName      xml.Name `xml:"amount"`
	Value        string   `xml:"value,attr" validate:"required"`
	CurrencyCode string   `xml:"currencyCode,attr" validate:"required"`
	Exponent     string   `xml:"exponent,attr" validate:"required"`
}