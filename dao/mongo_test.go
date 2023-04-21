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
				Links:         models.PaymentLinksDB{Refunds: "refunds_url"},
			},
			ExternalPaymentStatusURI:     "companieshouse.gov.uk",
			ExternalPaymentStatusID:      "id123",
			ExternalPaymentTransactionID: "id456",
			Refunds:                      []models.RefundResourceDB{},
			BulkRefund:                   []models.BulkRefundDB{{}},
		}
		err := dao.PatchPaymentResource("id123", &resource)
		So(err.Error(), ShouldEqual, "the Update operation must have a Deployment set before Execute can be called")
	})
}

func TestUnitGetPaymentResourceByExternalPaymentStatusID(t *testing.T) {
	Convey("Get payment resource by external ID", t, func() {
		cfg, _ := config.Get()
		client = &mongo.Client{}
		dao := NewDAO(cfg)

		resource, err := dao.GetPaymentResourceByProviderID("id123")
		So(resource, ShouldBeNil)
		So(err.Error(), ShouldEqual, "the Find operation must have a Deployment set before Execute can be called")
	})
}

func TestUnitGetPaymentsWithRefundStatus(t *testing.T) {
	Convey("Get payment with refund status", t, func() {
		cfg, _ := config.Get()
		client = &mongo.Client{}
		dao := NewDAO(cfg)

		_, err := dao.GetPaymentsWithRefundStatus()
		So(err.Error(), ShouldEqual, "the Find operation must have a Deployment set before Execute can be called")
	})
}

func TestUnitGetPaymentsWithRefundPendingStatus(t *testing.T) {
	Convey("Get payment with paid status", t, func() {
		cfg, _ := config.Get()
		client = &mongo.Client{}
		dao := NewDAO(cfg)

		_, err := dao.GetPaymentsWithRefundPendingStatus()
		So(err.Error(), ShouldEqual, "the Find operation must have a Deployment set before Execute can be called")
	})
}

func TestUnitPatchPaymentsWithRefundPendingStatus(t *testing.T) {
	Convey("Patch Payment Resource", t, func() {
		cfg, _ := config.Get()
		client = &mongo.Client{}
		dao := NewDAO(cfg)

		refundData := models.RefundResourceDB{
			RefundId:          "sasaswewq23wsw",
			CreatedAt:         "2020-11-19T12:57:30.Z06Z",
			Amount:            800.0,
			Status:            "pending",
			ExternalRefundUrl: "https://pulicapi.payments.service.gov.uk",
		}
		refundDatas := []models.RefundResourceDB{refundData}

		resource := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				PaymentMethod: "credit-card",
				Status:        "pending",
				CompletedAt:   time.Now(),
				ProviderID:    "id123",
			},
			ExternalPaymentStatusURI:     "companieshouse.gov.uk",
			ExternalPaymentStatusID:      "id123",
			ExternalPaymentTransactionID: "id456",
			Refunds:                      refundDatas,
		}
		_, err := dao.PatchRefundSuccessStatus("id123", true, &resource)
		So(err.Error(), ShouldEqual, "the FindAndModify operation must have a Deployment set before Execute can be called")
	})
}

func TestUnitGetPaymentRefunds(t *testing.T) {
	Convey("Get payment refunds by payment Id", t, func() {
		cfg, _ := config.Get()
		client = &mongo.Client{}
		dao := NewDAO(cfg)

		resource, err := dao.GetPaymentRefunds("id123")
		So(resource, ShouldBeNil)
		So(err.Error(), ShouldEqual, "the Find operation must have a Deployment set before Execute can be called")
	})
}

func TestUnitIncrementRefundAttempts(t *testing.T) {
	Convey("Increment refund attempts", t, func() {
		cfg, _ := config.Get()
		client = &mongo.Client{}
		dao := NewDAO(cfg)

		refundData := models.RefundResourceDB{
			RefundId:          "sasaswewq23wsw",
			CreatedAt:         "2020-11-19T12:57:30.Z06Z",
			Amount:            800.0,
			Status:            "pending",
			ExternalRefundUrl: "https://pulicapi.payments.service.gov.uk",
		}
		refundDatas := []models.RefundResourceDB{refundData}

		resource := models.PaymentResourceDB{
			Data: models.PaymentResourceDataDB{
				PaymentMethod: "credit-card",
				Status:        "pending",
				CompletedAt:   time.Now(),
				ProviderID:    "id123",
			},
			ExternalPaymentStatusURI:     "companieshouse.gov.uk",
			ExternalPaymentStatusID:      "id123",
			ExternalPaymentTransactionID: "id456",
			Refunds:                      refundDatas,
		}
		err := dao.IncrementRefundAttempts("id123", &resource)
		So(err.Error(), ShouldEqual, "the FindAndModify operation must have a Deployment set before Execute can be called")
	})
}
