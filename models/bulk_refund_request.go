package models

import "encoding/xml"

// RefundBatch is the overall model that consists of all of the refunds
type RefundBatch struct {
	XMLName       xml.Name        `xml:"batchService"`
	Version       string          `xml:"version,attr"`
	MerchantCode  string          `xml:"merchantCode,attr" validate:"required"`
	BatchCode     string          `xml:"batchCode,attr" validate:"required"`
	RefundDetails []RefundDetails `xml:"refund" validate:"required,dive,required"`
}

// RefundDetails is an individual GovPay refund
type RefundDetails struct {
	XMLName   xml.Name `xml:"refund"`
	Reference string   `xml:"reference,attr" validate:"required"`
	OrderCode string   `xml:"orderCode,attr" validate:"required"`
	Amount    Amount   `xml:"amount" validate:"required,dive,required"`
}

// Amount is the refund amount
type Amount struct {
	XMLName      xml.Name `xml:"amount"`
	Value        string   `xml:"value,attr" validate:"required"`
	CurrencyCode string   `xml:"currencyCode,attr" validate:"required"`
	Exponent     string   `xml:"exponent,attr" validate:"required"`
}
