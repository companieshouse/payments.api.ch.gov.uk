package service

import (
	"context"
	"fmt"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/companieshouse/chs.go/authentication"
	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/dao"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"github.com/golang/mock/gomock"
	"github.com/jarcoal/httpmock"
	. "github.com/smartystreets/goconvey/convey"
)

var defaultCost = models.CostResourceRest{
	Amount:                  "10",
	AvailablePaymentMethods: []string{"credit-card"},
	ClassOfPayment:          []string{"data-maintenance"},
	Description:             "desc",
	DescriptionIdentifier:   "identifier",
	ProductType:             "productType",
}

var defaultCosts = models.CostsRest{
	Description:   "costs_desc",
	Costs:         []models.CostResourceRest{defaultCost},
	CompanyNumber: "companyNumber",
}

var defaultUserDetails = authentication.AuthUserDetails{
	Email:    "email@companieshouse.gov.uk",
	Forename: "forename",
	ID:       "id",
	Surname:  "surname",
}

func createMockPaymentService(mockDAO *dao.MockDAO, cfg *config.Config) PaymentService {
	return PaymentService{
		DAO:    mockDAO,
		Config: *cfg,
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
	cfg.DomainAllowList = "http://dummy-url"
	cfg.ExpiryTimeInMinutes = "90"

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
		mockPaymentService := createMockPaymentService(dao.NewMockDAO(mockCtrl), cfg)
		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", `http://dummy-resource`, nil)

		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}

		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		resource := models.IncomingPaymentResourceRequest{
			RedirectURI: "http://www.companieshouse.gov.uk",
			Resource:    "http://dummy-url",
			State:       "state",
		}

		paymentResourceRest, status, err := mockPaymentService.CreatePaymentSession(req.WithContext(ctx), resource)
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error getting payment resource: [error getting Cost Resource: [Get \"http://dummy-url\": no responder found]]")
	})

	Convey("Error getting total amount from costs", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costResource := defaultCost
		costResource.Amount = "invalid_amount"
		costs := models.CostsRest{
			Costs: []models.CostResourceRest{costResource},
		}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costs)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

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
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		authUserDetails := authentication.AuthUserDetails{
			ID: "identity",
		}
		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, authUserDetails)

		resource := models.IncomingPaymentResourceRequest{
			RedirectURI: "http://www.companieshouse.gov.uk",
			Resource:    "http://dummy-url",
			State:       "state",
		}
		paymentResourceRest, status, err := mockPaymentService.CreatePaymentSession(req.WithContext(ctx), resource)
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error writing to DB: error")
	})

	cfg.PaymentsWebURL = "https://payments.companieshouse.gov.uk"

	Convey("Valid request - single cost", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().CreatePaymentResource(gomock.Any())

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, defaultUserDetails)

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
		So(paymentResourceRest.AvailablePaymentMethods, ShouldResemble, []string{"credit-card"})
		So(paymentResourceRest.CompletedAt, ShouldHaveSameTypeAs, time.Now())
		So(paymentResourceRest.CreatedAt, ShouldHaveSameTypeAs, time.Now())
		So(paymentResourceRest.CreatedBy, ShouldResemble, models.CreatedByRest{
			Email:    "email@companieshouse.gov.uk",
			Forename: "forename",
			ID:       "id",
			Surname:  "surname",
		})
		So(paymentResourceRest.Description, ShouldEqual, "costs_desc")
		So(paymentResourceRest.Links.Resource, ShouldEqual, "http://dummy-url")
		regJourney := regexp.MustCompile("https://payments.companieshouse.gov.uk/payments/(.*)/pay")
		So(regJourney.MatchString(paymentResourceRest.Links.Journey), ShouldEqual, true)
		regSelf := regexp.MustCompile("payments/(.*)")
		So(regSelf.MatchString(paymentResourceRest.Links.Self), ShouldEqual, true)
		So(paymentResourceRest.PaymentMethod, ShouldBeEmpty)
		So(paymentResourceRest.Reference, ShouldEqual, "ref")
		So(paymentResourceRest.CompanyNumber, ShouldEqual, "companyNumber")
		So(paymentResourceRest.Status, ShouldEqual, "pending")
		So(paymentResourceRest.Costs, ShouldResemble, defaultCosts.Costs)
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
		costs := defaultCosts
		costs.Costs = append(costs.Costs, defaultCost)
		jsonResponse, _ := httpmock.NewJsonResponder(200, costs)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, defaultUserDetails)

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
		So(paymentResourceRest.AvailablePaymentMethods, ShouldResemble, []string{"credit-card"})
		So(paymentResourceRest.CompletedAt, ShouldHaveSameTypeAs, time.Now())
		So(paymentResourceRest.CreatedAt, ShouldHaveSameTypeAs, time.Now())
		So(paymentResourceRest.CreatedBy, ShouldResemble, models.CreatedByRest{
			Email:    "email@companieshouse.gov.uk",
			Forename: "forename",
			ID:       "id",
			Surname:  "surname",
		})
		So(paymentResourceRest.Description, ShouldEqual, "costs_desc")
		So(paymentResourceRest.Links.Resource, ShouldEqual, "http://dummy-url")
		regJourney := regexp.MustCompile("https://payments.companieshouse.gov.uk/payments/(.*)/pay")
		So(regJourney.MatchString(paymentResourceRest.Links.Journey), ShouldEqual, true)
		regSelf := regexp.MustCompile("payments/(.*)")
		So(regSelf.MatchString(paymentResourceRest.Links.Self), ShouldEqual, true)
		So(paymentResourceRest.PaymentMethod, ShouldBeEmpty)
		So(paymentResourceRest.Reference, ShouldEqual, "ref")
		So(paymentResourceRest.CompanyNumber, ShouldEqual, "companyNumber")
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

	Convey("Valid request - API Key", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().CreatePaymentResource(gomock.Any())
		req := httptest.NewRequest("Get", "/test", nil)
		req.Header.Set("ERIC-Identity-Type", authentication.APIKeyIdentityType) // Set API Key auth for this test
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costs := defaultCosts
		costs.Costs = append(costs.Costs, defaultCost)
		jsonResponse, _ := httpmock.NewJsonResponder(200, costs)
		httpmock.RegisterResponder("GET", "http://dummy-url", jsonResponse)

		ctx := context.WithValue(req.Context(), authentication.ContextKeyUserDetails, defaultUserDetails)

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
		So(paymentResourceRest.AvailablePaymentMethods, ShouldResemble, []string{"credit-card"})
		So(paymentResourceRest.CompletedAt, ShouldHaveSameTypeAs, time.Now())
		So(paymentResourceRest.CreatedAt, ShouldHaveSameTypeAs, time.Now())
		So(paymentResourceRest.CreatedBy, ShouldResemble, models.CreatedByRest{
			Email:    "email@companieshouse.gov.uk",
			Forename: "forename",
			ID:       "id",
			Surname:  "surname",
		})
		So(paymentResourceRest.Description, ShouldEqual, "costs_desc")
		So(paymentResourceRest.Links.Resource, ShouldEqual, "http://dummy-url")
		// /api-key must be appended to Journey URL for API Key request
		regJourney := regexp.MustCompile("https://payments.companieshouse.gov.uk/payments/(.*)/pay/api-key")
		So(regJourney.MatchString(paymentResourceRest.Links.Journey), ShouldEqual, true)
		regSelf := regexp.MustCompile("payments/(.*)")
		So(regSelf.MatchString(paymentResourceRest.Links.Self), ShouldEqual, true)
		So(paymentResourceRest.PaymentMethod, ShouldBeEmpty)
		So(paymentResourceRest.Reference, ShouldEqual, "ref")
		So(paymentResourceRest.CompanyNumber, ShouldEqual, "companyNumber")
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
	defer resetConfig()

	Convey("Error Finding Payment Resource From GET Request", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{}, fmt.Errorf("error"))
		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costArray)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		resource := models.PaymentResourceRest{}
		responseType, err := mockPaymentService.PatchPaymentSession(req, "1234", resource)
		So(responseType, ShouldEqual, Error)
		So(err.Error(), ShouldStartWith, "error getting payment resource to patch:")
	})

	cfg.DomainAllowList = "http://dummy-resource"

	Convey("Error Patching Payment Resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().PatchPaymentResource(gomock.Any(), gomock.Any()).Return(fmt.Errorf("error"))
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "10.00", Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)
		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		resource := models.PaymentResourceRest{}
		responseType, err := mockPaymentService.PatchPaymentSession(req, "1234", resource)
		So(responseType, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error patching payment session on database: [error]")
	})

	Convey("Successful Patch Payment Resource", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().PatchPaymentResource("1234", gomock.Any()).Return(nil)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(&models.PaymentResourceDB{ID: "1234", Data: models.PaymentResourceDataDB{Amount: "10.00", Status: Pending.String(), Links: models.PaymentLinksDB{Resource: "http://dummy-resource"}}}, nil)
		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		resource := models.PaymentResourceRest{
			PaymentMethod: "credit-card",
		}

		responseType, err := mockPaymentService.PatchPaymentSession(req, "1234", resource)
		So(responseType, ShouldEqual, Success)
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
		So(err.Error(), ShouldEqual, "error getting payment resource: [error getting Cost Resource: [Get \"\": no responder found]]")
	})

	cfg.DomainAllowList = "http://dummy-resource"

	Convey("Invalid cost", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(
			&models.PaymentResourceDB{
				ID: "1234",
				Data: models.PaymentResourceDataDB{
					Amount: "x",
					Links:  models.PaymentLinksDB{Resource: "http://dummy-resource"},
				},
			},
			nil,
		)

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		costArray[0].Amount = "x"
		costs := defaultCosts
		costs.Costs = costArray
		jsonResponse, _ := httpmock.NewJsonResponder(200, costs)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		paymentResourceRest, status, err := mockPaymentService.GetPaymentSession(req, "1234")
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error getting amount from costs: [amount [x] format incorrect]")
	})

	Convey("Amount mismatch", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(
			&models.PaymentResourceDB{
				ID: "1234",
				Data: models.PaymentResourceDataDB{
					Amount: "100",
					Links:  models.PaymentLinksDB{Resource: "http://dummy-resource"},
				},
			},
			nil,
		)

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costArray := []models.CostResourceRest{defaultCost}
		costArray[0].Amount = "99"
		costs := defaultCosts
		costs.Costs = costArray
		jsonResponse, _ := httpmock.NewJsonResponder(200, costs)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		paymentResourceRest, status, err := mockPaymentService.GetPaymentSession(req, "1234")
		So(paymentResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Forbidden)
		So(err.Error(), ShouldEqual, "amount in payment resource [99.00] different from db [100] for id [1234]")
	})

	Convey("Get Payment session - success - Single cost", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(
			&models.PaymentResourceDB{
				ID: "1234",
				Data: models.PaymentResourceDataDB{
					Amount: "10.00",
					Links:  models.PaymentLinksDB{Resource: "http://dummy-resource"},
				},
			},
			nil,
		)

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(200, defaultCosts)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		paymentResourceRest, status, err := mockPaymentService.GetPaymentSession(req, "1234")
		So(paymentResourceRest, ShouldResemble, &models.PaymentResourceRest{
			Amount:      "10.00",
			Description: "costs_desc",
			Links: models.PaymentLinksRest{
				Resource: "http://dummy-resource",
			},
			Costs: []models.CostResourceRest{
				{
					Amount:                  "10",
					AvailablePaymentMethods: []string{"credit-card"},
					ClassOfPayment:          []string{"data-maintenance"},
					Description:             "desc",
					DescriptionIdentifier:   "identifier",
					ProductType:             "productType",
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
		mock.EXPECT().GetPaymentResource(gomock.Any()).Return(
			&models.PaymentResourceDB{
				ID: "1234",
				Data: models.PaymentResourceDataDB{
					Amount: "20.00",
					Links:  models.PaymentLinksDB{Resource: "http://dummy-resource"},
				},
			},
			nil,
		)

		req := httptest.NewRequest("Get", "/test", nil)

		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		costs := defaultCosts
		costs.Costs = append(costs.Costs, defaultCost)
		jsonResponse, _ := httpmock.NewJsonResponder(200, costs)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		paymentResourceRest, status, err := mockPaymentService.GetPaymentSession(req, "1234")
		So(paymentResourceRest, ShouldResemble, &models.PaymentResourceRest{
			Amount:      "20.00",
			Description: "costs_desc",
			Links: models.PaymentLinksRest{
				Resource: "http://dummy-resource",
			},
			Costs: []models.CostResourceRest{
				{
					Amount:                  "10",
					AvailablePaymentMethods: []string{"credit-card"},
					ClassOfPayment:          []string{"data-maintenance"},
					Description:             "desc",
					DescriptionIdentifier:   "identifier",
					ProductType:             "productType",
				},
				{
					Amount:                  "10",
					AvailablePaymentMethods: []string{"credit-card"},
					ClassOfPayment:          []string{"data-maintenance"},
					Description:             "desc",
					DescriptionIdentifier:   "identifier",
					ProductType:             "productType",
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

func TestUnitGetIncompletePayments(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	cfg, _ := config.Get()

	Convey("Error getting incomplete payments from DB", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)
		mock.EXPECT().GetIncompleteGovPayPayments(gomock.Any()).Return(nil, fmt.Errorf("err"))

		payments, err := mockPaymentService.GetIncompletePayments(cfg)
		So(payments, ShouldBeNil)
		So(err.Error(), ShouldEqual, "err")
	})

	Convey("Get incomplete payments - success", t, func() {
		mock := dao.NewMockDAO(mockCtrl)
		mockPaymentService := createMockPaymentService(mock, cfg)

		mock.EXPECT().GetIncompleteGovPayPayments(gomock.Any()).Return([]models.PaymentResourceDB{{ID: "id"}}, nil)

		payments, err := mockPaymentService.GetIncompletePayments(cfg)
		So(len(*payments), ShouldEqual, 1)
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
	cfg.DomainAllowList = "http://dummy-resource"
	cfg.SecureAppCostsRegex = "\\/secure-app-regex-test\\/"
	r := regexp.MustCompile(cfg.SecureAppCostsRegex)
	defer resetConfig()

	Convey("Error getting Cost Resource", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterResponder("GET", "http://dummy-resource", nil)

		costResourceRest, status, err := getCosts("http://dummy-resource", cfg, r)
		So(costResourceRest, ShouldBeNil)
		So(status, ShouldEqual, Error)
		So(err.Error(), ShouldEqual, "error getting Cost Resource: [Get \"http://dummy-resource\": no responder found]")
	})

	Convey("Failure status when getting Cost Resource", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(400, nil)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		costResourceRest, status, err := getCosts("http://dummy-resource", cfg, r)
		So(costResourceRest, ShouldBeNil)
		So(status, ShouldEqual, InvalidData)
		So(err.Error(), ShouldEqual, "error getting Cost Resource - status code: [400]")
	})

	Convey("Error reading Cost Resource", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		cost := defaultCost
		cost.Amount = ""
		costs := models.CostsRest{
			Costs: []models.CostResourceRest{cost},
		}
		jsonResponse, _ := httpmock.NewJsonResponder(200, costs)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		costResourceRest, status, err := getCosts("http://dummy-resource", cfg, r)
		So(costResourceRest, ShouldBeNil)
		So(status, ShouldEqual, InvalidData)
		So(err.Error(), ShouldEqual, "Key: 'CostResourceRest.Amount' Error:Field validation for 'Amount' failed on the 'required' tag")
	})

	Convey("CostsNotFound status when getting Cost Resource", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(404, nil)
		httpmock.RegisterResponder("GET", "http://dummy-resource", jsonResponse)

		costResourceRest, status, err := getCosts("http://dummy-resource", cfg, r)
		So(costResourceRest, ShouldBeNil)
		So(status, ShouldEqual, CostsNotFound)
		So(err.Error(), ShouldEqual, "error getting Cost Resource - Not Found: [404]")
	})

	Convey("CostsGone status when getting Cost Resource", t, func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()
		jsonResponse, _ := httpmock.NewJsonResponder(404, nil)
		httpmock.RegisterResponder("GET", "http://dummy-resource/secure-app-regex-test/123456789abc/payment", jsonResponse)

		costResourceRest, status, err := getCosts("http://dummy-resource/secure-app-regex-test/123456789abc/payment", cfg, r)
		So(costResourceRest, ShouldBeNil)
		So(status, ShouldEqual, CostsGone)
		So(err.Error(), ShouldEqual, "error getting Cost Resource - Gone: [410]")
	})
}

// TestUnitGenerateIDForDuplicates this will test the generateID func's capability for generating unique id's
// if a duplicate is generated, this test will fail.
func TestUnitGenerateIDForDuplicates(t *testing.T) {
	// generate 100,000 id's
	times := 100000 // 100 thousand
	generated := make([]string, times)

	for i := 0; i < times; i++ {
		ref := generateID()
		generated[i] = ref
	}

	// check for dups by creating a map of string->int and counting the the entry values whilst
	// iterating through the generated map
	generatedCheck := make(map[string]int)
	var duplicates []string
	for _, reference := range generated {
		_, exists := generatedCheck[reference]
		if exists {
			duplicates = append(duplicates, reference)
		} else {
			generatedCheck[reference] = 1
		}
	}

	if len(duplicates) != 0 {
		t.Errorf("%d duplicate id's generated", len(duplicates))
		t.Fail()
	}
}

func TestUnitGenerateID(t *testing.T) {
	Convey("generates a id with a length of 15", t, func() {
		So(generateID(), ShouldHaveLength, 15)
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

	cfg.DomainAllowList = "http://dummy-resource"

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
			ProductType:             "productType",
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
			},
			{
				Amount:                  "20",
				AvailablePaymentMethods: []string{"method"},
				ClassOfPayment:          []string{"class"},
				Description:             "",
				DescriptionIdentifier:   "identifier",
			},
		}
		So(validateCosts(&cost), ShouldNotBeNil)
	})
}

func TestUnitIsExpired(t *testing.T) {
	cfg, _ := config.Get()
	cfg.ExpiryTimeInMinutes = "90"
	defer resetConfig()

	Convey("Expired Session", t, func() {
		paymentResourceRest := models.PaymentResourceRest{CreatedAt: time.Now().Add(time.Hour * -2)}
		expired, err := IsExpired(paymentResourceRest, cfg)
		So(expired, ShouldEqual, true)
		So(err, ShouldEqual, nil)
	})

	Convey("Unexpired Session", t, func() {
		paymentResourceRest := models.PaymentResourceRest{CreatedAt: time.Now()}
		expired, err := IsExpired(paymentResourceRest, cfg)
		So(expired, ShouldEqual, false)
		So(err, ShouldEqual, nil)
	})
}

func resetConfig() {
	cfg, _ := config.Get()
	cfg.DomainAllowList = ""
}
