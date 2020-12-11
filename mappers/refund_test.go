package mappers

import (
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestUnitMapGovPayToRefundResponse(t *testing.T) {

	Convey("Maps successfully to refund response", t, func() {
		govPayResponse := models.CreateRefundGovPayResponse{
			RefundId:    "123",
			CreatedDate: "321",
			Amount:      400,
			Links: models.GovPayRefundLinks{
				Self: models.Self{
					HREF:   "asd",
					Method: "dsa",
				},
				Payment: models.Payment{
					HREF:   "qwe",
					Method: "ewq",
				},
			},
			Status: "success",
		}

		refundResponse := MapGovPayToRefundResponse(govPayResponse)

		So(refundResponse.RefundId, ShouldEqual, govPayResponse.RefundId)
		So(refundResponse.Amount, ShouldEqual, govPayResponse.Amount)
		So(refundResponse.Status, ShouldEqual, govPayResponse.Status)
		So(refundResponse.CreatedDateTime, ShouldEqual, govPayResponse.CreatedDate)
	})
}

func TestUnitMapToRefundRest(t *testing.T) {
	Convey("Maps successfully to refund rest", t, func() {

		govPayResponse := models.CreateRefundGovPayResponse{
			RefundId:    "123",
			CreatedDate: "321",
			Amount:      400,
			Links: models.GovPayRefundLinks{
				Self: models.Self{
					HREF:   "asd",
					Method: "dsa",
				},
				Payment: models.Payment{
					HREF:   "qwe",
					Method: "ewq",
				},
			},
			Status: "success",
		}

		refundRest := MapToRefundRest(govPayResponse)

		So(refundRest.RefundId, ShouldEqual, govPayResponse.RefundId)
		So(refundRest.Amount, ShouldEqual, govPayResponse.Amount)
		So(refundRest.CreatedAt, ShouldEqual, govPayResponse.CreatedDate)
		So(refundRest.Status, ShouldEqual, govPayResponse.Status)
		So(refundRest.ExternalRefundUrl, ShouldEqual, govPayResponse.Links.Self.HREF)
	})
}

func TestUnitMapRefundToRefundResponse(t *testing.T) {

	Convey("Maps successfully to refund response", t, func() {
		refund := models.RefundResourceRest{
			RefundId:  "123",
			CreatedAt: "321",
			Amount:    400,
			Status:    "success",
		}

		refundResponse := MapRefundToRefundResponse(refund)

		So(refundResponse.RefundId, ShouldEqual, refund.RefundId)
		So(refundResponse.Amount, ShouldEqual, refund.Amount)
		So(refundResponse.Status, ShouldEqual, refund.Status)
		So(refundResponse.CreatedDateTime, ShouldEqual, refund.CreatedAt)
	})
}
