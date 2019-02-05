package transformers

import (
	"testing"
	"time"

	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitTransformToDB(t *testing.T) {
	Convey("Rest converted to DB", t, func() {
		now := time.Now()
		paymentResourceRest := models.PaymentResourceRest{
			Amount:                  "123",
			AvailablePaymentMethods: []string{"pay1", "pay2"},
			CompletedAt:             now,
			CreatedAt:               now,
			CreatedBy: models.CreatedByRest{
				Email:    "created_by@companieshouse.gov.uk",
				Forename: "user_forename",
				ID:       "abc",
				Surname:  "user_surname",
			},
			Description: "payment_description",
			Links: models.PaymentLinksRest{
				Journey:  "links_journey",
				Resource: "links_resource",
				Self:     "links_self",
			},
			PaymentMethod: "method",
			Reference:     "ref",
			Status:        "pending",
			Costs: []models.CostResourceRest{
				{
					Amount:                  "65",
					AvailablePaymentMethods: []string{"method1", "method2"},
					ClassOfPayment:          []string{"class1", "class2"},
					Description:             "desc1",
					DescriptionIdentifier:   "desc_identifier1",
					DescriptionValues:       map[string]string{"val": "val1"},
					Links: models.CostLinksRest{
						Resource: "resource1",
						Self:     "self1",
					},
				},
				{
					Amount:                  "73",
					AvailablePaymentMethods: []string{"method3", "method4"},
					ClassOfPayment:          []string{"class3", "class4"},
					Description:             "desc2",
					DescriptionIdentifier:   "desc_identifier2",
					DescriptionValues:       map[string]string{"val": "val2"},
					Links: models.CostLinksRest{
						Resource: "resource2",
						Self:     "self2",
					},
				},
			},
		}

		expectedPaymentResourceDB := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				Amount:                  "123",
				AvailablePaymentMethods: []string{"pay1", "pay2"},
				CompletedAt:             now,
				CreatedAt:               now,
				CreatedBy: models.CreatedByDB{
					Email:    "created_by@companieshouse.gov.uk",
					Forename: "user_forename",
					ID:       "abc",
					Surname:  "user_surname",
				},
				Description: "payment_description",
				Links: models.PaymentLinksDB{
					Journey:  "links_journey",
					Resource: "links_resource",
					Self:     "links_self",
				},
				PaymentMethod: "method",
				Reference:     "ref",
				Status:        "pending",
				Costs: []models.CostResourceDB{
					{
						Amount:                  "65",
						AvailablePaymentMethods: []string{"method1", "method2"},
						ClassOfPayment:          []string{"class1", "class2"},
						Description:             "desc1",
						DescriptionIdentifier:   "desc_identifier1",
						DescriptionValues:       map[string]string{"val": "val1"},
						Links: models.CostLinksDB{
							Resource: "resource1",
							Self:     "self1",
						},
					},
					{
						Amount:                  "73",
						AvailablePaymentMethods: []string{"method3", "method4"},
						ClassOfPayment:          []string{"class3", "class4"},
						Description:             "desc2",
						DescriptionIdentifier:   "desc_identifier2",
						DescriptionValues:       map[string]string{"val": "val2"},
						Links: models.CostLinksDB{
							Resource: "resource2",
							Self:     "self2",
						},
					},
				},
			},
		}
		paymentResourceDB := PaymentTransformer{}.TransformToDB(paymentResourceRest)
		So(paymentResourceDB, ShouldResemble, expectedPaymentResourceDB)
	})
}

func TestUnitTransformToRest(t *testing.T) {
	Convey("DB converted to Rest", t, func() {
		now := time.Now()
		paymentResourceDataDB := models.PaymentResourceDataDB{
			Amount:                  "123",
			AvailablePaymentMethods: []string{"pay1", "pay2"},
			CompletedAt:             now,
			CreatedAt:               now,
			CreatedBy: models.CreatedByDB{
				Email:    "created_by@companieshouse.gov.uk",
				Forename: "user_forename",
				ID:       "abc",
				Surname:  "user_surname",
			},
			Description: "payment_description",
			Links: models.PaymentLinksDB{
				Journey:  "links_journey",
				Resource: "links_resource",
				Self:     "links_self",
			},
			PaymentMethod: "method",
			Reference:     "ref",
			Status:        "pending",
			Costs: []models.CostResourceDB{
				{
					Amount:                  "65",
					AvailablePaymentMethods: []string{"method1", "method2"},
					ClassOfPayment:          []string{"class1", "class2"},
					Description:             "desc1",
					DescriptionIdentifier:   "desc_identifier1",
					DescriptionValues:       map[string]string{"val": "val1"},
					Links: models.CostLinksDB{
						Resource: "resource1",
						Self:     "self1",
					},
				},
				{
					Amount:                  "73",
					AvailablePaymentMethods: []string{"method3", "method4"},
					ClassOfPayment:          []string{"class3", "class4"},
					Description:             "desc2",
					DescriptionIdentifier:   "desc_identifier2",
					DescriptionValues:       map[string]string{"val": "val2"},
					Links: models.CostLinksDB{
						Resource: "resource2",
						Self:     "self2",
					},
				},
			},
		}
		expectedPaymentResourceRest := models.PaymentResourceRest{
			Amount:                  "123",
			AvailablePaymentMethods: []string{"pay1", "pay2"},
			CompletedAt:             now,
			CreatedAt:               now,
			CreatedBy: models.CreatedByRest{
				Email:    "created_by@companieshouse.gov.uk",
				Forename: "user_forename",
				ID:       "abc",
				Surname:  "user_surname",
			},
			Description: "payment_description",
			Links: models.PaymentLinksRest{
				Journey:  "links_journey",
				Resource: "links_resource",
				Self:     "links_self",
			},
			PaymentMethod: "method",
			Reference:     "ref",
			Status:        "pending",
			Costs: []models.CostResourceRest{
				{
					Amount:                  "65",
					AvailablePaymentMethods: []string{"method1", "method2"},
					ClassOfPayment:          []string{"class1", "class2"},
					Description:             "desc1",
					DescriptionIdentifier:   "desc_identifier1",
					DescriptionValues:       map[string]string{"val": "val1"},
					Links: models.CostLinksRest{
						Resource: "resource1",
						Self:     "self1",
					},
				},
				{
					Amount:                  "73",
					AvailablePaymentMethods: []string{"method3", "method4"},
					ClassOfPayment:          []string{"class3", "class4"},
					Description:             "desc2",
					DescriptionIdentifier:   "desc_identifier2",
					DescriptionValues:       map[string]string{"val": "val2"},
					Links: models.CostLinksRest{
						Resource: "resource2",
						Self:     "self2",
					},
				},
			},
		}

		paymentResourceRest := PaymentTransformer{}.TransformToRest(paymentResourceDataDB)
		So(paymentResourceRest, ShouldResemble, expectedPaymentResourceRest)
	})
}
