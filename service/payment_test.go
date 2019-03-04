package service

import (
	"context"
	"fmt"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/helpers"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"gopkg.in/jarcoal/httpmock.v1"
)

var defaultCost = models.CostResourceRest{
	Amount:                  "10",
	AvailablePaymentMethods: []string{"GovPay"},
	ClassOfPayment:          []string{"class"},
	Description:             "desc",
	DescriptionIdentifier:   "identifier",
	Links:                   models.CostLinksRest{Self: "self"},
}

var defaultCostArray = []models.CostResourceRest{
	defaultCost,
}

var defaultUserDetails = models.AuthUserDetails{
	Email:    "email@companieshouse.gov.uk",
	Forename: "forename",
	Id:       "id",
	Surname:  "surname",
}

func createMockPaymentService(dao *dao.MockDAO, config *config.Config) PaymentService {
	return PaymentService{
		DAO:    dao,
		Config: *config,
	}
}

func TestUnitPaymentStatus(t *testing.T) {
	Convey("Payment Status", t, func() {
		status := Pending.String()
		So(status, ShouldEqual, "pending")
	})
}

func TestUnitCreatePaymentSession(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()
	cfg.DomainWhitelist = "http://dummy-url"

	Convey("Empty Request Body", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req := httptest.NewRequest("GET", "/test", nil)

		paymentResourceRest, status, err := mockPaymentService.CreatePaymentSession(req, models.IncomingPaymentResourceRequest{})
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, InvalidData)
		So(err.Error(), ShouldEqual, "invalid incoming payment: [Key: 'IncomingPaymentResourceRequest.RedirectURI' Error:Field validation for 'RedirectURI' failed on the 'required' tag\nKey: 'IncomingPaymentResourceRequest.Resource' Error:Field validation for 'Resource' failed on the 'required' tag\nKey: 'IncomingPaymentResourceRequest.State' Error:Field validation for 'State' failed on the 'required' tag]")
	})

	Convey("Empty Request Body", t, func() {
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req := httptest.NewRequest("GET", "/test", nil)

		resource := models.IncomingPaymentResourceRequest{
			RedirectURI: "http://www.companieshouse.gov.uk",
			Resource:    "http://dummy-url",
			State:       "state",
		}
		paymentResourceRest, status, err := mockPaymentService.CreatePaymentSession(req, resource)
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "invalid AuthUserDetails in request context")
	})

	Convey("Error getting cost resource", t, func() {
		fmt.Println("ENM start")
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "http://dummy-resource", nil)

		authUserDetails := models.AuthUserDetails{
			Id: "identity",
		}

		ctx := context.WithValue(req.Context(), helpers.ContextKeyUserDetails, authUserDetails)

		resource := models.IncomingPaymentResourceRequest{
			RedirectURI: "http://www.companieshouse.gov.uk",
			Resource:    "http://dummy-url",
			State:       "state",
		}

		paymentResourceRest, status, err := mockPaymentService.CreatePaymentSession(req.WithContext(ctx), resource)
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error getting payment resource: [error getting Cost Resource: [Get http://dummy-url: no responder found]]")
	})

	Convey("Error getting total amount from costs", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costResource := defaultCost
		costResource.Amount = "invalid_amount"
		jsonResponse, _ := httpmock.NewJsonResponder(200, []models.CostResourceRest{costResource})
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		authUserDetails := models.AuthUserDetails{
			Id: "identity",
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyUserDetails, authUserDetails)

		resource := models.IncomingPaymentResourceRequest{
			Resource:    "http://dummy-url",
			RedirectURI: "http://www.companieshouse.gov.uk",
			State:       "state",
		}
		paymentResourceRest, status, err := mockPaymentService.CreatePaymentSession(req.WithContext(ctx), resource)
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error getting amount from costs: [amount [invalid_amount] format incorrect]")
	})

	Convey("Error Creating DB Resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().CreatePaymentResource(gomock.Any()).Return(fmt.Errorf("error"))

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCostArray)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		authUserDetails := models.AuthUserDetails{
			Id: "identity",
		}
		ctx := context.WithValue(req.Context(), helpers.ContextKeyUserDetails, authUserDetails)

		resource := models.IncomingPaymentResourceRequest{
			RedirectURI: "http://www.companieshouse.gov.uk",
			Resource:    "http://dummy-url",
			State:       "state",
		}
		paymentResourceRest, status, err := mockPaymentService.CreatePaymentSession(req.WithContext(ctx), resource)
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error writing to MongoDB: error")
	})

	cfg.PaymentsWebURL = "https://payments.companieshouse.gov.uk"

	Convey("Valid request - single cost", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().CreatePaymentResource(gomock.Any())

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCostArray)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		ctx := context.WithValue(req.Context(), helpers.ContextKeyUserDetails, defaultUserDetails)

		resource := models.IncomingPaymentResourceRequest{
			Resource:    "http://dummy-url",
			Reference:   "ref",
			RedirectURI: "http://www.companieshouse.gov.uk",
			State:       "state",
		}

		paymentResourceRest, status, err := mockPaymentService.CreatePaymentSession(req.WithContext(ctx), resource)

		So(err, ShouldBeNil)
		So(status, ShouldEqual, Success)

		So(paymentResourceRest.Amount, ShouldEqual, "10.00")
		So(paymentResourceRest.AvailablePaymentMethods, ShouldResemble, []string{"GovPay"})
		So(paymentResourceRest.CompletedAt, ShouldHaveSameTypeAs, time.Now())
		So(paymentResourceRest.CreatedAt, ShouldHaveSameTypeAs, time.Now())
		So(paymentResourceRest.CreatedBy, ShouldResemble, models.CreatedByRest{
			Email:    "email@companieshouse.gov.uk",
			Forename: "forename",
			ID:       "id",
			Surname:  "surname",
		})
		So(paymentResourceRest.Description, ShouldBeEmpty) // TODO implement this
		So(paymentResourceRest.Links.Resource, ShouldEqual, "http://dummy-url")
		regJourney := regexp.MustCompile("https://payments.companieshouse.gov.uk/payments/(.*)/pay")
		So(regJourney.MatchString(paymentResourceRest.Links.Journey), ShouldEqual, true)
		regSelf := regexp.MustCompile("payments/(.*)")
		So(regSelf.MatchString(paymentResourceRest.Links.Self), ShouldEqual, true)
		So(paymentResourceRest.PaymentMethod, ShouldBeEmpty)
		So(paymentResourceRest.Reference, ShouldEqual, "ref")
		So(paymentResourceRest.Status, ShouldEqual, "pending")
		So(paymentResourceRest.Costs, ShouldResemble, defaultCostArray)
		So(paymentResourceRest.MetaData, ShouldResemble, models.PaymentResourceMetaDataRest{
			ID:                       "",
			RedirectURI:              "",
			State:                    "",
			ExternalPaymentStatusURI: "",
		})

		So(status, ShouldEqual, Success)
		So(err, ShouldBeNil)
	})

	Convey("Valid request - multiple costs", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().CreatePaymentResource(gomock.Any())
		req := httptest.NewRequest("Get", "/test", nil)
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost, defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		ctx := context.WithValue(req.Context(), helpers.ContextKeyUserDetails, defaultUserDetails)

		resource := models.IncomingPaymentResourceRequest{
			Resource:    "http://dummy-url",
			Reference:   "ref",
			RedirectURI: "http://www.companieshouse.gov.uk",
			State:       "state",
		}

		paymentResourceRest, status, err := mockPaymentService.CreatePaymentSession(req.WithContext(ctx), resource)

		So(status, ShouldEqual, Success)
		So(err, ShouldBeNil)

		So(paymentResourceRest.Amount, ShouldEqual, "20.00")
		So(paymentResourceRest.AvailablePaymentMethods, ShouldResemble, []string{"GovPay"})
		So(paymentResourceRest.CompletedAt, ShouldHaveSameTypeAs, time.Now())
		So(paymentResourceRest.CreatedAt, ShouldHaveSameTypeAs, time.Now())
		So(paymentResourceRest.CreatedBy, ShouldResemble, models.CreatedByRest{
			Email:    "email@companieshouse.gov.uk",
			Forename: "forename",
			ID:       "id",
			Surname:  "surname",
		})
		So(paymentResourceRest.Description, ShouldBeEmpty) // TODO implement this
		So(paymentResourceRest.Links.Resource, ShouldEqual, "http://dummy-url")
		regJourney := regexp.MustCompile("https://payments.companieshouse.gov.uk/payments/(.*)/pay")
		So(regJourney.MatchString(paymentResourceRest.Links.Journey), ShouldEqual, true)
		regSelf := regexp.MustCompile("payments/(.*)")
		So(regSelf.MatchString(paymentResourceRest.Links.Self), ShouldEqual, true)
		So(paymentResourceRest.PaymentMethod, ShouldBeEmpty)
		So(paymentResourceRest.Reference, ShouldEqual, "ref")
		So(paymentResourceRest.Status, ShouldEqual, "pending")
		So(paymentResourceRest.Costs, ShouldResemble, []models.CostResourceRest{defaultCost, defaultCost})
		So(paymentResourceRest.MetaData, ShouldResemble, models.PaymentResourceMetaDataRest{
			ID:                       "",
			RedirectURI:              "",
			State:                    "",
			ExternalPaymentStatusURI: "",
		})

		So(status, ShouldEqual, Success)
		So(err, ShouldBeNil)
	})
}

func TestUnitPatchPaymentSession(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	cfg, _ := config.Get()

	Convey("Error Patching Payment Resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(fmt.Errorf("error"))

		resource := models.PaymentResourceRest{}
		err := mockPaymentService.PatchPaymentSession("1234", resource)
		So(err.Error(), ShouldEqual, "error patching payment session on database: [error]")
	})

	Convey("Successful Patch Payment Resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().PatchPaymentResource("1234", gomock.Any()).Return(nil)

		resource := models.PaymentResourceRest{
			PaymentMethod: "GovPay",
			Status:        "status",
		}

		err := mockPaymentService.PatchPaymentSession("1234", resource)
		So(err, ShouldBeNil)

	})
}

func TestUnitGetPayment(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	cfg, _ := config.Get()
	defer resetConfig()

	Convey("Error getting payment from DB", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{}, fmt.Errorf("error"))

		req := httptest.NewRequest("Get", "/test", nil)

		paymentResourceRest, status, err := mockPaymentService.GetPaymentSession(req, "1234")
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error getting payment resource from db: [error]")
	})

	Convey("Payment ID not found", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource("invalid").Return(nil, nil)

		req := httptest.NewRequest("Get", "/test", nil)

		paymentResourceRest, status, err := mockPaymentService.GetPaymentSession(req, "invalid")
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, NotFound)
		So(err, ShouldBeNil)
	})

	Convey("Error getting payment resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource("1234").Return(&models.PaymentResourceDB{}, nil)

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "http://dummy-resource", nil)

		paymentResourceRest, status, err := mockPaymentService.GetPaymentSession(req, "1234")
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error getting payment resource: [error getting Cost Resource: [Get : no responder found]]")
	})

	cfg.DomainWhitelist = "http://dummy-resource"

	Convey("Invalid cost", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "x", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		costArray[0].Amount = "x"
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		paymentResourceRest, status, err := mockPaymentService.GetPaymentSession(req, "1234")
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error getting amount from costs: [amount [x] format incorrect]")
	})

	Convey("Amount mismatch", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "100", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		costArray[0].Amount = "99"
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		paymentResourceRest, status, err := mockPaymentService.GetPaymentSession(req, "1234")
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Forbidden)
		So(err.Error(), ShouldEqual, "amount in payment resource [99.00] different from db [100] for id [1234]")
	})

	Convey("Get Payment session - success - Single cost", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		paymentResourceRest, status, err := mockPaymentService.GetPaymentSession(req, "1234")
		So(paymentResourceRest, ShouldResemble, &models.PaymentResourceRest{
			Amount: "10.00",
			Links: models.PaymentLinksRest{
				Resource: "http://dummy-resource",
			},
			Costs: []models.CostResourceRest{
				{
					Amount:                  "10",
					AvailablePaymentMethods: []string{"GovPay"},
					ClassOfPayment:          []string{"class"},
					Description:             "desc",
					DescriptionIdentifier:   "identifier",
					Links: models.CostLinksRest{
						Self: "self",
					},
				},
			},
			MetaData: models.PaymentResourceMetaDataRest{
				ID: "1234",
			},
		})
		So(status, ShouldEqual, Success)
		So(err, ShouldBeNil)
	})

	Convey("Get Payment session - success - Multiple costs", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "20.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost, defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		paymentResourceRest, status, err := mockPaymentService.GetPaymentSession(req, "1234")
		So(paymentResourceRest, ShouldResemble, &models.PaymentResourceRest{
			Amount: "20.00",
			Links: models.PaymentLinksRest{
				Resource: "http://dummy-resource",
			},
			Costs: []models.CostResourceRest{
				{
					Amount:                  "10",
					AvailablePaymentMethods: []string{"GovPay"},
					ClassOfPayment:          []string{"class"},
					Description:             "desc",
					DescriptionIdentifier:   "identifier",
					Links: models.CostLinksRest{
						Self: "self",
					},
				},
				{
					Amount:                  "10",
					AvailablePaymentMethods: []string{"GovPay"},
					ClassOfPayment:          []string{"class"},
					Description:             "desc",
					DescriptionIdentifier:   "identifier",
					Links: models.CostLinksRest{
						Self: "self",
					},
				},
			},
			MetaData: models.PaymentResourceMetaDataRest{
				ID: "1234",
			},
		})
		So(status, ShouldEqual, Success)
		So(err, ShouldBeNil)
	})
}

func TestUnitGetTotalAmount(t *testing.T) {
	Convey("Get Total Amount - valid", t, func() {
		costs := []models.CostResourceRest{{Amount: "10"}, {Amount: "13"}, {Amount: "13.01"}}
		amount, err := getTotalAmount(&costs)
		So(err, ShouldBeNil)
		So(amount, ShouldEqual, "36.01")
	})
	Convey("Test invalid amounts", t, func() {
		invalidAmounts := []string{"alpha", "12,", "12.", "12,00", "12.012", "a.9", "9.a"}
		for _, amount := range invalidAmounts {
			totalAmount, err := getTotalAmount(&[]models.CostResourceRest{{Amount: amount}})
			So(totalAmount, ShouldEqual, "")
			So(err.Error(), ShouldEqual, fmt.Sprintf("amount [%s] format incorrect", amount))
		}
	})
}

func TestUnitGetCosts(t *testing.T) {
	cfg, _ := config.Get()
	cfg.DomainWhitelist = "http://dummy-resource"
	defer resetConfig()

	Convey("Error getting Cost Resource", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "http://dummy-resource", nil)

		costResourceRest, status, err := getCosts("http://dummy-resource")
		So(costResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error getting Cost Resource: [Get http://dummy-resource: no responder found]")
	})

	Convey("Failure status when getting Cost Resource", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(400, nil)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		costResourceRest, status, err := getCosts("http://dummy-resource")
		So(costResourceRest, ShouldBeNil)
		So(status, ShouldEqual, InvalidData)
		So(err.Error(), ShouldEqual, "error getting Cost Resource")
	})

	Convey("Error reading Cost Resource", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		cost := defaultCost
		cost.Amount = ""
		jsonResponse, _ := httpmock.NewJsonResponder(200, []models.CostResourceRest{cost})
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		costResourceRest, status, err := getCosts("http://dummy-resource")
		So(costResourceRest, ShouldBeNil)
		So(status, ShouldEqual, InvalidData)
		So(err.Error(), ShouldEqual, "Key: 'CostResourceRest.Amount' Error:Field validation for 'Amount' failed on the 'required' tag")
	})
}

func TestUnitGenerateID(t *testing.T) {
	Convey("Valid generated PaymentResource ID", t, func() {
		re := regexp.MustCompile("^[0-9]{20}$")
		So(re.MatchString(generateID()), ShouldEqual, true)
	})
}

func TestUnitValidateIncomingPayment(t *testing.T) {
	cfg, _ := config.Get()
	defer resetConfig()

	Convey("Invalid request", t, func() {
		err := validateIncomingPayment(models.IncomingPaymentResourceRequest{}, cfg)
		So(err.Error(), ShouldEqual, "Key: 'IncomingPaymentResourceRequest.RedirectURI' Error:Field validation for 'RedirectURI' failed on the 'required' tag\nKey: 'IncomingPaymentResourceRequest.Resource' Error:Field validation for 'Resource' failed on the 'required' tag\nKey: 'IncomingPaymentResourceRequest.State' Error:Field validation for 'State' failed on the 'required' tag")
	})

	Convey("Invalid Resource Domain", t, func() {
		request := models.IncomingPaymentResourceRequest{
			Resource:    "http://dummy-resource",
			RedirectURI: "http://www.companieshouse.gov.uk",
			State:       "state",
		}
		err := validateIncomingPayment(request, cfg)
		So(err.Error(), ShouldEqual, "invalid resource domain: http://dummy-resource")
	})

	cfg.DomainWhitelist = "http://dummy-resource"

	Convey("Valid Resource Domain", t, func() {
		request := models.IncomingPaymentResourceRequest{
			Resource:    "http://dummy-resource",
			RedirectURI: "http://dummy-resource",
			State:       "state",
		}
		err := validateIncomingPayment(request, cfg)
		So(err, ShouldBeNil)
	})
}

func TestUnitValidateCosts(t *testing.T) {
	Convey("Invalid Cost", t, func() {
		cost := []models.CostResourceRest{{
			Amount:                  "10",
			AvailablePaymentMethods: []string{"method"},
			ClassOfPayment:          []string{"class"},
			Description:             "",
			DescriptionIdentifier:   "identifier",
			Links:                   models.CostLinksRest{Self: "self"},
		}}
		So(validateCosts(&cost), ShouldNotBeNil)
	})
	Convey("Valid Cost", t, func() {
		cost := []models.CostResourceRest{{
			Amount:                  "10",
			AvailablePaymentMethods: []string{"method"},
			ClassOfPayment:          []string{"class"},
			Description:             "desc",
			DescriptionIdentifier:   "identifier",
			Links:                   models.CostLinksRest{Self: "self"},
		}}
		So(validateCosts(&cost), ShouldBeNil)
	})
	Convey("Multiple Costs", t, func() {
		cost := []models.CostResourceRest{
			{
				Amount:                  "10",
				AvailablePaymentMethods: []string{"method"},
				ClassOfPayment:          []string{"class"},
				Description:             "desc",
				DescriptionIdentifier:   "identifier",
				Links:                   models.CostLinksRest{Self: "self"},
			},
			{
				Amount:                  "20",
				AvailablePaymentMethods: []string{"method"},
				ClassOfPayment:          []string{"class"},
				Description:             "",
				DescriptionIdentifier:   "identifier",
				Links:                   models.CostLinksRest{Self: "self"},
			},
		}
		So(validateCosts(&cost), ShouldNotBeNil)
	})
}

func resetConfig() {
	cfg, _ := config.Get()
	cfg.DomainWhitelist = ""
}
