package service

import (
	"fmt"
	"testing"

	"gopkg.in/jarcoal/httpmock.v1"

	"github.com/companieshouse/payments.api.ch.gov.uk/models"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/golang/mock/gomock"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitGovPay(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	Convey("Error converting amount to pay to pence", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("POST", cfg.GovPayURL, httpmock.NewErrorResponder(fmt.Errorf("error")))

		paymentResourceData := models.PaymentResourceData{Amount: "250.567"}
		govPayResponse, err := returnNextURLGovPay(&paymentResourceData, "1234", cfg)

		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Error sending request to GovPay", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("POST", cfg.GovPayURL, httpmock.NewErrorResponder(fmt.Errorf("error")))

		paymentResourceData := models.PaymentResourceData{Amount: "250"}
		govPayResponse, err := returnNextURLGovPay(&paymentResourceData, "1234", cfg)

		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Error reading response from GovPay", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		jsonResponse, _ := httpmock.NewJsonResponder(500, "string")
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		paymentResourceData := models.PaymentResourceData{Amount: "250"}
		govPayResponse, err := returnNextURLGovPay(&paymentResourceData, "1234", cfg)

		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Status code not 201", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		IncomingGovPayResponse := models.IncomingGovPayResponse{}

		jsonResponse, _ := httpmock.NewJsonResponder(500, IncomingGovPayResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		paymentResourceData := models.PaymentResourceData{Amount: "250"}
		govPayResponse, err := returnNextURLGovPay(&paymentResourceData, "1234", cfg)

		So(govPayResponse, ShouldEqual, "")
		So(err, ShouldNotBeNil)
	})

	Convey("Valid request to GovPay and returned NextURL", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		journeyURL := "nextUrl"
		NextURL := models.NextURL{HREF: journeyURL}
		GovPayLinks := models.GovPayLinks{NextURL: NextURL}
		IncomingGovPayResponse := models.IncomingGovPayResponse{GovPayLinks: GovPayLinks}

		jsonResponse, _ := httpmock.NewJsonResponder(201, IncomingGovPayResponse)
		httpmock.RegisterResponder("POST", cfg.GovPayURL, jsonResponse)

		paymentResourceData := models.PaymentResourceData{Amount: "250"}
		govPayResponse, err := returnNextURLGovPay(&paymentResourceData, "1234", cfg)

		So(govPayResponse, ShouldEqual, journeyURL)
		So(err, ShouldBeNil)
	})

}
