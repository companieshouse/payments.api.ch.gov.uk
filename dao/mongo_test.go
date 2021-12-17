package dao

import (
	"testing"
	"time"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"go.mongodb.org/mongo-driver/mongo"

	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitCreatePaymentResource(t *testing.T) {
	Convey("Create Payment Resource", t, func() {
		cfg, _ := config.Get()
		client = &mongo.Client{}
		dao := NewDAO(cfg)

		resource := models.PaymentResourceDB{}
		err := dao.CreatePaymentResource(&resource)
		So(err.Error(), ShouldEqual, "the Insert operation must have a Deployment set before Execute can be called")
	})
}

func TestUnitGetPaymentResource(t *testing.T) {
	Convey("Get Payment Resource", t, func() {
		cfg, _ := config.Get()
		client = &mongo.Client{}
		dao := NewDAO(cfg)

		resource, err := dao.GetPaymentResource("id123")
		So(resource, ShouldBeNil)
		So(err.Error(), ShouldEqual, "the Find operation must have a Deployment set before Execute can be called")
	})
}

func TestUnitPatchPaymentResource(t *testing.T) {
	Convey("Patch Payment Resource", t, func() {
		cfg, _ := config.Get()
		client = &mongo.Client{}
		dao := NewDAO(cfg)

		resource := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				PaymentMethod: "credit-card",
				Status:        "pending",
				CompletedAt:   time.Now(),
				ProviderID:    "id123",
			},
			ExternalPaymentStatusURI: "companieshouse.gov.uk",
			ExternalPaymentStatusID:  "id123",
			Refunds:                  []models.RefundResourceDB{},
		}
		err := dao.PatchPaymentResource("id123", &resource)
		So(err.Error(), ShouldEqual, "the Update operation must have a Deployment set before Execute can be called")
	})
}
